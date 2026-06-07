package llm

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestGeminiCacheKey(t *testing.T) {
	t.Run("same inputs produce same key", func(t *testing.T) {
		left := geminiCacheKey("gemini-2.5-flash", "system", "prefix")
		right := geminiCacheKey("gemini-2.5-flash", "system", "prefix")
		require.Equal(t, left, right)
	})

	t.Run("different inputs produce different key", func(t *testing.T) {
		base := geminiCacheKey("gemini-2.5-flash", "system", "prefix")
		require.NotEqual(t, base, geminiCacheKey("gemini-2.5-flash", "system 2", "prefix"))
		require.NotEqual(t, base, geminiCacheKey("gemini-2.5-flash", "system", "prefix 2"))
	})
}

func TestIsGeminiCachedContentInvalidError(t *testing.T) {
	t.Run("matches 404 API errors", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", genai.APIError{Code: 404, Message: "not found"})
		require.True(t, isGeminiCachedContentInvalidError(err))
	})

	t.Run("matches cached-content invalid message", func(t *testing.T) {
		err := errors.New("Error 400, Message: CachedContent is invalid or expired")
		require.True(t, isGeminiCachedContentInvalidError(err))
	})

	t.Run("does not match unrelated errors", func(t *testing.T) {
		err := errors.New("network timeout")
		require.False(t, isGeminiCachedContentInvalidError(err))
	})
}
