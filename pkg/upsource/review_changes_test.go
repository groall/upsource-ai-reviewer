package upsource

import (
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func TestReviewDiffAddedAndDeletedHeaders(t *testing.T) {
	for _, tc := range []struct{ old, new, want string }{
		{"/dev/null", "file.go", "--- /dev/null\n+++ b/file.go\n"},
		{"file.go", "/dev/null", "--- a/file.go\n+++ /dev/null\n"},
	} {
		var b strings.Builder
		appendReviewFileDiff(&b, "file.go", tc.old, tc.new, "@@ -0,0 +1 @@\n+hello\n")
		if !strings.Contains(b.String(), tc.want) {
			t.Errorf("headers: got %q, want %q", b.String(), tc.want)
		}
	}
}

func TestSplitDiffLinesPreservesFinalNewlineChange(t *testing.T) {
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: splitDiffLines("hello"), B: splitDiffLines("hello\n"), Context: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "-hello\n\\ No newline at end of file\n+hello\n") {
		t.Fatalf("unexpected diff: %q", diff)
	}
}

func TestCountChangedCharacters(t *testing.T) {
	diff := "--- a/file\n+++ b/file\n@@ -1 +1 @@\n---old\n+++new\n"
	added, removed := countChangedCharacters(diff)
	if added != 5 || removed != 5 {
		t.Fatalf("got added=%d removed=%d", added, removed)
	}
}
