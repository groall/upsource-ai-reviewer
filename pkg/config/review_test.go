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

	t.Run("fails when activeProvider is not set", func(t *testing.T) {
		r := validReview()
		r.ActiveProvider = ""
		require.EqualError(t, r.Validate(), "review.activeProvider is required")
	})

	t.Run("fails when activeProvider is invalid", func(t *testing.T) {
		r := validReview()
		r.ActiveProvider = "invalid"
		require.Error(t, r.Validate())
		require.Contains(t, r.Validate().Error(), "unknown provider ID")
	})

	t.Run("succeeds with agent provider", func(t *testing.T) {
		r := validReview()
		r.ActiveProvider = ProviderAgent
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with gemini provider", func(t *testing.T) {
		r := validReview()
		r.ActiveProvider = ProviderGemini
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with anthropic provider", func(t *testing.T) {
		r := validReview()
		r.ActiveProvider = ProviderAnthropic
		require.NoError(t, r.Validate())
	})

	t.Run("supports legacy single systemMessage", func(t *testing.T) {
		r := &Review{
			ActiveProvider:     ProviderOpenAI,
			MaxPerReview:       5,
			SystemMessage:      "legacy {{max_per_review}}",
			UserPromptTemplate: "diffs: {{diffs}}\nmessages: {{messages}}",
		}
		require.NoError(t, r.Validate())
	})

	t.Run("fails when systemMessageOutputFormat is missing in split mode", func(t *testing.T) {
		r := validReview()
		r.SystemMessageOutputFormat = ""
		require.EqualError(t, r.Validate(), "review.systemMessageOutputFormat is required")
	})

	t.Run("fails when system message lacks maxPerReview placeholder", func(t *testing.T) {
		r := validReview()
		r.SystemMessageGuidelines = "guidelines without placeholder"
		require.EqualError(t, r.Validate(), "review.systemMessage is not a valid template (expected maxPerReview placeholder like {{max_per_review}})")
	})

	t.Run("fails when user prompt lacks diffs placeholder", func(t *testing.T) {
		r := validReview()
		r.UserPromptTemplate = "messages: {{messages}}"
		require.EqualError(t, r.Validate(), "review.userPromptTemplate is not a valid template (expected placeholder for diff like {{diffs}})")
	})

	t.Run("succeeds with MaxPerReview of 1", func(t *testing.T) {
		r := validReview()
		r.MaxPerReview = 1
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with high MaxPerReview", func(t *testing.T) {
		r := validReview()
		r.MaxPerReview = 1000
		require.NoError(t, r.Validate())
	})
}

func validReview() *Review {
	return &Review{
		ActiveProvider:            ProviderOpenAI,
		MaxPerReview:              10,
		SystemMessageIntro:        "intro",
		SystemMessageGuidelines:   "max {{max_per_review}}",
		SystemMessageOutputFormat: "output",
		UserPromptTemplate:        "diffs: {{diffs}}\nmessages: {{messages}}",
	}
}
