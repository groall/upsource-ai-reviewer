package review

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/llm"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

// Reviewer is responsible for reviewing code changes in Upsource using AI models.
type Reviewer struct {
	upsourceClient *client.Client
	config         *config.Config
	ctx            context.Context
	llmReviewer    *llm.Reviewer
	gitProvider    git.Provider
}

// New creates a new Reviewer instance.
func New(ctx context.Context, config *config.Config) (*Reviewer, error) {
	upsourceClient, err := client.New(client.Options{
		BaseURL:  config.Upsource.BaseURL,
		Username: config.Upsource.Username,
		Password: config.Upsource.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Upsource client: %w", err)
	}

	gitlabProvider, err := git.NewGitlabProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab provider: %w", err)
	}

	llmReviewer, err := llm.New(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM reviewer: %w", err)
	}

	return &Reviewer{
		upsourceClient: upsourceClient,
		gitProvider:    gitlabProvider,
		llmReviewer:    llmReviewer,
		config:         config,
		ctx:            ctx,
	}, nil
}

// Run starts the AI Reviewer process, fetching reviews from Upsource, generating comments, and posting them back.
func (r *Reviewer) Run() error {
	reviews, err := r.listReviews()
	if err != nil {
		return fmt.Errorf("failed to list reviews: %w", err)
	}
	log.Printf("Found %d reviews to process.\n", len(reviews))

	var comments []*llm.ReviewComment
	for _, review := range reviews {
		if comments, err = r.doReview(review); err != nil {
			log.Printf("Error processing review %s: %v\n", review.GetBranch(), err)
			continue
		}

		if len(comments) == 0 {
			log.Printf("AI Reviewer found no issues to comment on for %s.\n", review.GetBranch())
			continue
		}

		if err := r.postComments(review, comments); err != nil {
			log.Printf("Error posting comments for review %s: %v\n", review.GetBranch(), err)
		}
	}

	return nil
}

func (r *Reviewer) doReview(review *upsource.Review) ([]*llm.ReviewComment, error) {
	log.Printf("Processing review for the branch %s.\n", review.GetBranch())

	changes, commitsComments, err := r.gitProvider.GetReviewChanges(review)
	if err != nil {
		return nil, fmt.Errorf("Error getting review changes for %s: %w\n", review.GetBranch(), err)
	}

	comments, err := r.llmReviewer.Do(changes, commitsComments)
	if err != nil {
		return nil, fmt.Errorf("Error getting review comments for %s: %w\n", review.GetBranch(), err)
	}

	if err := upsource.AddReviewLabel(r.ctx, r.upsourceClient, review, r.config.Upsource.ReviewedLabel); err != nil {
		return nil, fmt.Errorf("failed to add review label: %w", err)
	}

	return comments, nil
}

// listReviews fetches reviews from Upsource based on the configured query.
func (r *Reviewer) listReviews() ([]*upsource.Review, error) {
	reviews, err := upsource.ListReviews(r.ctx, r.upsourceClient, r.config.Upsource.Query, r.config.Upsource.ReviewedLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	return reviews, nil
}

// postComments posts review comments to Upsource, splitting high severity comments into separate discussions if configured.
func (r *Reviewer) postComments(review *upsource.Review, comments []*llm.ReviewComment) error {
	var postInOneComments []*llm.ReviewComment
	var inlineComments []*llm.ReviewComment

	for _, comment := range comments {
		thereIsLine := comment.LineNumber > 0 && comment.FilePath != "" && comment.LineVerified
		switch {
		case comment.Severity == llm.SeverityHigh && r.config.Comments.PostInLine == "high" && thereIsLine:
			inlineComments = append(inlineComments, comment)
		case (comment.Severity == llm.SeverityMedium || comment.Severity == llm.SeverityHigh) && r.config.Comments.PostInLine == "mid" && thereIsLine:
			inlineComments = append(inlineComments, comment)
		case r.config.Comments.PostInLine == "low" && thereIsLine:
			inlineComments = append(inlineComments, comment)
		default:
			postInOneComments = append(postInOneComments, comment)
		}
	}

	for _, comment := range inlineComments {
		err := upsource.CreateDiscussion(r.ctx, r.upsourceClient, r.config, upsource.CreateDiscussionRequest{
			Review:  review,
			Comment: comment.Comment,
			File:    comment.FilePath,
			Line:    comment.LineNumber,
		})
		if err != nil {
			return fmt.Errorf("failed to post inline comment to %s:%d -> %s: %w", comment.FilePath, comment.LineNumber, comment.Comment, err)
		}
	}

	if len(postInOneComments) > 0 {
		if err := r.createOneDiscussion(postInOneComments, review); err != nil {
			return fmt.Errorf("failed to post comments to review %s: %w", review.GetBranch(), err)
		}
	}

	return nil
}

// createOneDiscussion posts a single discussion to Upsource, containing low and medium priority comments.
func (r *Reviewer) createOneDiscussion(comments []*llm.ReviewComment, review *upsource.Review) error {
	discussionText := generateLowPriorityComment(comments)
	if len(discussionText) > 0 {
		err := upsource.CreateDiscussion(r.ctx, r.upsourceClient, r.config, upsource.CreateDiscussionRequest{
			Review:  review,
			Comment: discussionText,
			File:    "",
			Line:    0,
		})
		if err != nil {
			return fmt.Errorf("failed to post low priority comment to review %s: %w", review.GetBranch(), err)
		}
	}

	return nil
}

// generateLowPriorityComment creates a formatted string for low and medium priority comments.
func generateLowPriorityComment(comments []*llm.ReviewComment) string {
	var commentsBuilder strings.Builder
	commentsBuilder.WriteString("### Low-Medium Priority Comments (AI generated):\n\n")

	for _, comment := range comments {
		commentsBuilder.WriteString(fmt.Sprintf("**%s** %s:%d %s\n\n", strings.ToUpper(comment.Severity), comment.FilePath, comment.LineNumber, comment.Comment))
	}

	return commentsBuilder.String()
}
