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
	Enabled(providers config.Providers) bool
	Build(ctx context.Context, providers config.Providers) (Provider, error)
}

type orderedProviderFactory struct {
	enabled func(providers config.Providers) bool
	build   func(ctx context.Context, providers config.Providers) (Provider, error)
}

func (f orderedProviderFactory) Enabled(providers config.Providers) bool {
	return f.enabled(providers)
}

func (f orderedProviderFactory) Build(ctx context.Context, providers config.Providers) (Provider, error) {
	return f.build(ctx, providers)
}

var providerFactories = []providerFactory{
	orderedProviderFactory{
		enabled: func(providers config.Providers) bool {
			return providers.AgentEnabled()
		},
		build: func(ctx context.Context, providers config.Providers) (Provider, error) {
			provider, err := pkgllm.NewAgentCompletion(ctx, &pkgllm.AgentConfig{
				Command:        providers.Agent.Command,
				Workdir:        providers.Agent.Workdir,
				RequestTimeout: providers.Agent.RequestTimeout,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Agent client: %w", err)
			}
			return provider, nil
		},
	},
	orderedProviderFactory{
		enabled: func(providers config.Providers) bool {
			return providers.OpenAI.APIKey != ""
		},
		build: func(ctx context.Context, providers config.Providers) (Provider, error) {
			provider, err := pkgllm.NewOpenAICompletion(ctx, &pkgllm.OpenAIConfig{
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
		},
	},
	orderedProviderFactory{
		enabled: func(providers config.Providers) bool {
			return providers.Gemini.APIKey != ""
		},
		build: func(ctx context.Context, providers config.Providers) (Provider, error) {
			provider, err := pkgllm.NewGeminiCompletion(ctx, &pkgllm.GeminiConfig{
				APIKey:    providers.Gemini.APIKey,
				Model:     providers.Gemini.Model,
				MaxTokens: int32(providers.Gemini.MaxTokens),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Gemini client: %w", err)
			}
			return provider, nil
		},
	},
	orderedProviderFactory{
		enabled: func(providers config.Providers) bool {
			return providers.Anthropic.APIKey != ""
		},
		build: func(ctx context.Context, providers config.Providers) (Provider, error) {
			provider, err := pkgllm.NewAnthropicCompletion(ctx, &pkgllm.AnthropicConfig{
				APIKey:         providers.Anthropic.APIKey,
				Model:          providers.Anthropic.Model,
				MaxTokens:      providers.Anthropic.MaxTokens,
				RequestTimeout: providers.Anthropic.RequestTimeout,
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
	providers := cfg.Providers

	for _, factory := range providerFactories {
		if !factory.Enabled(providers) {
			continue
		}

		provider, err := factory.Build(ctx, providers)
		if err != nil {
			return nil, err
		}

		return provider, nil
	}

	return nil, fmt.Errorf("no LLM provider configured")
}
