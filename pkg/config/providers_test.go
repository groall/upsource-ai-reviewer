package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProvidersValidate(t *testing.T) {
	t.Run("fails when no provider is configured", func(t *testing.T) {
		providers := &Providers{}
		err := providers.Validate()
		require.EqualError(t, err, "either providers.openai.apiKey, providers.gemini.apiKey, providers.anthropic.apiKey, or providers.agent.command is required")
	})

	t.Run("fails when openai is enabled without model", func(t *testing.T) {
		providers := &Providers{OpenAI: OpenAI{APIKey: "key"}}
		err := providers.Validate()
		require.EqualError(t, err, "providers.openai.model is required when providers.openai.apiKey is set")
	})

	t.Run("fails when gemini is enabled without model", func(t *testing.T) {
		providers := &Providers{Gemini: Gemini{APIKey: "key"}}
		err := providers.Validate()
		require.EqualError(t, err, "providers.gemini.model is required when providers.gemini.apiKey is set")
	})

	t.Run("fails when anthropic is enabled without model", func(t *testing.T) {
		providers := &Providers{Anthropic: Anthropic{APIKey: "key"}}
		err := providers.Validate()
		require.EqualError(t, err, "providers.anthropic.model is required when providers.anthropic.apiKey is set")
	})

	t.Run("allows each provider", func(t *testing.T) {
		testCases := []struct {
			name      string
			providers Providers
		}{
			{
				name:      "openai",
				providers: Providers{OpenAI: OpenAI{APIKey: "key", Model: "gpt-5-mini"}},
			},
			{
				name:      "gemini",
				providers: Providers{Gemini: Gemini{APIKey: "key", Model: "gemini-2.5-flash"}},
			},
			{
				name:      "anthropic",
				providers: Providers{Anthropic: Anthropic{APIKey: "key", Model: "claude-opus-4-1"}},
			},
			{
				name:      "agent",
				providers: Providers{Agent: Agent{Command: "codex"}},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				providers := tc.providers
				require.NoError(t, providers.Validate())
			})
		}
	})
}

func TestProvidersAgentEnabled(t *testing.T) {
	t.Run("returns false for empty command", func(t *testing.T) {
		providers := Providers{}
		require.False(t, providers.AgentEnabled())
	})

	t.Run("returns false for whitespace command", func(t *testing.T) {
		providers := Providers{
			Agent: Agent{Command: "   "},
		}
		require.False(t, providers.AgentEnabled())
	})

	t.Run("returns true for non-empty command", func(t *testing.T) {
		providers := Providers{
			Agent: Agent{Command: "codex"},
		}
		require.True(t, providers.AgentEnabled())
	})
}

func TestProvidersActiveLLMProvider(t *testing.T) {
	t.Run("returns unknown when no providers are configured", func(t *testing.T) {
		providers := Providers{}
		require.Equal(t, unknownLLMProvider, providers.ActiveLLMProvider())
	})

	t.Run("prefers agent over all other providers", func(t *testing.T) {
		providers := Providers{
			Agent:     Agent{Command: "codex"},
			OpenAI:    OpenAI{APIKey: "openai"},
			Gemini:    Gemini{APIKey: "gemini"},
			Anthropic: Anthropic{APIKey: "anthropic"},
		}
		require.Equal(t, "agent", providers.ActiveLLMProvider())
	})

	t.Run("prefers openai over gemini and anthropic", func(t *testing.T) {
		providers := Providers{
			OpenAI:    OpenAI{APIKey: "openai", Model: "gpt-5-mini"},
			Gemini:    Gemini{APIKey: "gemini", Model: "gemini-2.5-flash"},
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "openai", providers.ActiveLLMProvider())
	})

	t.Run("prefers gemini over anthropic when openai is not configured", func(t *testing.T) {
		providers := Providers{
			Gemini:    Gemini{APIKey: "gemini", Model: "gemini-2.5-flash"},
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "gemini", providers.ActiveLLMProvider())
	})

	t.Run("returns anthropic when only anthropic is configured", func(t *testing.T) {
		providers := Providers{
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "anthropic", providers.ActiveLLMProvider())
	})

	t.Run("ignores whitespace-only API keys", func(t *testing.T) {
		providers := Providers{
			OpenAI: OpenAI{
				APIKey: "   ",
				Model:  "gpt-5-mini",
			},
		}
		require.Equal(t, unknownLLMProvider, providers.ActiveLLMProvider())
	})
}
