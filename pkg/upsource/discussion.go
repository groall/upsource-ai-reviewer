package upsource

import (
	"context"
	"fmt"
	"strings"

	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

type CreateDiscussionRequest struct {
	Review  *Review
	Comment string
	File    string
	Line    int
}

const markdownMarkupType = "markdown"

// CreateDiscussion creates a discussion for a given review, file, and line.
func CreateDiscussion(ctx context.Context, upsourceClient *client.Client, config *config.Config, req CreateDiscussionRequest) error {
	if req.File == "" { // No file specified, create a general discussion
		_, err := upsourceClient.CreateDiscussion(ctx, client.CreateDiscussionRequestDTO{
			Anchor:     client.AnchorDTO{},
			ReviewID:   &req.Review.review.ReviewID,
			Text:       req.Comment,
			ProjectID:  req.Review.review.ReviewID.ProjectID,
			MarkupType: markdownMarkupType,
			Labels:     []client.LabelDTO{{Name: config.Upsource.ReviewedLabel}},
		})

		return err
	}

	for _, fileDiffSummary := range req.Review.filesDiffSummary {
		if fileDiffSummary.File.FileName != "/"+req.File && fileDiffSummary.File.FileName != req.File {
			continue // Skip files that are not the specified one
		}

		anchor, err := createAnchorForLine(ctx, upsourceClient, fileDiffSummary, req.Line)
		if err != nil {
			return fmt.Errorf("Error creating anchor for line %d in file %s: %v\n", req.Line, req.File, err)
		}

		_, err = upsourceClient.CreateDiscussion(ctx, client.CreateDiscussionRequestDTO{
			Anchor:     *anchor,
			ReviewID:   &req.Review.review.ReviewID,
			Text:       req.Comment,
			ProjectID:  req.Review.review.ReviewID.ProjectID,
			MarkupType: markdownMarkupType,
			Labels:     []client.LabelDTO{{Name: config.Upsource.ReviewedLabel}},
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
		return nil, fmt.Errorf("Error getting file content for %s: %v\n", fileDiffSummary.File.FileName, err)
	}

	text := fileContent.FileContent.Text
	startOffset, endOffset, err := findRangeInFileContent(text, line)
	if err != nil {
		return nil, fmt.Errorf("Error finding range for line %d in file %s: %v\n", line, fileDiffSummary.File.FileName, err)
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
		offset += len(lines[i]) + 1 // +1 for the '\n'
	}

	startOffset := int32(offset)
	endOffset := startOffset + int32(len(lineContent))

	return startOffset, endOffset, nil
}
