package review

import (
	"testing"

	"github.com/groall/upsource-ai-reviewer/internal/llm"
)

func TestSortAndCapComments(t *testing.T) {
	comments := []*llm.ReviewComment{
		{Severity: "medium", FilePath: "b.go", LineNumber: 2, LineVerified: true, Comment: "m1"},
		{Severity: "high", FilePath: "a.go", LineNumber: 0, LineVerified: false, Comment: "h0"},
		{Severity: "HIGH", FilePath: "a.go", LineNumber: 10, LineVerified: true, Comment: "h1"},
		{Severity: "low", FilePath: "a.go", LineNumber: 1, LineVerified: true, Comment: "l1"},
		{Severity: "medium", FilePath: "a.go", LineNumber: 0, LineVerified: false, Comment: "m0"},
	}

	got := sortAndCapComments(comments, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(got))
	}

	if got[0].Comment != "h1" {
		t.Fatalf("expected first comment h1, got %q", got[0].Comment)
	}
	if got[1].Comment != "h0" {
		t.Fatalf("expected second comment h0, got %q", got[1].Comment)
	}
	if got[2].Comment != "m1" {
		t.Fatalf("expected third comment m1, got %q", got[2].Comment)
	}
}
