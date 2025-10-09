package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GeminiCompletion struct {
	client *genai.Client
	config *GeminiConfig
	ctx    context.Context
}

type GeminiConfig struct {
	APIKey    string
	Model     string
	MaxTokens int32
}

func NewGeminiCompletion(ctx context.Context, cfg *GeminiConfig) (*GeminiCompletion, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("a Gemini API key is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiCompletion{
		client: client,
		config: cfg,
		ctx:    ctx,
	}, nil
}

// Completion calls Gemini Chat Completion API.
func (c *GeminiCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	parts := []*genai.Part{
		genai.NewPartFromText(userPrompt),
	}
	content := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	think := int32(0)
	resp, err := c.client.Models.GenerateContent(c.ctx, c.config.Model, content, &genai.GenerateContentConfig{
		MaxOutputTokens: c.config.MaxTokens,
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &think, // Disables thinking
		},
	})
	if err != nil {
		return "", fmt.Errorf("the Gemini request failed: %w", err)
	}

	if resp.Text() == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return resp.Text(), nil
}
