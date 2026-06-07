package llm

import (
	"context"
	"testing"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	pkgllm "github.com/groall/upsource-ai-reviewer/pkg/llm"
	"github.com/stretchr/testify/require"
)

func TestCreateLLMProviderPrefersAgent(t *testing.T) {
	cfg := &config.Config{
		Providers: config.Providers{
			Agent: config.Agent{
				Command: "echo ok",
			},
			OpenAI: config.OpenAI{
				APIKey: "openai",
				Model:  "gpt-5-mini",
			},
		},
	}

	provider, err := createLLMProvider(context.Background(), cfg)
	require.NoError(t, err)

	_, isAgent := provider.(*pkgllm.AgentCompletion)
	require.True(t, isAgent)
}

func TestCreateLLMProviderUsesOpenAIWhenAgentDisabled(t *testing.T) {
	cfg := &config.Config{
		Providers: config.Providers{
			OpenAI: config.OpenAI{
				APIKey: "openai",
				Model:  "gpt-5-mini",
			},
			Gemini: config.Gemini{
				APIKey: "gemini",
				Model:  "gemini-2.5-flash",
			},
		},
	}

	provider, err := createLLMProvider(context.Background(), cfg)
	require.NoError(t, err)

	_, isOpenAI := provider.(*pkgllm.OpenAICompletion)
	require.True(t, isOpenAI)
}

func TestCreateLLMProviderReturnsErrorWhenNoProviderConfigured(t *testing.T) {
	cfg := &config.Config{}
	provider, err := createLLMProvider(context.Background(), cfg)
	require.Nil(t, provider)
	require.EqualError(t, err, "no LLM provider configured")
}
