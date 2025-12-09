package git

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	cfg := &config.Config{
		Gitlab: config.Gitlab{
			BaseURL:     "https://gitlab.com",
			AccessToken: "test-token",
		},
	}

	provider, err := NewGitlabProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestGetReviewChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/group/repo/repository/compare" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{
				"commits": [{"id": "123", "message": "feat: new feature"}],
				"diffs": [{
					"old_path": "file.go",
					"new_path": "file.go",
					"diff": "--- a/file.go
+++ b/file.go
@@ -1,1 +1,1 @@
-hello
+world"
				}]
			}`)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Gitlab: config.Gitlab{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		},
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
		if r.URL.Path == "/api/v4/projects/group/repo/repository/compare" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"commits": [], "diffs": []}`)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Gitlab: config.Gitlab{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		},
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
