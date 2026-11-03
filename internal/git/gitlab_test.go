package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

func TestNewGitlabProvider(t *testing.T) {
	cfg := &config.Gitlab{
		BaseURL:     "https://gitlab.com",
		AccessToken: "test-token",
	}

	provider, err := NewGitlabProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, cfg.BaseURL, provider.baseURL)
	assert.Equal(t, cfg.AccessToken, provider.accessToken)
}

func TestCloneHost(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{"plain host", "https://gitlab.example.com", "https://gitlab.example.com", false},
		{"api path stripped", "https://gitlab.example.com/api/v4", "https://gitlab.example.com", false},
		{"trailing slash", "https://gitlab.example.com/", "https://gitlab.example.com", false},
		{"missing scheme", "gitlab.example.com", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cloneHost(tt.baseURL)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateRepo(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	workDir := filepath.Join(baseDir, "work")
	cloneDir := filepath.Join(baseDir, "clone")

	_, err := runGit("", "init", "--bare", remoteDir)
	assert.NoError(t, err)

	assert.NoError(t, os.MkdirAll(workDir, 0o755))
	_, err = runGit(workDir, "init")
	assert.NoError(t, err)
	_, err = runGit(workDir, "config", "user.email", "test@example.com")
	assert.NoError(t, err)
	_, err = runGit(workDir, "config", "user.name", "Test User")
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("main\n"), 0o644))
	_, err = runGit(workDir, "add", "file.txt")
	assert.NoError(t, err)
	_, err = runGit(workDir, "commit", "-m", "initial")
	assert.NoError(t, err)
	_, err = runGit(workDir, "branch", "-M", "main")
	assert.NoError(t, err)
	_, err = runGit(workDir, "branch", "feature/one")
	assert.NoError(t, err)
	_, err = runGit(workDir, "remote", "add", "origin", remoteDir)
	assert.NoError(t, err)
	_, err = runGit(workDir, "push", "origin", "main", "feature/one")
	assert.NoError(t, err)

	_, err = runGit("", "clone", "--single-branch", "--branch", "main", remoteDir, cloneDir)
	assert.NoError(t, err)

	provider := &GitlabProvider{}
	err = provider.updateRepo(cloneDir, remoteDir, "feature/one")
	assert.NoError(t, err)

	branch, err := runGit(cloneDir, "rev-parse", "--abbrev-ref", "HEAD")
	assert.NoError(t, err)
	assert.Equal(t, "feature/one", branch)

	_, err = runGit(cloneDir, "show-ref", "--verify", "refs/remotes/origin/feature/one")
	assert.NoError(t, err)

	t.Run("missing origin", func(t *testing.T) {
		_, err := runGit(cloneDir, "remote", "remove", "origin")
		if !assert.NoError(t, err) {
			return
		}
		if !assert.NoError(t, provider.updateRepo(cloneDir, remoteDir, "main")) {
			return
		}
		origin, err := runGit(cloneDir, "remote", "get-url", "origin")
		assert.NoError(t, err)
		assert.Equal(t, remoteDir, origin)
		branch, err := runGit(cloneDir, "rev-parse", "--abbrev-ref", "HEAD")
		assert.NoError(t, err)
		assert.Equal(t, "main", branch)
		_, err = runGit(cloneDir, "show-ref", "--verify", "refs/remotes/origin/main")
		assert.NoError(t, err)
	})
}

func TestBuildCloneURL(t *testing.T) {
	got := buildCloneURL("https://gitlab.example.com", "secret-token", "group/sub", "repo")
	assert.Equal(t, "https://oauth2:secret-token@gitlab.example.com/group/sub/repo.git", got)
}

func TestRedactCloneURL(t *testing.T) {
	got := redactCloneURL("https://gitlab.example.com", "group", "repo")
	assert.Equal(t, "https://oauth2:***@gitlab.example.com/group/repo.git", got)
	assert.NotContains(t, got, "secret")
}
