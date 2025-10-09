package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type OpenAIConfig struct {
	APIKey         string
	Endpoint       string
	Model          string
	MaxTokens      int
	Temperature    float32
	RequestTimeout time.Duration
}

type OpenAICompletion struct {
	client *openai.Client
	config OpenAIConfig
	ctx    context.Context
}

func NewOpenAICompletion(ctx context.Context, cfg *OpenAIConfig) (*OpenAICompletion, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	c := openai.DefaultConfig(cfg.APIKey)
	c.BaseURL = normalizeOpenAIBaseURL(cfg.Endpoint)
	openAIClient := openai.NewClientWithConfig(c)

	return &OpenAICompletion{
		client: openAIClient,
		config: *cfg,
		ctx:    ctx,
	}, nil
}

// Completion calls OpenAI Chat Completion API.
func (c *OpenAICompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:               c.config.Model,
		MaxCompletionTokens: c.config.MaxTokens,
		Temperature:         c.config.Temperature,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	if content == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return content, nil
}

// normalizeOpenAIBaseURL ensures the BaseURL is suitable for go-openai client
// - appends "/v1" if missing
// - trims any path after "/v1/" if a full endpoint URL was provided
func normalizeOpenAIBaseURL(endpoint string) string {
	if endpoint == "" {
		return ""
	}

	e := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(e, "/v1") {
		return e
	}

	if idx := strings.Index(e, "/v1/"); idx != -1 {
		return e[:idx+3]
	}

	return e + "/v1"
}
