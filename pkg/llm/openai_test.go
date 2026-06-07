package llm

import (
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAIBaseURL(t *testing.T) {
	t.Run("uses default when endpoint is empty", func(t *testing.T) {
		require.Equal(t, openai.DefaultConfig("").BaseURL, normalizeOpenAIBaseURL(""))
	})

	t.Run("keeps v1 base URL", func(t *testing.T) {
		require.Equal(t, "https://api.openai.com/v1", normalizeOpenAIBaseURL("https://api.openai.com/v1"))
	})

	t.Run("normalizes completions endpoint to v1", func(t *testing.T) {
		require.Equal(t, "https://api.openai.com/v1", normalizeOpenAIBaseURL("https://api.openai.com/v1/chat/completions"))
	})

	t.Run("appends v1 when missing", func(t *testing.T) {
		require.Equal(t, "https://example.com/v1", normalizeOpenAIBaseURL("https://example.com"))
	})
}
