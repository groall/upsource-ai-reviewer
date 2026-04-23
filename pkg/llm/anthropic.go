package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

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
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	return &AnthropicCompletion{
		client: client,
		config: *cfg,
		ctx:    ctx,
	}, nil
}

func (c *AnthropicCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	maxTokens := c.config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.config.Model),
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic request failed: %w", err)
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
