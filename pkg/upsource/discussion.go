package upsource

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/groall/upsource-go-client/client"
)

// ListReviewDiscussions returns the discussions belonging to a specific review.
// Falls back to client-side filtering by ReviewID since the project-scoped query
// is the only listing endpoint exposed by the Upsource API.
func ListReviewDiscussions(ctx context.Context, upsourceClient *client.Client, review *Review) ([]client.DiscussionInFileDTO, error) {
	resp, err := upsourceClient.GetProjectDiscussions(ctx, client.DiscussionsInProjectRequestDTO{
		ProjectID: review.GetProjectID(),
		Query:     fmt.Sprintf("review: %s", review.GetReviewID().ReviewID),
		Limit:     10000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get project discussions: %w", err)
	}

	want := review.GetReviewID().ReviewID
	out := make([]client.DiscussionInFileDTO, 0, len(resp.Discussions))
	for _, d := range resp.Discussions {
		if d.Review == nil || d.Review.ReviewID.ReviewID != want {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

// AddDiscussionComment posts a reply comment to an existing discussion.
func AddDiscussionComment(ctx context.Context, upsourceClient *client.Client, projectID, discussionID, parentCommentID, text string) error {
	_, err := upsourceClient.AddComment(ctx, client.AddCommentRequestDTO{
		ProjectID:    projectID,
		DiscussionID: discussionID,
		ParentID:     parentCommentID,
		Text:         text,
		MarkupType:   markdownMarkupType,
	})
	return err
}

// ResolveDiscussion marks the given discussion as resolved.
func ResolveDiscussion(ctx context.Context, upsourceClient *client.Client, projectID, discussionID string) error {
	_, err := upsourceClient.ResolveDiscussion(ctx, client.ResolveDiscussionRequestDTO{
		ProjectID:    projectID,
		DiscussionID: discussionID,
		IsResolved:   true,
	})
	return err
}

// ShouldReplyToDiscussion is the "should the bot reply now?" predicate.
// Returns the last comment (parent target for the reply) and true when:
//   - discussion carries reviewedLabel
//   - is not resolved
//   - has at least one comment
//   - the most recent comment was not authored by the bot
//   - the bot has authored fewer than maxPerThread comments in this thread
func ShouldReplyToDiscussion(d client.DiscussionInFileDTO, reviewedLabel, botUserID string, maxPerThread int) (client.CommentDTO, bool) {
	var zero client.CommentDTO

	var hasLabel bool
	for _, l := range d.Labels {
		if l.Name == reviewedLabel {
			hasLabel = true
			break
		}
	}
	if !hasLabel {
		return zero, false
	}

	if d.IsResolved != nil && *d.IsResolved {
		return zero, false
	}

	if len(d.Comments) == 0 {
		return zero, false
	}

	last := d.Comments[len(d.Comments)-1]
	if last.AuthorID == botUserID {
		return zero, false
	}

	var botCount int
	for _, c := range d.Comments {
		if c.AuthorID == botUserID {
			botCount++
		}
	}
	if maxPerThread > 0 && botCount >= maxPerThread {
		return zero, false
	}

	return last, true
}

type CreateDiscussionRequest struct {
	Review  *Review
	Comment string
	File    string
	Line    int
}

const markdownMarkupType = "markdown"

// CreateDiscussion creates a discussion for a given review, file, and line.
func CreateDiscussion(ctx context.Context, upsourceClient *client.Client, reviewedLabel string, req CreateDiscussionRequest) error {
	if req.File == "" { // No file specified, create a general discussion
		_, err := upsourceClient.CreateDiscussion(ctx, client.CreateDiscussionRequestDTO{
			Anchor:     client.AnchorDTO{},
			ReviewID:   &req.Review.review.ReviewID,
			Text:       req.Comment,
			ProjectID:  req.Review.review.ReviewID.ProjectID,
			MarkupType: markdownMarkupType,
			Labels:     []client.LabelDTO{{Name: reviewedLabel}},
		})

		return err
	}

	for _, fileDiffSummary := range req.Review.filesDiffSummary {
		if fileDiffSummary.File.FileName != "/"+req.File && fileDiffSummary.File.FileName != req.File {
			continue // Skip files that are not the specified one
		}

		anchor, err := createAnchorForLine(ctx, upsourceClient, fileDiffSummary, req.Line)
		if err != nil {
			return fmt.Errorf("error creating anchor for line %d in file %s: %v", req.Line, req.File, err)
		}

		_, err = upsourceClient.CreateDiscussion(ctx, client.CreateDiscussionRequestDTO{
			Anchor:     *anchor,
			ReviewID:   &req.Review.review.ReviewID,
			Text:       req.Comment,
			ProjectID:  req.Review.review.ReviewID.ProjectID,
			MarkupType: markdownMarkupType,
			Labels:     []client.LabelDTO{{Name: reviewedLabel}},
		})

		return err
	}

	return fmt.Errorf("file %s not found in review %s", req.File, req.Review.review.Title)
}

// createAnchorForLine creates an anchor for a given line in a file.
func createAnchorForLine(ctx context.Context, upsourceClient *client.Client, fileDiffSummary client.FileDiffSummaryDTO, line int) (*client.AnchorDTO, error) {
	fileContent, err := upsourceClient.GetFileContent(ctx, client.FileInRevisionDTO{
		ProjectID:  fileDiffSummary.File.ProjectID,
		RevisionID: fileDiffSummary.File.RevisionID,
		FileName:   fileDiffSummary.File.FileName,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting file content for %s: %v", fileDiffSummary.File.FileName, err)
	}

	text := fileContent.FileContent.Text
	startOffset, endOffset, err := findRangeInFileContent(text, line)
	if err != nil {
		return nil, fmt.Errorf("error finding range for line %d in file %s: %v", line, fileDiffSummary.File.FileName, err)
	}

	return &client.AnchorDTO{
		RevisionID: fileDiffSummary.File.RevisionID,
		FileID:     fileDiffSummary.File.FileName,
		Range: &client.RangeDTO{
			StartOffset: startOffset,
			EndOffset:   endOffset,
		},
	}, nil
}

// findRangeInFileContent finds the start and end offsets for a given line in a file content.
func findRangeInFileContent(fileContent string, line int) (int32, int32, error) {
	if line <= 0 {
		return 0, 0, fmt.Errorf("line number must be positive: %d", line)
	}

	lines := strings.Split(fileContent, "\n")
	if line > len(lines) {
		return 0, 0, fmt.Errorf("line %d does not exist in a file with %d lines", line, len(lines))
	}

	lineIndex := line - 1
	lineContent := lines[lineIndex]
	if len(lineContent) == 0 {
		return 0, 0, fmt.Errorf("line %d is empty", line)
	}

	var offset int
	for i := 0; i < lineIndex; i++ {
		offset += utf8.RuneCount([]byte(lines[i])) + 1 // +1 for the '\n'
	}

	startOffset := int32(offset)
	endOffset := startOffset + int32(utf8.RuneCount([]byte(lineContent)))

	return startOffset, endOffset, nil
}
