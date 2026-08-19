package git

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

// GitlabProvider implements the Provider interface for GitLab.
type GitlabProvider struct {
	gitlabClient *gitlab.Client
	baseURL      string
	accessToken  string
}

// NewGitlabProvider creates a new GitlabProvider instance.
func NewGitlabProvider(cfg *config.Gitlab) (*GitlabProvider, error) {
	gitlabClient, err := gitlab.NewClient(cfg.AccessToken, gitlab.WithBaseURL(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitlabProvider{
		gitlabClient: gitlabClient,
		baseURL:      cfg.BaseURL,
		accessToken:  cfg.AccessToken,
	}, nil
}

// GetReviewChanges fetches the changes between the default branch and the review branch.
func (g *GitlabProvider) GetReviewChanges(review Review) (string, string, error) {
	fmt.Printf("Fetching changes between branch '%s' and '%s'\n", review.GetDefaultBranch(), review.GetBranch())

	branch := review.GetBranch()
	compareOpts := &gitlab.CompareOptions{
		To: &branch,
	}
	namespace, repoName := review.GetGitNamespaceAndName()
	gitlabProjectID := fmt.Sprintf("%s/%s", namespace, repoName)

	branches, _, err := g.gitlabClient.Branches.ListBranches(gitlabProjectID, &gitlab.ListBranchesOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to list branches for project %s: %w", gitlabProjectID, err)
	}

	var defaultBranch string
	for _, b := range branches {
		if b.Default {
			defaultBranch = b.Name
			break
		}
	}
	if defaultBranch == "" {
		defaultBranch = review.GetDefaultBranch()
	}
	compareOpts.From = &defaultBranch

	comparison, _, err := g.gitlabClient.Repositories.Compare(gitlabProjectID, compareOpts)
	if err != nil {
		return "", "", fmt.Errorf("failed to compare branches for review %s: %w", review.GetBranch(), err)
	}

	if len(comparison.Diffs) == 0 {
		return "", "", fmt.Errorf("no diffs found between '%s' and '%s'", review.GetDefaultBranch(), review.GetBranch())
	}

	return createChangesText(comparison.Diffs), createCommitsCommentsText(comparison.Commits), nil
}

// PrepareReviewRepo ensures the review's repository is available under baseDir
// and checked out at its branch. It reuses an existing clone when present: it
// fetches updates from origin and checks out the branch instead of cloning
// from scratch. It returns the path to the repository working tree.
func (g *GitlabProvider) PrepareReviewRepo(review Review, baseDir string) (string, error) {
	host, err := cloneHost(g.baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to resolve clone host: %w", err)
	}

	namespace, repoName := review.GetGitNamespaceAndName()
	branch := review.GetBranch()
	cloneURL := buildCloneURL(host, g.accessToken, namespace, repoName)
	redactedURL := redactCloneURL(host, namespace, repoName)

	repoDir := filepath.Join(baseDir, namespace, repoName)

	if isGitRepo(repoDir) {
		log.Printf("Reusing clone of %s/%s at %s (branch %s)", namespace, repoName, repoDir, branch)
		return repoDir, g.updateRepo(repoDir, cloneURL, branch)
	}

	args := []string{"clone", "--single-branch", "--branch", branch, cloneURL, repoDir}
	log.Printf("Cloning %s/%s (branch %s): git %s",
		namespace, repoName, branch,
		strings.Replace(strings.Join(args, " "), cloneURL, redactedURL, 1))

	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return "", fmt.Errorf("failed to create clone parent directory: %w", err)
	}
	if out, err := runGit("", args...); err != nil {
		log.Printf("git clone failed: %v; command output: %s", err, out)
		return "", fmt.Errorf("git clone failed: %w: %s", err, out)
	}

	return repoDir, nil
}

// updateRepo refreshes an existing clone: it points origin at the current clone
// URL (the access token may have rotated), fetches the branch and resets the
// working tree to match origin.
func (g *GitlabProvider) updateRepo(repoDir, cloneURL, branch string) error {
	if out, err := runGit(repoDir, "remote", "set-url", "origin", cloneURL); err != nil {
		return fmt.Errorf("git remote set-url failed: %w: %s", err, out)
	}
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
	if out, err := runGit(repoDir, "fetch", "--prune", "origin", refspec); err != nil {
		return fmt.Errorf("git fetch failed: %w: %s", err, out)
	}
	if out, err := runGit(repoDir, "checkout", "-B", branch, "origin/"+branch); err != nil {
		return fmt.Errorf("git checkout failed: %w: %s", err, out)
	}
	if out, err := runGit(repoDir, "reset", "--hard", "origin/"+branch); err != nil {
		return fmt.Errorf("git reset failed: %w: %s", err, out)
	}
	if out, err := runGit(repoDir, "clean", "-fd"); err != nil {
		return fmt.Errorf("git clean failed: %w: %s", err, out)
	}

	return nil
}

// isGitRepo reports whether dir contains a git repository.
func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// runGit runs a git command in dir (or the current directory when empty) and
// returns its combined, trimmed output.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// cloneHost extracts scheme+host from the GitLab API base URL, dropping any API path.
func cloneHost(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid gitlab baseUrl %q: %w", baseURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("gitlab baseUrl %q must include scheme and host", baseURL)
	}

	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

// buildCloneURL builds an HTTPS clone URL embedding the access token.
func buildCloneURL(host, token, namespace, repoName string) string {
	scheme, hostOnly, _ := strings.Cut(strings.TrimSuffix(host, "/"), "://")
	return fmt.Sprintf("%s://oauth2:%s@%s/%s/%s.git", scheme, token, hostOnly, namespace, repoName)
}

// redactCloneURL builds the same clone URL with the token masked, for logging.
func redactCloneURL(host, namespace, repoName string) string {
	return buildCloneURL(host, "***", namespace, repoName)
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

// createCommitsCommentsText constructs the comments text from the GitLab comparison commits.
func createCommitsCommentsText(commits []*gitlab.Commit) string {
	var commentsBuilder strings.Builder
	for _, comment := range commits {
		commentsBuilder.WriteString(fmt.Sprintf("Commit %s:\n%s\n\n", comment.ID, comment.Message))
	}

	return commentsBuilder.String()
}
