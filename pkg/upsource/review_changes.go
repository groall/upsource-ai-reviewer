package upsource

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/groall/upsource-go-client/client"
	"github.com/pmezard/go-difflib/difflib"
)

// GetReviewChanges returns the review diff and its revision commit messages from Upsource.
func GetReviewChanges(ctx context.Context, upsourceClient *client.Client, review *Review) (string, string, error) {
	var changes strings.Builder
	for _, fileSummary := range review.filesDiffSummary {
		file := fileSummary.File
		oldFileName, newFileName, diffText, err := getReviewFileDiff(ctx, upsourceClient, review, file)
		if err != nil {
			return "", "", err
		}
		if strings.TrimSpace(diffText) == "" {
			continue
		}
		appendReviewFileDiff(&changes, file.FileName, oldFileName, newFileName, diffText)
	}

	revisions, err := upsourceClient.GetRevisionsInReview(ctx, review.review.ReviewID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get Upsource review revisions: %w", err)
	}

	var messages strings.Builder
	for _, revision := range revisions.AllRevisions.Revision {
		messages.WriteString(fmt.Sprintf("Commit %s:\n%s\n\n", revision.ShortRevisionID, revision.RevisionCommitMessage))
	}

	if changes.Len() == 0 {
		return "", messages.String(), fmt.Errorf("no diffs found in Upsource review %s", review.review.ReviewID.ReviewID)
	}

	return changes.String(), messages.String(), nil
}

func getReviewFileDiff(ctx context.Context, upsourceClient *client.Client, review *Review, file client.FileInRevisionDTO) (string, string, string, error) {
	diffRequest := client.FileInReviewDiffRequestDTO{
		File: client.FileInReviewDTO{
			ReviewID: review.review.ReviewID,
			File:     file,
		},
		Revisions:    &client.RevisionsSetDTO{SelectAll: boolPointer(true)},
		ContextLines: 3,
	}

	diffText, oldFileName, newFileName, err := buildReviewFileDiff(ctx, upsourceClient, diffRequest, file.FileName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to build Upsource diff for %s: %w", file.FileName, err)
	}
	return oldFileName, newFileName, diffText, nil
}

func appendReviewFileDiff(changes *strings.Builder, originalFileName, oldFileName, newFileName, diffText string) {
	if isMinifiedFile(originalFileName) || isMinifiedFile(oldFileName) || isMinifiedFile(newFileName) {
		// Keep generated one-line files out of the review prompt while retaining the change size.
		added, removed := countChangedCharacters(diffText)
		fileName := newFileName
		if newFileName == "/dev/null" {
			fileName = oldFileName
		}
		if oldFileName != "/dev/null" && newFileName != "/dev/null" && oldFileName != newFileName {
			fileName = oldFileName + " -> " + newFileName
		}
		changes.WriteString(fmt.Sprintf("Minified file changed: %s (added %d characters, removed %d characters); diff omitted.\n\n", fileName, added, removed))
		return
	}

	writeReviewFileDiff(changes, oldFileName, newFileName, replaceLongSourceLines(diffText))
}

func replaceLongSourceLines(diffText string) string {
	const maxSourceLineLength = 256
	const replacement = "[source line omitted: exceeds 256 characters]"

	// Preserve each unified-diff marker while replacing only its oversized source text.
	var result strings.Builder
	for _, line := range strings.SplitAfter(diffText, "\n") {
		lineEnding := ""
		if strings.HasSuffix(line, "\r\n") {
			lineEnding = "\r\n"
			line = strings.TrimSuffix(line, lineEnding)
		} else if strings.HasSuffix(line, "\n") {
			lineEnding = "\n"
			line = strings.TrimSuffix(line, lineEnding)
		}

		if len(line) > 0 && (line[0] == '+' || line[0] == '-' || line[0] == ' ') &&
			!strings.HasPrefix(line, "+++ ") && !strings.HasPrefix(line, "--- ") &&
			len([]rune(line[1:])) > maxSourceLineLength {
			line = line[:1] + replacement
		}

		result.WriteString(line)
		result.WriteString(lineEnding)
	}

	return result.String()
}

