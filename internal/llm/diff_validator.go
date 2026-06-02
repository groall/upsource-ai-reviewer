package llm

import (
	"regexp"
	"strings"
)

// validateCommentsAgainstDiff checks whether each LLM-reported location points at
// an added line in the provided unified diff. Comments that cannot be placed
// inline are downgraded by clearing LineVerified and zeroing LineNumber.
func validateCommentsAgainstDiff(diff string, comments []*ReviewComment) []*ReviewComment {
	if diff == "" || len(comments) == 0 {
		return comments
	}

	fileLines := buildNewFileLineIndex(diff)

	for _, c := range comments {
		c.LineVerified = false
		if c.LineNumber <= 0 {
			c.LineNumber = 0
			continue
		}

		path := normalizeDiffPath(c.FilePath)
		if lines, ok := fileLines[path]; ok {
			if lines[c.LineNumber] {
				c.LineVerified = true
				continue
			}
		}

		// Not found in diff; set to 0 to indicate unknown.
		c.LineNumber = 0
	}

	return comments
}

func buildNewFileLineIndex(diff string) map[string]map[int]bool {
	result := make(map[string]map[int]bool)

	var currentFile string
	var newLine int

	// Hunk headers carry the starting line number for the new side of the diff.
	hunkHeader := regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			currentFile = ""
			newLine = 0
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if path == "/dev/null" {
				currentFile = ""
				newLine = 0
				continue
			}
			currentFile = normalizeDiffPath(path)
		case strings.HasPrefix(line, "@@ "):
			if currentFile == "" {
				continue
			}
			matches := hunkHeader.FindStringSubmatch(line)
			if len(matches) == 2 {
				newLine = parseInt(matches[1])
			}
		default:
			if currentFile == "" || newLine == 0 {
				continue
			}
			if len(line) == 0 {
				continue
			}

			switch line[0] {
			case '+':
				// Only added lines are considered safe targets for inline comments.
				if _, ok := result[currentFile]; !ok {
					result[currentFile] = make(map[int]bool)
				}
				result[currentFile][newLine] = true
				newLine++
			case ' ':
				newLine++
			case '-':
				// Removed lines belong only to the old side of the diff.
			}
		}
	}

	return result
}

// normalizeDiffPath converts unified diff paths to repository-relative paths.
func normalizeDiffPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return path
}

func parseInt(s string) int {
	v := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return v
		}
		v = v*10 + int(r-'0')
	}
	return v
}
