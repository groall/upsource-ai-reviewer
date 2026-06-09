package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewValidate(t *testing.T) {
	t.Run("succeeds for valid config", func(t *testing.T) {
		r := validReview()
		require.NoError(t, r.Validate())
	})

	t.Run("fails when max per review is missing", func(t *testing.T) {
		r := validReview()
		r.MaxPerReview = 0
		require.EqualError(t, r.Validate(), "review.maxPerReview is required")
	})

	t.Run("fails when system message intro is missing", func(t *testing.T) {
		r := validReview()
		r.SystemMessageIntro = ""
		require.EqualError(t, r.Validate(), "review.systemMessageIntro is required")
	})

	t.Run("fails when system message template is invalid", func(t *testing.T) {
		r := validReview()
		r.SystemMessageGuidelines = "max 10"
		require.EqualError(t, r.Validate(), "review.systemMessage is not a valid template (expected maxPerReview placeholder like {{max_per_review}})")
	})

	t.Run("fails when user prompt template is missing", func(t *testing.T) {
		r := validReview()
		r.UserPromptTemplate = ""
		require.EqualError(t, r.Validate(), "review.userPromptTemplate is required")
	})

	t.Run("fails when user prompt template is invalid", func(t *testing.T) {
		r := validReview()
		r.UserPromptTemplate = "diffs: {{diffs}}"
		require.EqualError(t, r.Validate(), "review.userPromptTemplate is not a valid template (expected placeholder for messages like {{messages}})")
	})
}

func validReview() *Review {
	return &Review{
		MaxPerReview:              10,
		SystemMessageIntro:        "intro",
		SystemMessageGuidelines:   "max {{max_per_review}}",
		SystemMessageOutputFormat: "output",
		UserPromptTemplate:        "diffs: {{diffs}}\nmessages: {{messages}}",
	}
}
