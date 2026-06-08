package review

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/internal/llm"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

type replier struct {
	upsourceClient *client.Client
	config         *replierConfig
	ctx            context.Context
	llmReplier     *llm.Replier
	botUserID      string
}

type replierConfig struct {
	reviewedLabel      string
	maxPerThread       int
	searchReviewsQuery string
}

func newReplier(ctx context.Context, config *replierConfig, upsourceClient *client.Client, llmReplier *llm.Replier) (*replier, error) {
	replier := &replier{
		config:         config,
		ctx:            ctx,
		upsourceClient: upsourceClient,
		llmReplier:     llmReplier,
	}

	return replier, nil
}

// replyToOpenThreads scans reviews the bot has already engaged with and posts a
// follow-up reply in any thread where a human spoke after the bot's last word.
// Errors are logged per discussion / per review; a single failure never aborts the loop.
func (r *replier) replyToOpenThreads() error {
	botUserID, err := r.resolveBotUserID()
	if err != nil {
		return fmt.Errorf("failed to resolve bot user id: %w", err)
	}

	reviews, err := upsource.ListReviewedReviews(r.ctx, r.upsourceClient, r.config.searchReviewsQuery, r.config.reviewedLabel)
	if err != nil {
		return fmt.Errorf("failed to list reviewed reviews: %w", err)
	}

	projects, reviewsByProject := groupReviewsByProject(reviews)
	log.Printf("Reply pass: scanning %d already-reviewed reviews across %d projects\n", len(reviews), len(projects))

	for _, projectID := range projects {
		projectReviews := reviewsByProject[projectID]
		sort.Slice(projectReviews, func(i, j int) bool {
			return projectReviews[i].GetBranch() < projectReviews[j].GetBranch()
		})
		log.Printf("Reply pass: processing %d reviews in project %s\n", len(projectReviews), projectID)

		for _, review := range projectReviews {
			if err := r.replyInReview(review, botUserID); err != nil {
				log.Printf("Reply pass error in review %s: %v\n", review.GetBranch(), err)
			}
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

	reviewReplier := r.llmReplier.ForReview(review)

	for _, d := range discussions {
		last, ok := upsource.ShouldReplyToDiscussion(d, r.config.reviewedLabel, botUserID, r.config.maxPerThread)
		if !ok {
			log.Printf("Skipping discussion %s in review %s\n", d.DiscussionID, review.GetBranch())
			continue
		}

		reply, lerr := reviewReplier.Reply(d, botUserID)
		if lerr != nil {
			log.Printf("Failed to get reply for discussion %s: %v\n", d.DiscussionID, lerr)
			continue
		}

		if reply.Comment != "" {
			if err := upsource.AddDiscussionComment(r.ctx, r.upsourceClient, review.GetProjectID(), d.DiscussionID, last.CommentID, reply.Comment); err != nil {
				log.Printf("Failed to post reply for discussion %s: %v\n", d.DiscussionID, err)
				continue
			}
			metrics.DefaultRecorder.RecordReplySent()
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
