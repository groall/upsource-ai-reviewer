package llm

import (
	"context"
	"fmt"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

// Provider is an interface for LLM completion providers.
type Provider interface {
	Completion(userPrompt, systemPrompt string) (string, error)
}

// PrefixCacheProvider is an optional extension interface for providers that can
// cache a stable prefix of the user prompt independently of the suffix.
//
// Implementations should treat userPromptPrefix as the cacheable part and
// userPromptSuffix as non-cacheable.
type PrefixCacheProvider interface {
	Provider
	CompletionWithPrefixCache(userPromptPrefix, userPromptSuffix, systemPrompt string) (string, error)
}

// CreateLLMProvider creates an LLM provider based on the configuration and a specific provider ID if provided.
func CreateLLMProvider(ctx context.Context, providers config.Providers, providerID string) (Provider, error) {
	switch providerID {
	case config.ProviderOpenAI:
		provider, err := NewOpenAICompletion(ctx, &OpenAIConfig{
			APIKey:         providers.OpenAI.APIKey,
			Endpoint:       providers.OpenAI.Endpoint,
			Model:          providers.OpenAI.Model,
			MaxTokens:      providers.OpenAI.MaxTokens,
			Temperature:    float32(providers.OpenAI.Temperature),
			RequestTimeout: providers.OpenAI.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
		}
		return provider, nil
	case config.ProviderGemini:
		provider, err := NewGeminiCompletion(ctx, &GeminiConfig{
			APIKey:         providers.Gemini.APIKey,
			Model:          providers.Gemini.Model,
			MaxTokens:      int32(providers.Gemini.MaxTokens),
			RequestTimeout: providers.Gemini.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		return provider, nil
	case config.ProviderAnthropic:
		provider, err := NewAnthropicCompletion(ctx, &AnthropicConfig{
			APIKey:         providers.Anthropic.APIKey,
			Model:          providers.Anthropic.Model,
			MaxTokens:      providers.Anthropic.MaxTokens,
			RequestTimeout: providers.Anthropic.RequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("no LLM provider configured")
	}
}
