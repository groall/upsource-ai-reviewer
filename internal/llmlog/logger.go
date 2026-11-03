package llmlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Logger persists each LLM invocation as a matching input/output file pair.
type Logger struct {
	directory string
	mu        sync.Mutex
}

func New(enabled bool, directory string) *Logger {
	if !enabled {
		return nil
	}
	return &Logger{directory: directory}
}

// Save writes the complete prompt and response for one review attempt.
func (l *Logger) Save(reviewNumber, input, output string) error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(l.directory, 0o750); err != nil {
		return fmt.Errorf("create LLM log directory: %w", err)
	}

	reviewNumber = safeReviewNumber(reviewNumber)
	prefix := "review_" + reviewNumber + "_try_"
	entries, err := os.ReadDir(l.directory)
	if err != nil {
		return fmt.Errorf("read LLM log directory: %w", err)
	}
	nextAttempt := 1
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		separator := strings.IndexByte(rest, '_')
		if separator < 0 {
			continue
		}
		attempt, parseErr := strconv.Atoi(rest[:separator])
		if parseErr == nil && attempt >= nextAttempt {
			nextAttempt = attempt + 1
		}
	}

	base := fmt.Sprintf("%s%03d", prefix, nextAttempt)
	if err := os.WriteFile(filepath.Join(l.directory, base+"_input.txt"), []byte(input), 0o600); err != nil {
		return fmt.Errorf("write LLM input log: %w", err)
	}
	if err := os.WriteFile(filepath.Join(l.directory, base+"_output.txt"), []byte(output), 0o600); err != nil {
		return fmt.Errorf("write LLM output log: %w", err)
	}
	return nil
}

var unsafeReviewNumber = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeReviewNumber(reviewNumber string) string {
	safe := unsafeReviewNumber.ReplaceAllString(reviewNumber, "_")
	safe = strings.Trim(safe, "._-")
	if safe == "" {
		return "unknown"
	}
	return safe
}
