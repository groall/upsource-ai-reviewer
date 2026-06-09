package config

import (
	"fmt"
	"strings"
)

type Review struct {
	MaxPerReview int `yaml:"maxPerReview"`

	// SystemMessage is a legacy, single-block system message template for reviews.
	// Prefer the split fields below.
	SystemMessage string `yaml:"systemMessage"`

	// Split system message for reviews.
	SystemMessageIntro        string `yaml:"systemMessageIntro"`
	SystemMessageGuidelines   string `yaml:"systemMessageGuidelines"`
	SystemMessageOutputFormat string `yaml:"systemMessageOutputFormat"`

	UserPromptTemplate string `yaml:"userPromptTemplate"`
}

func (r *Review) Validate() error {
	if r.MaxPerReview == 0 {
		return fmt.Errorf("review.maxPerReview is required")
	}

	if r.usesSplitSystemMessage() {
		if r.SystemMessageIntro == "" {
			return fmt.Errorf("review.systemMessageIntro is required")
		}
		if r.SystemMessageGuidelines == "" {
			return fmt.Errorf("review.systemMessageGuidelines is required")
		}
		if r.SystemMessageOutputFormat == "" {
			return fmt.Errorf("review.systemMessageOutputFormat is required")
		}
	} else if r.SystemMessage == "" {
		return fmt.Errorf("review.systemMessage is required")
	}

	// Guard against accidental missing fmt args in templates.
	if !containsMaxPerReviewPlaceholder(r.SystemMessageTemplate()) {
		return fmt.Errorf("review.systemMessage is not a valid template (expected maxPerReview placeholder like {{max_per_review}})")
	}
	if r.UserPromptTemplate == "" {
		return fmt.Errorf("review.userPromptTemplate is required")
	}
	if !containsDiffsPlaceholder(r.UserPromptTemplate) {
		return fmt.Errorf("review.userPromptTemplate is not a valid template (expected placeholder for diff like {{diffs}})")
	}
	if !containsMessagesPlaceholder(r.UserPromptTemplate) {
		return fmt.Errorf("review.userPromptTemplate is not a valid template (expected placeholder for messages like {{messages}})")
	}

	return nil
}

func (r *Review) SystemMessageTemplate() string {
	if !r.usesSplitSystemMessage() {
		return r.SystemMessage
	}

	parts := []string{
		strings.TrimRight(r.SystemMessageIntro, "\n"),
		strings.TrimRight(r.SystemMessageGuidelines, "\n"),
		strings.TrimRight(r.SystemMessageOutputFormat, "\n"),
	}
	return strings.Join(parts, "\n\n")
}

func (r *Review) usesSplitSystemMessage() bool {
	return r.SystemMessageIntro != "" || r.SystemMessageGuidelines != "" || r.SystemMessageOutputFormat != ""
}

func containsMaxPerReviewPlaceholder(s string) bool {
	return strings.Contains(s, "{{max_per_review}}")
}

func containsDiffsPlaceholder(s string) bool {
	return strings.Contains(s, "{{diffs}}")
}

func containsMessagesPlaceholder(s string) bool {
	return strings.Contains(s, "{{messages}}")
}
