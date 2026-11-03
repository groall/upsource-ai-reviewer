package review

import (
	"regexp"
	"strings"
)

// validateCommentsAgainstDiff checks whether each LLM-reported location points at
// an added line in the provided unified diff. Comments that cannot be placed
// inline are downgraded by clearing LineVerified and zeroing LineNumber.
func validateCommentsAgainstDiff(diff string, comments []*reviewComment) []*reviewComment {
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
	forEachAddedLine(diff, func(file string, line int, _ string) {
		if result[file] == nil {
			result[file] = make(map[int]bool)
		}
		result[file][line] = true
	})
	return result
}

func forEachAddedLine(diff string, visit func(file string, line int, content string)) {
	var currentFile string
	var newLine int
	var inHunk bool

	// Hunk headers carry the starting line number for the new side of the diff.
	hunkHeader := regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			currentFile = ""
			newLine = 0
			inHunk = false
		case !inHunk && strings.HasPrefix(line, "+++ "):
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
				inHunk = true
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
				visit(currentFile, newLine, strings.TrimSuffix(line[1:], "\r"))
				newLine++
			case ' ':
				newLine++
			case '-':
				// Removed lines belong only to the old side of the diff.
			}
		}
	}

}

// normalizeDiffPath converts unified diff paths to repository-relative paths.
func normalizeDiffPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "/dev/null" {
		return path
	}
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return strings.TrimLeft(path, "/")
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
