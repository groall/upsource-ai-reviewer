package llm

import (
	"context"
	"fmt"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/llm"
)

// Provider is an interface for LLM completion providers.
type Provider interface {
	Completion(userPrompt, systemPrompt string) (string, error)
}

// createLLMProvider creates an LLM provider based on the configuration.
func createLLMProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	if cfg.OpenAI.APIKey != "" {
		provider, err := llm.NewOpenAICompletion(ctx, &llm.OpenAIConfig{
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
		provider, err := llm.NewGeminiCompletion(ctx, &llm.GeminiConfig{
			APIKey:    cfg.Gemini.APIKey,
			Model:     cfg.Gemini.Model,
			MaxTokens: int32(cfg.Gemini.MaxTokens),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		return provider, nil
	}

	return nil, fmt.Errorf("no LLM API key provided")

}
