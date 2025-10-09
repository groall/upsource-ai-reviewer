package git

import (
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

// GitlabProvider implements the Provider interface for GitLab.
type GitlabProvider struct {
	gitlabClient *gitlab.Client
}

// NewGitlabProvider creates a new GitlabProvider instance.
func NewGitlabProvider(cfg *config.Config) (*GitlabProvider, error) {
	gitlabClient, err := gitlab.NewClient(cfg.Gitlab.AccessToken, gitlab.WithBaseURL(cfg.Gitlab.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitlabProvider{
		gitlabClient: gitlabClient,
	}, nil
}

// GetReviewChanges fetches the changes between the default branch and the review branch.
func (g *GitlabProvider) GetReviewChanges(review Review) (string, string, error) {
	fmt.Printf("Fetching changes between branch '%s' and '%s'\n", review.GetDefaultBranch(), review.GetBranch())

	defaultBranch, branch := review.GetDefaultBranch(), review.GetBranch()
	compareOpts := &gitlab.CompareOptions{
		From: &defaultBranch,
		To:   &branch,
	}
	groupPath, repoName := review.GetGitGroupAndName()
	gitlabProjectID := fmt.Sprintf("%s/%s", groupPath, repoName)

	comparison, _, err := g.gitlabClient.Repositories.Compare(gitlabProjectID, compareOpts)
	if err != nil {
		return "", "", fmt.Errorf("failed to compare branches for review %s: %w", review.GetBranch(), err)
	}

	if len(comparison.Diffs) == 0 {
		return "", "", fmt.Errorf("No diffs found between '%s' and '%s'.\n", review.GetDefaultBranch(), review.GetBranch())
	}

	return createChangesText(comparison.Diffs), createCommentsText(comparison.Commits), nil
}

// createChangesText constructs the changes text from the GitLab comparison diffs.
func createChangesText(diffs []*gitlab.Diff) string {
	var changesBuilder strings.Builder
	for _, diff := range diffs {
		if diff.NewFile {
			changesBuilder.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n", diff.NewPath))
		} else if diff.DeletedFile {
			changesBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ /dev/null\n", diff.OldPath))
		} else if diff.RenamedFile {
			changesBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", diff.OldPath, diff.NewPath))
		} else {
			changesBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", diff.OldPath, diff.NewPath))
		}
		changesBuilder.WriteString(diff.Diff)
		changesBuilder.WriteString("\n\n")
	}

	return changesBuilder.String()
}

// createCommentsText constructs the comments text from the GitLab comparison commits.
func createCommentsText(commits []*gitlab.Commit) string {
	var commentsBuilder strings.Builder
	for _, comment := range commits {
		commentsBuilder.WriteString(fmt.Sprintf("Commit %s:\n%s\n\n", comment.ID, comment.Message))
	}

	return commentsBuilder.String()
}
