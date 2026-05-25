package review

import (
	"context"
	"fmt"
	"log"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/internal/llm"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

type replier struct {
	upsourceClient *client.Client
	config         *config.Config
	ctx            context.Context
	llmReviewer    *llm.Reviewer
	gitProvider    git.Provider
	botUserID      string
}

func newReplier(ctx context.Context, config *config.Config, upsourceClient *client.Client, gitProvider git.Provider, llmReviewer *llm.Reviewer) (*replier, error) {
	replier := &replier{
		config:         config,
		ctx:            ctx,
		upsourceClient: upsourceClient,
		gitProvider:    gitProvider,
		llmReviewer:    llmReviewer,
	}

	return replier, nil
}

// replyToOpenThreads scans reviews the bot has already engaged with and posts a
// follow-up reply in any thread where a human spoke after the bot's last word.
// Errors are logged per discussion / per review; a single failure never aborts the loop.
func (r *replier) replyToOpenThreads() error {
	if !r.config.Replies.Enabled {
		return nil
	}

	botUserID, err := r.resolveBotUserID()
	if err != nil {
		return fmt.Errorf("failed to resolve bot user id: %w", err)
	}

	reviews, err := upsource.ListReviewedReviews(r.ctx, r.upsourceClient, r.config.Upsource.Query, r.config.Upsource.ReviewedLabel)
	if err != nil {
		return fmt.Errorf("failed to list reviewed reviews: %w", err)
	}

	log.Printf("Reply pass: scanning %d already-reviewed reviews\n", len(reviews))

	for _, review := range reviews {
		if err := r.replyInReview(review, botUserID); err != nil {
			log.Printf("Reply pass error in review %s: %v\n", review.GetBranch(), err)
		}
	}

	return nil
}

func (r *replier) replyInReview(review *upsource.Review, botUserID string) error {
	discussions, err := upsource.ListReviewDiscussions(r.ctx, r.upsourceClient, review)
	if err != nil {
		return fmt.Errorf("list discussions: %w", err)
	}
	if len(discussions) == 0 {
		return nil
	}

	var codeContext string
	for _, d := range discussions {
		last, ok := upsource.ShouldReplyToDiscussion(d, r.config.Upsource.ReviewedLabel, botUserID, r.config.Replies.MaxPerThread)
		if !ok {
			log.Printf("Skipping discussion %s in review %s\n", d.DiscussionID, review.GetBranch())
			continue
		}

		if codeContext == "" {
			diff, _, derr := r.gitProvider.GetReviewChanges(review)
			if derr != nil {
				return fmt.Errorf("get review changes: %w", derr)
			}
			codeContext = diff
		}

		thread := buildThreadTranscript(d.Comments, botUserID)
		fileContext := codeContext
		anchorText := buildReplyAnchorText(d.Anchor)

		reply, lerr := r.llmReviewer.Reply(thread, fileContext, anchorText)
		if lerr != nil {
			log.Printf("Failed to get reply for discussion %s: %v\n", d.DiscussionID, lerr)
			continue
		}

		if reply.Comment != "" {
			if err := upsource.AddDiscussionComment(r.ctx, r.upsourceClient, review.GetProjectID(), d.DiscussionID, last.CommentID, reply.Comment); err != nil {
				log.Printf("Failed to post reply for discussion %s: %v\n", d.DiscussionID, err)
				continue
			}
			log.Printf("Posted reply in discussion %s (review %s)\n", d.DiscussionID, review.GetBranch())
		}

		if reply.Close {
			if err := upsource.ResolveDiscussion(r.ctx, r.upsourceClient, review.GetProjectID(), d.DiscussionID); err != nil {
				log.Printf("Failed to resolve discussion %s: %v\n", d.DiscussionID, err)
				continue
			}
			log.Printf("Resolved discussion %s (review %s)\n", d.DiscussionID, review.GetBranch())
		}
	}

	return nil
}

func (r *replier) resolveBotUserID() (string, error) {
	if r.botUserID != "" {
		return r.botUserID, nil
	}

	user, err := r.upsourceClient.GetCurrentUser(r.ctx)
	if err != nil {
		return "", err
	}
	r.botUserID = user.UserID

	return r.botUserID, nil
}

func buildThreadTranscript(comments []client.CommentDTO, botUserID string) []llm.CommentMsg {
	out := make([]llm.CommentMsg, 0, len(comments))
	for _, c := range comments {
		out = append(out, llm.CommentMsg{
			Author: c.AuthorID,
			IsBot:  c.AuthorID == botUserID,
			Text:   c.Text,
		})
	}
	return out
}

func buildReplyAnchorText(anchor client.AnchorDTO) string {
	if anchor.FileID == "" {
		return ""
	}

	var rangeText string
	if anchor.Range != nil {
		rangeText = fmt.Sprintf(" range=[%d,%d]", anchor.Range.StartOffset, anchor.Range.EndOffset)
	}

	return fmt.Sprintf("fileId=%s revisionId=%s inlineInRevision=%s%s", anchor.FileID, anchor.RevisionID, anchor.InlineInRevision, rangeText)
}
