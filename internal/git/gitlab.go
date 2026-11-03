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

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

// GitlabProvider clones GitLab repositories for agentic review.
type GitlabProvider struct {
	baseURL     string
	accessToken string
}

// NewGitlabProvider creates a new GitlabProvider instance.
func NewGitlabProvider(cfg *config.Gitlab) (*GitlabProvider, error) {
	return &GitlabProvider{
		baseURL:     cfg.BaseURL,
		accessToken: cfg.AccessToken,
	}, nil
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

	repoDir := filepath.Join(baseDir, namespace, repoName)

	if isGitRepo(repoDir) {
		log.Printf("Reusing clone of %s/%s at %s (branch %s)", namespace, repoName, repoDir, branch)
		return repoDir, g.updateRepo(repoDir, cloneURL, branch)
	}

	args := []string{"clone", "--single-branch", "--branch", branch, cloneURL, repoDir}

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
	remotes, err := runGit(repoDir, "remote")
	if err != nil {
		return fmt.Errorf("git remote failed: %w: %s", err, remotes)
	}
	remoteArgs := []string{"remote", "add", "origin", cloneURL}
	for _, remote := range strings.Fields(remotes) {
		if remote == "origin" {
			remoteArgs[1] = "set-url"
			break
		}
	}
	if out, err := runGit(repoDir, remoteArgs...); err != nil {
		return fmt.Errorf("git remote %s failed: %w: %s", remoteArgs[1], err, out)
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
	log.Printf("Executing command: git %s", strings.Join(redactGitArgs(args), " "))
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func redactGitArgs(args []string) []string {
	redacted := make([]string, len(args))
	for i, arg := range args {
		redacted[i] = arg
		u, err := url.Parse(arg)
		if err == nil && u.User != nil {
			u.User = url.UserPassword(u.User.Username(), "***")
			redacted[i] = u.String()
		}
	}
	return redacted
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
