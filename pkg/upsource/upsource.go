package upsource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/groall/upsource-go-client/client"
)

// Review represents a review in Upsource.
type Review struct {
	defaultBranch    string
	branch           string
	gitGroup         string
	gitName          string
	review           *client.ReviewDescriptorDTO
	filesDiffSummary []client.FileDiffSummaryDTO
}

func (r *Review) GetDefaultBranch() string {
	return r.defaultBranch
}

func (r *Review) GetBranch() string {
	return r.branch
}

func (r *Review) GetGitGroupAndName() (string, string) {
	return r.gitGroup, r.gitName
}

// ListReviews lists reviews in Upsource that match the given query.
func ListReviews(ctx context.Context, upsourceClient *client.Client, query string, reviewedLabel string) ([]*Review, error) {
	upsourceReviews, err := upsourceClient.GetReviews(ctx, client.ReviewsRequestDTO{
		Limit: 10000,
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get upsourceReviews: %w", err)
	}

	var reviewsToDo []*Review

	for _, review := range upsourceReviews.Reviews {
		if len(review.Branch) == 0 {
			log.Printf("Skipping review %s because it has no branch\n", review.Title)
			continue
		}

		var skip bool
		for _, label := range review.Labels {
			if label.Name == reviewedLabel {
				log.Printf("Skipping review %s already AI-reviewed.\n", review.Title)
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		reviewTodo, err := newReviewFromUpsourceReview(ctx, review, upsourceClient)
		if err != nil {
			log.Printf("Skipping review %s: %v\n", review.Title, err)
			continue
		}

		reviewsToDo = append(reviewsToDo, reviewTodo)
	}

	return reviewsToDo, nil
}

// newReviewFromUpsourceReview creates a Review from an Upsource review descriptor.
func newReviewFromUpsourceReview(ctx context.Context, upsourceReview client.ReviewDescriptorDTO, upsourceClient *client.Client) (*Review, error) {
	projectID := upsourceReview.ReviewID.ProjectID

	projectVcsLinks, err := upsourceClient.GetProjectVcsLinks(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Error getting project VCS links for %s: %v\n", upsourceReview.Title, err)
	}

	groupPath, repoName, err := parseGitGroupAndName(projectVcsLinks.Repo[0].URL[0])
	if err != nil {
		return nil, fmt.Errorf("Error parsing Git group and name for %s: %v\n", projectID, err)
	}

	projectInfo, err := upsourceClient.GetProjectInfo(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Error getting project info for %s: %v\n", upsourceReview.Title, err)
	}

	var t = true
	reviewSummaryChanges, err := upsourceClient.GetReviewSummaryChanges(ctx, client.ReviewSummaryChangesRequestDTO{
		ReviewID: upsourceReview.ReviewID,
		Revisions: &client.RevisionsSetDTO{
			SelectAll: &t,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error getting review summary changes for %s: %v\n", upsourceReview.Title, err)
	}

	return &Review{
		defaultBranch:    projectInfo.DefaultBranch,
		branch:           upsourceReview.Branch[0],
		gitGroup:         groupPath,
		gitName:          repoName,
		review:           &upsourceReview,
		filesDiffSummary: reviewSummaryChanges.FileDiffSummary,
	}, nil
}

// parseGitGroupAndName parses a git remote URL and returns:
// - groupPath: everything except the final path segment (can contain slashes for subgroups)
// - repoName: final path segment (without ".git")
// Example:
//
//	input:  "git@gitlab.com:groupName/repo.git"
//	output: groupPath="groupName", repoName="repo", nil
func parseGitGroupAndName(remote string) (groupPath, repoName string, err error) {
	if remote == "" {
		return "", "", errors.New("empty remote")
	}

	// Remove optional trailing slash(es)
	remote = strings.TrimRight(remote, "/")

	// Remove trailing .git if present (we'll also remove from path later, but strip early)
	remote = strings.TrimSuffix(remote, ".git")

	var p string // the path portion like "group/subgroup/repo"
	// Detect scp-like syntax: "user@host:group/repo"
	// It contains a ':' after the host but no "//"
	if strings.Contains(remote, ":") && !strings.Contains(remote, "://") {
		// Split at first colon and take the right side
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) < 2 || parts[1] == "" {
			return "", "", fmt.Errorf("invalid scp-like remote: %q", remote)
		}
		p = parts[1]
	} else {
		// Try to parse as URL (http(s)://, ssh://, etc.)
		u, parseErr := url.Parse(remote)
		if parseErr != nil || u.Path == "" {
			// as fallback, try to interpret entire remote as a path
			// (this handles "host/group/repo" without scheme)
			// but first try removing any user@host/ prefix if present
			// e.g. "gitlab.example.com/group/repo"
			// find first '/' and take everything after it
			if i := strings.Index(remote, "/"); i >= 0 && i < len(remote)-1 {
				p = remote[i+1:]
			} else {
				return "", "", fmt.Errorf("cannot parse remote: %q", remote)
			}
		} else {
			// u.Path starts with '/', strip it
			p = strings.TrimPrefix(u.Path, "/")
		}
	}

	// Normalize: remove trailing .git if somehow still present
	p = strings.TrimSuffix(p, ".git")
	p = strings.Trim(p, "/")

	if p == "" {
		return "", "", fmt.Errorf("no path found in remote: %q", remote)
	}

	segments := strings.Split(p, "/")
	if len(segments) < 2 {
		// less than two segments means we can't get group + repo
		return "", "", fmt.Errorf("path %q does not contain group and repo", p)
	}

	repoName = segments[len(segments)-1]
	groupPath = path.Join(segments[:len(segments)-1]...) // preserves subgroups with slashes

	return groupPath, repoName, nil
}

func AddReviewLabel(ctx context.Context, upsourceClient *client.Client, review *Review, label string) error {
	_, err := upsourceClient.AddReviewLabel(ctx, client.UpdateReviewLabelRequestDTO{
		ProjectID: review.review.ReviewID.ProjectID,
		ReviewID:  &review.review.ReviewID,
		Label: client.LabelDTO{
			Name: label,
		},
	})

	return err
}
