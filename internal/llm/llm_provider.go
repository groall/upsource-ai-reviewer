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

type providerFactory interface {
	Enabled(cfg *config.Config) bool
	Build(ctx context.Context, cfg *config.Config) (Provider, error)
}

type orderedProviderFactory struct {
	enabled func(cfg *config.Config) bool
	build   func(ctx context.Context, cfg *config.Config) (Provider, error)
}

func (f orderedProviderFactory) Enabled(cfg *config.Config) bool {
	return f.enabled(cfg)
}

func (f orderedProviderFactory) Build(ctx context.Context, cfg *config.Config) (Provider, error) {
	return f.build(ctx, cfg)
}

var providerFactories = []providerFactory{
	orderedProviderFactory{
		enabled: func(cfg *config.Config) bool {
			return cfg.AgentEnabled()
		},
		build: func(ctx context.Context, cfg *config.Config) (Provider, error) {
			provider, err := pkgllm.NewAgentCompletion(ctx, &pkgllm.AgentConfig{
				Command:        cfg.Agent.Command,
				Workdir:        cfg.Agent.Workdir,
				RequestTimeout: cfg.Agent.RequestTimeout,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Agent client: %w", err)
			}
			return provider, nil
		},
	},
	orderedProviderFactory{
		enabled: func(cfg *config.Config) bool {
			return cfg.OpenAI.APIKey != ""
		},
		build: func(ctx context.Context, cfg *config.Config) (Provider, error) {
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
		},
	},
	orderedProviderFactory{
		enabled: func(cfg *config.Config) bool {
			return cfg.Gemini.APIKey != ""
		},
		build: func(ctx context.Context, cfg *config.Config) (Provider, error) {
			provider, err := pkgllm.NewGeminiCompletion(ctx, &pkgllm.GeminiConfig{
				APIKey:    cfg.Gemini.APIKey,
				Model:     cfg.Gemini.Model,
				MaxTokens: int32(cfg.Gemini.MaxTokens),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Gemini client: %w", err)
			}
			return provider, nil
		},
	},
	orderedProviderFactory{
		enabled: func(cfg *config.Config) bool {
			return cfg.Anthropic.APIKey != ""
		},
		build: func(ctx context.Context, cfg *config.Config) (Provider, error) {
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
		},
	},
}

// createLLMProvider creates an LLM provider based on the configuration.
func createLLMProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	for _, factory := range providerFactories {
		if !factory.Enabled(cfg) {
			continue
		}

		provider, err := factory.Build(ctx, cfg)
		if err != nil {
			return nil, err
		}

		return provider, nil
	}

	return nil, fmt.Errorf("no LLM provider configured")
}
