package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepliesValidate(t *testing.T) {
	t.Run("succeeds when disabled", func(t *testing.T) {
		r := &Replies{Enabled: false}
		require.NoError(t, r.Validate())
	})

	t.Run("fails when enabled but maxPerThread is 0", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   0,
			SystemMessage:  "reply",
			ActiveProvider: ProviderOpenAI,
		}
		require.EqualError(t, r.Validate(), "replies.maxPerThread must be > 0 when replies.enabled is true")
	})

	t.Run("fails when enabled but maxPerThread is negative", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   -1,
			SystemMessage:  "reply",
			ActiveProvider: ProviderOpenAI,
		}
		require.EqualError(t, r.Validate(), "replies.maxPerThread must be > 0 when replies.enabled is true")
	})

	t.Run("fails when enabled but systemMessage is empty", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "",
			ActiveProvider: ProviderOpenAI,
		}
		require.EqualError(t, r.Validate(), "replies.systemMessage is required when replies.enabled is true")
	})

	t.Run("fails when enabled but activeProvider is not set", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: "",
		}
		require.EqualError(t, r.Validate(), "replies.activeProvider is required")
	})

	t.Run("fails when enabled with invalid provider", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: "invalid",
		}
		require.Error(t, r.Validate())
		require.Contains(t, r.Validate().Error(), "unknown provider ID")
	})

	t.Run("fails when activeProvider is agent", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: ProviderAgent,
		}
		require.EqualError(t, r.Validate(), "replies.activeProvider cannot be agent")
	})

	t.Run("succeeds with openai provider", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: ProviderOpenAI,
		}
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with gemini provider", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: ProviderGemini,
		}
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with anthropic provider", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   1,
			SystemMessage:  "reply",
			ActiveProvider: ProviderAnthropic,
		}
		require.NoError(t, r.Validate())
	})

	t.Run("succeeds with high maxPerThread value", func(t *testing.T) {
		r := &Replies{
			Enabled:        true,
			MaxPerThread:   100,
			SystemMessage:  "reply",
			ActiveProvider: ProviderOpenAI,
		}
		require.NoError(t, r.Validate())
	})
}
