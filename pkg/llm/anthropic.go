package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultMaxTokens = 4096

type AnthropicConfig struct {
	APIKey         string
	Model          string
	MaxTokens      int
	RequestTimeout time.Duration
}

type AnthropicCompletion struct {
	client anthropic.Client
	config AnthropicConfig
	ctx    context.Context
}

func NewAnthropicCompletion(ctx context.Context, cfg *AnthropicConfig) (*AnthropicCompletion, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	return &AnthropicCompletion{
		client: client,
		config: *cfg,
		ctx:    ctx,
	}, nil
}

func (c *AnthropicCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	return c.runCompletion(anthropic.MessageNewParams{
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
}

func (c *AnthropicCompletion) CompletionWithPrefixCache(userPromptPrefix, userPromptSuffix, systemPrompt string) (string, error) {
	if strings.TrimSpace(userPromptPrefix) == "" {
		return c.Completion(userPromptSuffix, systemPrompt)
	}

	return c.runCompletion(anthropic.MessageNewParams{
		System: []anthropic.TextBlockParam{{
			Text: systemPrompt,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.ContentBlockParamUnion{OfText: &anthropic.TextBlockParam{
					Text:         userPromptPrefix,
					CacheControl: anthropic.NewCacheControlEphemeralParam(),
				}},
				anthropic.ContentBlockParamUnion{OfText: &anthropic.TextBlockParam{
					Text: userPromptSuffix,
				}},
			),
		},
	})
}

func (c *AnthropicCompletion) runCompletion(params anthropic.MessageNewParams) (string, error) {
	ctx, cancel := withRequestTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	params.Model = c.config.Model
	params.MaxTokens = int64(c.maxTokens())

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}

	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			content := strings.TrimSpace(tb.Text)
			if content != "" {
				return content, nil
			}
		}
	}

	return "", fmt.Errorf("empty LLM response")
}

func (c *AnthropicCompletion) maxTokens() int {
	if c.config.MaxTokens > 0 {
		return c.config.MaxTokens
	}

	return defaultMaxTokens
}
