package config

import (
	"fmt"
	"strings"
)

type Review struct {
	MaxPerReview       int    `yaml:"maxPerReview"`
	SystemMessage      string `yaml:"systemMessage"`
	UserPromptTemplate string `yaml:"userPromptTemplate"`
}

func (r *Review) Validate() error {
	if r.MaxPerReview == 0 {
		return fmt.Errorf("review.maxPerReview is required")
	}
	if r.SystemMessage == "" {
		return fmt.Errorf("review.systemMessage is required")
	}
	// Guard against accidental missing fmt args in templates.
	if s := fmt.Sprintf(r.SystemMessage, r.MaxPerReview); s == "" || containsFmtError(s) {
		return fmt.Errorf("review.systemMessage is not a valid fmt template (expected maxPerReview placeholder like %%d)")
	}
	if r.UserPromptTemplate == "" {
		return fmt.Errorf("review.userPromptTemplate is required")
	}
	if s := fmt.Sprintf(r.UserPromptTemplate, "diff", "commits"); s == "" || containsFmtError(s) {
		return fmt.Errorf("review.userPromptTemplate is not a valid fmt template (expected placeholders for diff and commits comments)")
	}

	return nil
}

func containsFmtError(s string) bool {
	// fmt.Sprintf reports formatting problems as "%!<verb>(...)".
	return strings.Contains(s, "%!")
}
