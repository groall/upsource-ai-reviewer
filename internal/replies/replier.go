package replies

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	appConfig "github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

type Replier struct {
	upsourceClient *client.Client
	config         *config
	ctx            context.Context
	generator      *reviewReplyGenerator
	botUserID      string
}

type config struct {
	reviewedLabel      string
	maxPerThread       int
	searchReviewsQuery string
}

func NewReplier(ctx context.Context, appConfig *appConfig.Config) (*Replier, error) {
	upsourceClient, err := upsource.NewClient(appConfig.Upsource.BaseURL, appConfig.Upsource.Username, appConfig.Upsource.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create Upsource client: %w", err)
	}

	repliesGenerator, err := newGenerator(ctx, appConfig.Replies, appConfig.Providers)
	if err != nil {
		return nil, fmt.Errorf("failed to create Replier generator: %w", err)
	}
	reviewReplyGenerator, err := newReviewReplyGenerator(repliesGenerator, &appConfig.Gitlab)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReviewReplyGenerator: %w", err)
	}

	config := &config{
		reviewedLabel:      appConfig.Upsource.ReviewedLabel,
		maxPerThread:       appConfig.Replies.MaxPerThread,
		searchReviewsQuery: appConfig.Upsource.Query,
	}

	replier := &Replier{
		config:         config,
		ctx:            ctx,
		upsourceClient: upsourceClient,
		generator:      reviewReplyGenerator,
	}

	return replier, nil
}

// Run scans reviews the bot has already engaged with and posts a
// follow-up reply in any thread where a human spoke after the bot's last word.
// Errors are logged per discussion / per review; a single failure never aborts the loop.
func (r *Replier) Run() error {
	botUserID, err := r.resolveBotUserID()
	if err != nil {
		return fmt.Errorf("failed to resolve bot user id: %w", err)
	}

	r.generator.setBotUserID(botUserID)

	reviews, err := upsource.ListReviewedReviews(r.ctx, r.upsourceClient, r.config.searchReviewsQuery, r.config.reviewedLabel)
	if err != nil {
		return fmt.Errorf("failed to list reviewed reviews: %w", err)
	}

	projects, reviewsByProject := upsource.GroupReviewsByProject(reviews)
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

func (r *Replier) replyInReview(review *upsource.Review, botUserID string) error {
	discussions, err := upsource.ListReviewDiscussions(r.ctx, r.upsourceClient, review)
	if err != nil {
		return fmt.Errorf("list discussions: %w", err)
	}
	if len(discussions) == 0 {
		return nil
	}

	err = r.generator.prepareReview(review)
	if err != nil {
		return fmt.Errorf("prepare review: %w", err)
	}

	for _, d := range discussions {
		last, ok := upsource.ShouldReplyToDiscussion(d, r.config.reviewedLabel, botUserID, r.config.maxPerThread)
		if !ok {
			//log.Printf("Skipping discussion %s in review %s\n", d.DiscussionID, review.GetBranch())
			continue
		}

		reply, lerr := r.generator.reply(d)
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

func (r *Replier) resolveBotUserID() (string, error) {
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
