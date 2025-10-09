package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

type Reviewer struct {
	llmProvider Provider
	config      *config.Config
	ctx         context.Context
}

// New creates a new LLM Reviewer instance.
func New(ctx context.Context, cfg *config.Config) (*Reviewer, error) {
	reviewer := &Reviewer{
		config: cfg,
		ctx:    ctx,
	}

	var err error
	reviewer.llmProvider, err = createLLMProvider(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return reviewer, nil
}

// Do calls OpenAI Chat Completion API to review changes.
func (c *Reviewer) Do(changes string, comments string) ([]*ReviewComment, error) {
	// Build a concise prompt and send to OpenAI-compatible API using SDK
	prompt := fmt.Sprintf(c.config.Promts.UserPromptTemplate, changes, comments)

	systemMessage := fmt.Sprintf(c.config.Promts.SystemMessage, c.config.Comments.MaxPerReview)

	llmResponse, err := c.llmProvider.Completion(prompt, systemMessage)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	log.Printf("Received LLM response: %s\n", llmResponse)

	return processAndPostLLMResponse(llmResponse)
}

// processAndPostLLMResponse processes the LLM response and returns the review comments.
func processAndPostLLMResponse(llmResponse string) ([]*ReviewComment, error) {
	// Try to extract JSON from the assistant content
	llmResponse = parseLLMDecision(llmResponse)

	var comments []*ReviewComment
	// Unmarshal the JSON response from the LLM
	if err := json.Unmarshal([]byte(llmResponse), &comments); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	if len(comments) == 0 {
		log.Println("AI Reviewer found no issues to comment on.")
		return nil, nil
	}

	return comments, nil

}

// parseLLMDecision extracts the JSON response from the LLM assistant content.
func parseLLMDecision(content string) string {
	// Try to extract JSON from the assistant content
	// find first '{' and last '}'
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return ""
}
