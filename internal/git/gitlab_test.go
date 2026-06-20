package git

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

// MockReview is a mock implementation of the Review interface.
type MockReview struct {
	mock.Mock
}

func (m *MockReview) GetDefaultBranch() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockReview) GetBranch() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockReview) GetGitNamespaceAndName() (string, string) {
	args := m.Called()
	return args.String(0), args.String(1)
}

func TestNewGitlabProvider(t *testing.T) {
	cfg := &config.Gitlab{
		BaseURL:     "https://gitlab.com",
		AccessToken: "test-token",
	}

	provider, err := NewGitlabProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestGetReviewChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/group/repo/repository/branches" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"name":"main","default":true}]`)
			return
		}

		if r.URL.Path == "/api/v4/projects/group/repo/repository/compare" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{
				"commits": [{"id": "123", "message": "feat: new feature"}],
				"diffs": [{
					"old_path": "file.go",
					"new_path": "file.go",
					"diff": "--- a/file.go\n+++ b/file.go\n@@ -1,1 +1,1 @@\n-hello\n+world"
				}]
			}`)
		}
	}))
	defer server.Close()

	cfg := &config.Gitlab{
		BaseURL:     server.URL,
		AccessToken: "test-token",
	}

	provider, err := NewGitlabProvider(cfg)
	assert.NoError(t, err)

	review := new(MockReview)
	review.On("GetDefaultBranch").Return("main")
	review.On("GetBranch").Return("feature")
	review.On("GetGitNamespaceAndName").Return("group", "repo")

	changes, comments, err := provider.GetReviewChanges(review)
	assert.NoError(t, err)
	assert.NotEmpty(t, changes)
	assert.NotEmpty(t, comments)
}

func TestGetReviewChanges_NoDiffs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/group/repo/repository/branches" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"name":"main","default":true}]`)
			return
		}

		if r.URL.Path == "/api/v4/projects/group/repo/repository/compare" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"commits": [], "diffs": []}`)
		}
	}))
	defer server.Close()

	cfg := &config.Gitlab{
		BaseURL:     server.URL,
		AccessToken: "test-token",
	}

	provider, err := NewGitlabProvider(cfg)
	assert.NoError(t, err)

	review := new(MockReview)
	review.On("GetDefaultBranch").Return("main")
	review.On("GetBranch").Return("feature")
	review.On("GetGitNamespaceAndName").Return("group", "repo")

	_, _, err = provider.GetReviewChanges(review)
	assert.Error(t, err)
}

func TestCreateChangesText(t *testing.T) {
	diffs := []*gitlab.Diff{
		{
			OldPath: "file.go",
			NewPath: "file.go",
			Diff: `--- a/file.go
+++ b/file.go
@@ -1,1 +1,1 @@
-hello
+world`,
		},
		{
			NewFile: true,
			NewPath: "new_file.go",
			Diff: `--- /dev/null
+++ b/new_file.go
@@ -0,0 +1,1 @@
+new file`,
		},
		{
			DeletedFile: true,
			OldPath:     "deleted_file.go",
			Diff: `--- a/deleted_file.go
+++ /dev/null
@@ -1,1 +0,0 @@
-deleted file`,
		},
		{
			RenamedFile: true,
			OldPath:     "old_name.go",
			NewPath:     "new_name.go",
			Diff: `--- a/old_name.go
+++ b/new_name.go
@@ -1,1 +1,1 @@
-old
+new`,
		},
	}

	expected := `--- a/file.go
+++ b/file.go
--- a/file.go
+++ b/file.go
@@ -1,1 +1,1 @@
-hello
+world

--- /dev/null
+++ b/new_file.go
--- /dev/null
+++ b/new_file.go
@@ -0,0 +1,1 @@
+new file

--- a/deleted_file.go
+++ /dev/null
--- a/deleted_file.go
+++ /dev/null
@@ -1,1 +0,0 @@
-deleted file

--- a/old_name.go
+++ b/new_name.go
--- a/old_name.go
+++ b/new_name.go
@@ -1,1 +1,1 @@
-old
+new

`
	assert.Equal(t, expected, createChangesText(diffs))
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

func TestCreateCommentsText(t *testing.T) {
	commits := []*gitlab.Commit{
		{
			ID:      "123",
			Message: "feat: new feature",
		},
		{
			ID:      "456",
			Message: "fix: bug fix",
		},
	}

	expected := `Commit 123:
feat: new feature

Commit 456:
fix: bug fix

`
	assert.Equal(t, expected, createCommitsCommentsText(commits))
}
