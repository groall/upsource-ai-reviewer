package review

import (
	"testing"
)

func TestNormalizeDiffPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a/file.go", "file.go"},
		{"b/file.go", "file.go"},
		{"a/dir/file.go", "dir/file.go"},
		{"b/dir/subdir/file.go", "dir/subdir/file.go"},
		{" a/file.go ", "file.go"},
		{"  b/file.go  ", "file.go"},
		{"file.go", "file.go"},
		{"/dev/null", "/dev/null"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeDiffPath(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"1", 1},
		{"42", 42},
		{"1234", 1234},
		{"999", 999},
		{"123abc", 123},
		{"abc", 0},
		{"", 0},
		{" ", 0},
		{"0000", 0},
		{"0001", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt(tt.input)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildNewFileLineIndex_EmptyDiff(t *testing.T) {
	result := buildNewFileLineIndex("")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestBuildNewFileLineIndex_SingleFile(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+func hello() {
 func main() {
+}
`

	result := buildNewFileLineIndex(diff)

	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}

	lines, ok := result["file.go"]
	if !ok {
		t.Fatalf("file.go not found in result")
	}

	// Lines 2 and 4 are added (have +)
	expectedAdded := []int{2, 4}
	for _, lineNum := range expectedAdded {
		if !lines[lineNum] {
			t.Errorf("line %d should be marked as added", lineNum)
		}
	}

	// Line 1 and 3 are context (have space), should not be marked as added
	if lines[1] {
		t.Errorf("line 1 should not be marked as added (context line)")
	}
	if lines[3] {
		t.Errorf("line 3 should not be marked as added (context line)")
	}
}

func TestBuildNewFileLineIndex_MultipleHunks(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
@@ -5,2 +6,3 @@
 line5
+line6
 line7
`

	result := buildNewFileLineIndex(diff)
	lines := result["file.go"]

	// First hunk: line 2 is added
	if !lines[2] {
		t.Error("line 2 should be added (first hunk)")
	}

	// Second hunk: line 7 is added
	if !lines[7] {
		t.Error("line 7 should be added (second hunk)")
	}
}

func TestBuildNewFileLineIndex_DeletedFile(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`

	result := buildNewFileLineIndex(diff)

	// Deleted files should not create an entry
	if _, ok := result["file.go"]; ok {
		t.Error("deleted file should not create entry")
	}
}

func TestBuildNewFileLineIndex_NewFile(t *testing.T) {
	diff := `diff --git a/newfile.go b/newfile.go
index 0000000..abc 100644
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {
`

	result := buildNewFileLineIndex(diff)

	lines, ok := result["newfile.go"]
	if !ok {
		t.Fatalf("newfile.go not found")
	}

	// All lines are added in new file
	for i := 1; i <= 3; i++ {
		if !lines[i] {
			t.Errorf("line %d should be added", i)
		}
	}
}

func TestBuildNewFileLineIndex_WithSubdirectories(t *testing.T) {
	diff := `diff --git a/pkg/module/file.go b/pkg/module/file.go
index abc..def 100644
--- a/pkg/module/file.go
+++ b/pkg/module/file.go
@@ -1,2 +1,3 @@
 package module
+func New() {}
 func main() {}
`

	result := buildNewFileLineIndex(diff)

	lines, ok := result["pkg/module/file.go"]
	if !ok {
		t.Fatalf("pkg/module/file.go not found")
	}

	if !lines[2] {
		t.Error("line 2 should be added")
	}
}

func TestValidateCommentsAgainstDiff_EmptyDiff(t *testing.T) {
	comments := []*reviewComment{
		{FilePath: "file.go", LineNumber: 1, LineVerified: true},
	}

	result := validateCommentsAgainstDiff("", comments)

	// Should return comments unchanged when diff is empty
	if !result[0].LineVerified {
		t.Error("comments should remain unchanged with empty diff")
	}
}

func TestValidateCommentsAgainstDiff_EmptyComments(t *testing.T) {
	diff := "diff content"
	var comments []*reviewComment

	result := validateCommentsAgainstDiff(diff, comments)

	if len(result) != 0 {
		t.Error("expected empty result")
	}
}

func TestValidateCommentsAgainstDiff_ValidLineNumbers(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+func hello() {
 func main() {
+}
`

	comments := []*reviewComment{
		{FilePath: "file.go", LineNumber: 2, Comment: "comment1"},
		{FilePath: "file.go", LineNumber: 4, Comment: "comment2"},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	// Lines 2 and 4 are added, should be verified
	if !result[0].LineVerified {
		t.Error("line 2 should be verified")
	}
	if !result[1].LineVerified {
		t.Error("line 4 should be verified")
	}
}

func TestValidateCommentsAgainstDiff_InvalidLineNumbers(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+func hello() {
 func main() {
+}
`

	comments := []*reviewComment{
		{FilePath: "file.go", LineNumber: 1, Comment: "not added"},
		{FilePath: "file.go", LineNumber: 3, Comment: "not added"},
		{FilePath: "file.go", LineNumber: 100, Comment: "doesn't exist"},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	// None should be verified, all should have LineNumber set to 0
	for i, c := range result {
		if c.LineVerified {
			t.Errorf("comment %d should not be verified", i)
		}
		if c.LineNumber != 0 {
			t.Errorf("comment %d should have LineNumber=0, got %d", i, c.LineNumber)
		}
	}
}

func TestValidateCommentsAgainstDiff_NonexistentFile(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
`

	comments := []*reviewComment{
		{FilePath: "otherfile.go", LineNumber: 1},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	if result[0].LineVerified {
		t.Error("comment in non-existent file should not be verified")
	}
	if result[0].LineNumber != 0 {
		t.Error("line number should be reset to 0")
	}
}

func TestValidateCommentsAgainstDiff_ZeroLineNumber(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
`

	comments := []*reviewComment{
		{FilePath: "file.go", LineNumber: 0},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	if result[0].LineVerified {
		t.Error("zero line number should not be verified")
	}
	if result[0].LineNumber != 0 {
		t.Error("line number should remain 0")
	}
}

func TestValidateCommentsAgainstDiff_NegativeLineNumber(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
`

	comments := []*reviewComment{
		{FilePath: "file.go", LineNumber: -5},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	if result[0].LineVerified {
		t.Error("negative line number should not be verified")
	}
	if result[0].LineNumber != 0 {
		t.Error("line number should be reset to 0")
	}
}

func TestValidateCommentsAgainstDiff_PathNormalization(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
`

	comments := []*reviewComment{
		// Comment uses path without a/b prefix
		{FilePath: "file.go", LineNumber: 2},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	if !result[0].LineVerified {
		t.Error("path should be normalized for matching")
	}
}

func TestValidateCommentsAgainstDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/file1.go b/file1.go
index abc..def 100644
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,2 @@
 line1
+line2
diff --git a/file2.go b/file2.go
index abc..def 100644
--- a/file2.go
+++ b/file2.go
@@ -1,1 +1,2 @@
 line1
+line2
`

	comments := []*reviewComment{
		{FilePath: "file1.go", LineNumber: 2},
		{FilePath: "file2.go", LineNumber: 2},
	}

	result := validateCommentsAgainstDiff(diff, comments)

	if !result[0].LineVerified {
		t.Error("file1.go line 2 should be verified")
	}
	if !result[1].LineVerified {
		t.Error("file2.go line 2 should be verified")
	}
}