func isMinifiedFile(fileName string) bool {
	return strings.Contains(strings.ToLower(path.Base(fileName)), ".min.")
}

func countChangedCharacters(diffText string) (added, removed int) {
	inHunk := false
	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			inHunk = true
			continue
		}
		if !inHunk || len(line) == 0 {
			continue
		}
		line = strings.TrimSuffix(line, "\r")
		switch line[0] {
		case '+':
			added += len([]rune(line[1:]))
		case '-':
			removed += len([]rune(line[1:]))
		}
	}
	return added, removed
}

func buildReviewFileDiff(ctx context.Context, upsourceClient *client.Client, request client.FileInReviewDiffRequestDTO, fallbackFileName string) (string, string, string, error) {
	// Fetch both file revisions and build unified hunks from their contents.
	diff, err := upsourceClient.GetFileInReviewSummaryDiff(ctx, request)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get non-inline diff: %w", err)
	}
	if diff.LeftFile == nil && diff.RightFile == nil {
		return "", "", "", errors.New("upsource returned no old or new file reference")
	}

	oldName, newName := fallbackFileName, fallbackFileName
	oldContent, newContent := "", ""
	oldIsText, newIsText := true, true

	if diff.LeftFile == nil {
		oldName = "/dev/null"
	}

	if diff.RightFile == nil {
		newName = "/dev/null"
	}

	if diff.LeftFile != nil {
		oldName = diff.LeftFile.FileName
		content, contentErr := upsourceClient.GetFileContent(ctx, *diff.LeftFile)
		if contentErr != nil {
			return "", "", "", fmt.Errorf("failed to load old file contents: %w", contentErr)
		}
		if content.FileContent != nil {
			oldContent = content.FileContent.Text
		}
		oldIsText = content.ContentType.IsText
	}

	if diff.RightFile != nil {
		newName = diff.RightFile.FileName
		content, contentErr := upsourceClient.GetFileContent(ctx, *diff.RightFile)
		if contentErr != nil {
			return "", "", "", fmt.Errorf("failed to load new file contents: %w", contentErr)
		}
		if content.FileContent != nil {
			newContent = content.FileContent.Text
		}
		newIsText = content.ContentType.IsText
	}

	if diff.LeftFile != nil && !oldIsText || diff.RightFile != nil && !newIsText {
		return "Binary or non-text file differs; contents omitted.\n", oldName, newName, nil
	}

	oldLines, newLines := splitDiffLines(oldContent), splitDiffLines(newContent)

	diffText, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: oldLines, B: newLines, Context: 3})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to build unified diff: %w", err)
	}

	if diffText == "" {
		return "", oldName, newName, nil
	}

	return diffText, oldName, newName, nil
}

func splitDiffLines(content string) []string {
	if content == "" {
		return nil
	}
	// Preserve a missing final newline in the patch and in line comparisons.
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if !strings.HasSuffix(lines[len(lines)-1], "\n") {
		lines[len(lines)-1] += "\n\\ No newline at end of file\n"
	}
	return lines
}

func writeReviewFileDiff(changes *strings.Builder, oldFileName, newFileName, diffText string) {
	oldHeaderName, newHeaderName := oldFileName, newFileName
	oldPath, newPath := "a/"+oldFileName, "b/"+newFileName
	if oldFileName == "/dev/null" {
		oldHeaderName = newFileName
		oldPath = "/dev/null"
	}

	if newFileName == "/dev/null" {
		newHeaderName = oldFileName
		newPath = "/dev/null"
	}

	changes.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", oldHeaderName, newHeaderName))

	if !strings.HasPrefix(diffText, "--- ") {
		changes.WriteString(fmt.Sprintf("--- %s\n+++ %s\n", oldPath, newPath))
	}

	changes.WriteString(strings.TrimLeft(diffText, "\n"))

	if !strings.HasSuffix(diffText, "\n") {
		changes.WriteByte('\n')
	}

	changes.WriteByte('\n')
}

func boolPointer(value bool) *bool {
	return &value
}
