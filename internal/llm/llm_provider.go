package llm

import (
	"context"
	"fmt"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	pkgllm "github.com/groall/upsource-ai-reviewer/pkg/llm"
)

// Provider is an interface for LLM completion providers.
type Provider interface {
	Completion(userPrompt, systemPrompt string) (string, error)
}

// PrefixCacheProvider is an optional extension interface for providers that can
// cache a stable prefix of the user prompt independently from the suffix.
//
// Implementations should treat userPromptPrefix as the cacheable part and
// userPromptSuffix as non-cacheable.
type PrefixCacheProvider interface {
	Provider
	CompletionWithPrefixCache(userPromptPrefix, userPromptSuffix, systemPrompt string) (string, error)
}

// createLLMProvider creates an LLM provider based on the configuration.
func createLLMProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	if cfg.Agent.Command != "" {
		provider, err := pkgllm.NewAgentCompletion(ctx, &pkgllm.AgentConfig{
			Command:        cfg.Agent.Command,
			Workdir:        cfg.Agent.Workdir,
			RequestTimeout: cfg.Agent.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Agent client: %w", err)
		}
		return provider, nil
	}

	if cfg.OpenAI.APIKey != "" {
		provider, err := pkgllm.NewOpenAICompletion(ctx, &pkgllm.OpenAIConfig{
			APIKey:         cfg.OpenAI.APIKey,
			Endpoint:       cfg.OpenAI.Endpoint,
			Model:          cfg.OpenAI.Model,
			MaxTokens:      cfg.OpenAI.MaxTokens,
			Temperature:    float32(cfg.OpenAI.Temperature),
			RequestTimeout: cfg.OpenAI.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
		}
		return provider, nil
	}

	if cfg.Gemini.APIKey != "" {
		provider, err := pkgllm.NewGeminiCompletion(ctx, &pkgllm.GeminiConfig{
			APIKey:    cfg.Gemini.APIKey,
			Model:     cfg.Gemini.Model,
			MaxTokens: int32(cfg.Gemini.MaxTokens),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		return provider, nil
	}

	if cfg.Anthropic.APIKey != "" {
		provider, err := pkgllm.NewAnthropicCompletion(ctx, &pkgllm.AnthropicConfig{
			APIKey:         cfg.Anthropic.APIKey,
			Model:          cfg.Anthropic.Model,
			MaxTokens:      cfg.Anthropic.MaxTokens,
			RequestTimeout: cfg.Anthropic.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
		}
		return provider, nil
	}

	return nil, fmt.Errorf("no LLM provider configured")

}
