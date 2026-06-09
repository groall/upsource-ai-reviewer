package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

type Reviewer struct {
	llmProvider Provider
	gitProvider git.Provider
	cfg         ReviewConfig
	ctx         context.Context
}

// New creates a new LLM Reviewer instance.
func New(ctx context.Context, cfg ReviewConfig, providers config.Providers, gitProvider git.Provider) (*Reviewer, error) {
	reviewer := &Reviewer{
		cfg:         cfg,
		ctx:         ctx,
		gitProvider: gitProvider,
	}

	var err error
	reviewer.llmProvider, err = createLLMProvider(ctx, providers)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return reviewer, nil
}

// Do calls OpenAI Chat Completion API to review changes.
func (c *Reviewer) Do(review *upsource.Review) ([]*ReviewComment, error) {
	changes, commitsComments, err := c.gitProvider.GetReviewChanges(review)
	if err != nil {
		return nil, fmt.Errorf("error getting review changes for %s: %w", review.GetBranch(), err)
	}

	// Build a concise prompt and send to OpenAI-compatible API using SDK
	prompt := fmt.Sprintf(c.cfg.UserPromptTemplate, changes, commitsComments)

	systemMessage := fmt.Sprintf(c.cfg.SystemMessage, c.cfg.MaxPerReview)

	log.Print("Sending prompt to LLM...")

	llmResponse, err := c.complete(prompt, systemMessage)
	if err != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReview, c.cfg.ActiveProvider)
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	log.Printf("Received LLM response: %s\n", llmResponse)

	comments, err := processAndPostLLMResponse(llmResponse)
	if err != nil {
		return nil, err
	}

	comments = validateCommentsAgainstDiff(changes, comments)

	return comments, nil
}

func (c *Reviewer) complete(userPrompt, systemPrompt string) (string, error) {
	return c.llmProvider.Completion(userPrompt, systemPrompt)
}

// processAndPostLLMResponse processes the LLM response and returns the review comments.
func processAndPostLLMResponse(llmResponse string) ([]*ReviewComment, error) {
	// Try to extract JSON from the assistant content
	extracted := parseLLMReviewComments(llmResponse)

	var comments []*ReviewComment
	// Unmarshal the JSON response from the LLM
	if err := json.Unmarshal([]byte(extracted), &comments); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	if len(comments) == 0 {
		log.Println("AI Reviewer found no issues to comment on.")
		return nil, nil
	}

	return comments, nil

}

// parseLLMReviewComments extracts the JSON response from the LLM assistant content.
func parseLLMReviewComments(content string) string {
	// Try to extract JSON from the assistant content
	// find first '{' and last '}'
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return ""
}
