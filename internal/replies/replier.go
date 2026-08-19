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
	botNickname    string
}

type config struct {
	reviewedLabel      string
	logMessages        bool
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
		logMessages:        appConfig.Replies.LogMessages,
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

// Run scans matching open reviews and posts a follow-up reply in eligible discussions.
// Errors are logged per discussion / per review; a single failure never aborts the loop.
func (r *Replier) Run() error {
	botUserID, _, err := r.resolveBotIdentity()
	if err != nil {
		return fmt.Errorf("failed to resolve bot user id: %w", err)
	}

	r.generator.setBotUserID(botUserID)

	reviews, err := upsource.ListOpenReviews(r.ctx, r.upsourceClient, r.config.searchReviewsQuery)
	if err != nil {
		return fmt.Errorf("failed to list open reviews: %w", err)
	}

	projects, reviewsByProject := upsource.GroupReviewsByProject(reviews)
	r.logf("scanning %d reviews for discussions across %d projects", len(reviews), len(projects))

	for _, projectID := range projects {
		projectReviews := reviewsByProject[projectID]
		sort.Slice(projectReviews, func(i, j int) bool {
			return projectReviews[i].GetBranch() < projectReviews[j].GetBranch()
		})
		r.logf("processing %d reviews in project %s", len(projectReviews), projectID)

		for _, review := range projectReviews {
			if err := r.replyInReview(review, botUserID); err != nil {
				r.logf("reply error in review %s: %v", review.GetBranch(), err)
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

	discussionsToReply := discussions[0:0]
	for _, d := range discussions {
		_, ok := upsource.ShouldReplyToDiscussion(d, r.config.reviewedLabel, botUserID, r.config.maxPerThread, r.botNickname)
		if ok {
			discussionsToReply = append(discussionsToReply, d)
		}
	}

	if len(discussionsToReply) == 0 {
		r.logf("skipping the review %s as there are no unanswered discussions and there are no mentions", review.GetTitle())
		return nil
	}

	err = r.generator.prepareReview(review)
	if err != nil {
		return fmt.Errorf("prepare review: %w", err)
	}

	for _, d := range discussionsToReply {
		reply, lerr := r.generator.reply(d)
		if lerr != nil {
			r.logf("failed to get reply for discussion %s: %v", d.DiscussionID, lerr)
			continue
		}

		if reply.Comment != "" {
			last := d.Comments[len(d.Comments)-1]
			if err := upsource.AddDiscussionComment(r.ctx, r.upsourceClient, review.GetProjectID(), d.DiscussionID, last.CommentID, reply.Comment); err != nil {
				r.logf("failed to post reply for discussion %s: %v", d.DiscussionID, err)
				continue
			}
			metrics.DefaultRecorder.RecordReplySent()
			r.logf("posted reply in discussion %s (review %s)", d.DiscussionID, review.GetBranch())
		}

		if reply.Close {
			if err := upsource.ResolveDiscussion(r.ctx, r.upsourceClient, review.GetProjectID(), d.DiscussionID); err != nil {
				r.logf("failed to resolve discussion %s: %v", d.DiscussionID, err)
				continue
			}
			r.logf("resolved discussion %s (review %s)", d.DiscussionID, review.GetBranch())
		}
	}

	return nil
}

func (r *Replier) logf(format string, args ...interface{}) {
	if r.config.logMessages {
		log.Printf("Replier: "+format, args...)
	}
}

func (r *Replier) resolveBotIdentity() (string, string, error) {
	if r.botUserID != "" && r.botNickname != "" {
		return r.botUserID, r.botNickname, nil
	}

	user, err := r.upsourceClient.GetCurrentUser(r.ctx)
	if err != nil {
		return "", "", err
	}
	if user.UserID == "" || user.Login == "" {
		return "", "", fmt.Errorf("current user has empty user id or login")
	}
	r.botUserID = user.UserID
	r.botNickname = user.Login

	return r.botUserID, r.botNickname, nil
}
