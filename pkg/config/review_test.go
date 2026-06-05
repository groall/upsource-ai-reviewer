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

	t.Run("fails when post inline is missing", func(t *testing.T) {
		r := validReview()
		r.PostInLine = ""
		require.EqualError(t, r.Validate(), "review.postInLine is required")
	})

	t.Run("fails when post inline value is invalid", func(t *testing.T) {
		r := validReview()
		r.PostInLine = "critical"
		require.EqualError(t, r.Validate(), "review.postInLine must be one of: high, mid, low, none")
	})

	t.Run("fails when system message is missing", func(t *testing.T) {
		r := validReview()
		r.SystemMessage = ""
		require.EqualError(t, r.Validate(), "review.systemMessage is required")
	})

	t.Run("fails when system message template is invalid", func(t *testing.T) {
		r := validReview()
		r.SystemMessage = "max %s"
		require.EqualError(t, r.Validate(), "review.systemMessage is not a valid fmt template (expected maxPerReview placeholder like %d)")
	})

	t.Run("fails when user prompt template is missing", func(t *testing.T) {
		r := validReview()
		r.UserPromptTemplate = ""
		require.EqualError(t, r.Validate(), "review.userPromptTemplate is required")
	})

	t.Run("fails when user prompt template is invalid", func(t *testing.T) {
		r := validReview()
		r.UserPromptTemplate = "%d %d"
		require.EqualError(t, r.Validate(), "review.userPromptTemplate is not a valid fmt template (expected placeholders for diff and commits comments)")
	})
}

func TestContainsFmtError(t *testing.T) {
	require.False(t, containsFmtError("ok"))
	require.True(t, containsFmtError("value %!d(string=test)"))
}

func validReview() *Review {
	return &Review{
		MaxPerReview:       10,
		PostInLine:         "high",
		SystemMessage:      "max %d",
		UserPromptTemplate: "%s %s",
	}
}
