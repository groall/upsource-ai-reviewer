package review

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/llm"
	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

// Reviewer is responsible for reviewing code changes in Upsource using AI models.
type Reviewer struct {
	upsourceClient *client.Client
	config         *config.Config
	ctx            context.Context
	llmReviewer    *llm.Reviewer
	replier        *replier
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

	activeProvider := config.Providers.ActiveLLMProvider()
	llmReviewerCfg := llm.ReviewConfig{
		UserPromptTemplate: config.Review.UserPromptTemplate,
		SystemMessage:      config.Review.SystemMessage,
		MaxPerReview:       config.Review.MaxPerReview,
		ActiveProvider:     activeProvider,
	}
	llmReviewer, err := llm.New(ctx, llmReviewerCfg, config.Providers, gitlabProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM reviewer: %w", err)
	}

	llmReplierCfg := llm.ReplyConfig{
		SystemMessage:  config.Replies.SystemMessage,
		ActiveProvider: activeProvider,
	}
	llmReplier := llm.NewReplier(llmReviewer, llmReplierCfg)

	replierConfig := &replierConfig{
		reviewedLabel:      config.Upsource.ReviewedLabel,
		maxPerThread:       config.Replies.MaxPerThread,
		searchReviewsQuery: config.Upsource.Query,
	}
	replier, err := newReplier(ctx, replierConfig, upsourceClient, llmReplier)
	if err != nil {
		return nil, fmt.Errorf("failed to create replier: %w", err)
	}

	return &Reviewer{
		upsourceClient: upsourceClient,
		llmReviewer:    llmReviewer,
		replier:        replier,
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
	projects, reviewsByProject := groupReviewsByProject(reviews)
	log.Printf("Found %d reviews to process across %d projects.\n", len(reviews), len(projects))

	var comments []*llm.ReviewComment
	for _, projectID := range projects {
		projectReviews := reviewsByProject[projectID]
		sort.Slice(projectReviews, func(i, j int) bool {
			return projectReviews[i].GetBranch() < projectReviews[j].GetBranch()
		})
		log.Printf("Processing %d reviews in project %s.\n", len(projectReviews), projectID)

		for _, review := range projectReviews {
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
	}

	if r.config.Replies.Enabled {
		if err := r.replier.replyToOpenThreads(); err != nil {
			log.Printf("Error during thread replies: %v", err)
		}
	}

	return nil
}

func groupReviewsByProject(reviews []*upsource.Review) ([]string, map[string][]*upsource.Review) {
	byProject := make(map[string][]*upsource.Review)
	for _, review := range reviews {
		projectID := review.GetProjectID()
		byProject[projectID] = append(byProject[projectID], review)
	}

	projects := make([]string, 0, len(byProject))
	for projectID := range byProject {
		projects = append(projects, projectID)
	}
	sort.Strings(projects)
	return projects, byProject
}

func (r *Reviewer) doReview(review *upsource.Review) ([]*llm.ReviewComment, error) {
	log.Printf("Processing review for the branch %s.\n", review.GetBranch())

	comments, err := r.llmReviewer.Do(review)
	if err != nil {
		return nil, fmt.Errorf("error getting review comments for %s: %w", review.GetBranch(), err)
	}

	if err := upsource.AddReviewLabel(r.ctx, r.upsourceClient, review, r.config.Upsource.ReviewedLabel); err != nil {
		return nil, fmt.Errorf("failed to add review label: %w", err)
	}
	metrics.DefaultRecorder.RecordReviewReviewed()

	return comments, nil
}

// listReviews fetches reviews from Upsource based on the configured query.
func (r *Reviewer) listReviews() ([]*upsource.Review, error) {
	reviews, err := upsource.ListReviews(r.ctx, r.upsourceClient, r.config.Upsource.Query, r.config.Upsource.ReviewedLabel, r.config.Upsource.InvitationLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	return reviews, nil
}

// postComments posts review comments to Upsource, splitting high severity comments into separate discussions if configured.
func (r *Reviewer) postComments(review *upsource.Review, comments []*llm.ReviewComment) error {
	comments = sortAndCapComments(comments, r.config.Review.MaxPerReview)

	var postInOneComments []*llm.ReviewComment
	var inlineComments []*llm.ReviewComment

	for _, comment := range comments {
		thereIsLine := comment.LineNumber > 0 && comment.FilePath != "" && comment.LineVerified
		if thereIsLine {
			inlineComments = append(inlineComments, comment)
		} else {
			postInOneComments = append(postInOneComments, comment)
		}
	}

	for _, comment := range inlineComments {
		err := r.createDiscussion(comment, review)
		if err != nil {
			return fmt.Errorf("failed to post inline comment to %s:%d -> %s: %w", comment.FilePath, comment.LineNumber, comment.Comment, err)
		}
	}

	if len(postInOneComments) > 0 {
		if err := r.createDiscussionWithoutLine(postInOneComments, review); err != nil {
			return fmt.Errorf("failed to post comments to review %s: %w", review.GetBranch(), err)
		}
	}

	return nil
}

// createDiscussionWithoutLine posts comments to a single discussion to Upsource without a link to a file and a line in it
func (r *Reviewer) createDiscussionWithoutLine(comments []*llm.ReviewComment, review *upsource.Review) error {
	discussionText := generateLowPriorityComment(comments)
	if len(discussionText) > 0 {
		err := upsource.CreateDiscussion(r.ctx, r.upsourceClient, r.config.Upsource.ReviewedLabel, upsource.CreateDiscussionRequest{
			Review:  review,
			Comment: discussionText,
			File:    "",
			Line:    0,
		})
		if err != nil {
			return fmt.Errorf("failed to post low priority comment to review %s: %w", review.GetBranch(), err)
		}
		metrics.DefaultRecorder.RecordReviewCommentsPosted(len(comments))
	}

	return nil
}

// createDiscussion posts a single discussion to Upsource.
func (r *Reviewer) createDiscussion(comment *llm.ReviewComment, review *upsource.Review) error {
	err := upsource.CreateDiscussion(r.ctx, r.upsourceClient, r.config.Upsource.ReviewedLabel, upsource.CreateDiscussionRequest{
		Review:  review,
		Comment: comment.Comment,
		File:    comment.FilePath,
		Line:    comment.LineNumber,
	})
	if err != nil {
		return fmt.Errorf("failed to post low priority comment to review %s: %w", review.GetBranch(), err)
	}
	metrics.DefaultRecorder.RecordReviewCommentsPosted(1)

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
