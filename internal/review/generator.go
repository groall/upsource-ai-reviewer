package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/llm"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
)

type iCommentGenerator interface {
	generate(review *upsource.Review) ([]*reviewComment, error)
}

type generatorConfig struct {
	userPromptTemplate string
	systemMessage      string
	maxPerReview       int
	activeProvider     string
}

type commentGenerator struct {
	llmProvider llm.Provider
	gitProvider git.Provider
	cfg         generatorConfig
	ctx         context.Context
}

// newGenerator creates a new LLM commentGenerator instance.
func newGenerator(ctx context.Context, cfg generatorConfig, providers config.Providers, gitProvider git.Provider) (*commentGenerator, error) {
	reviewer := &commentGenerator{
		cfg:         cfg,
		ctx:         ctx,
		gitProvider: gitProvider,
	}

	var err error
	reviewer.llmProvider, err = llm.CreateLLMProvider(ctx, providers, cfg.activeProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return reviewer, nil
}

// Do calls OpenAI Chat Completion API to review changes.
func (c *commentGenerator) generate(review *upsource.Review) ([]*reviewComment, error) {

	changes, commitsComments, err := c.gitProvider.GetReviewChanges(review)
	if err != nil {
		return nil, fmt.Errorf("error getting review changes for %s: %w", review.GetBranch(), err)
	}

	systemPrompt := buildSystemPrompt(c.cfg.systemMessage, c.cfg.maxPerReview)
	userPrompt := buildUserPrompt(c.cfg.userPromptTemplate, changes, commitsComments)

	log.Print("Sending prompt to LLM...")

	llmResponse, err := c.complete(userPrompt, systemPrompt)
	if err != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReview, c.cfg.activeProvider)
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	log.Printf("Received LLM response: %s\n", llmResponse)

	comments, err := parsLLMResponse(llmResponse)
	if err != nil {
		return nil, err
	}

	comments = validateCommentsAgainstDiff(changes, comments)

	return comments, nil
}

// buildSystemPrompt substitutes the limit into the configured templates.
func buildSystemPrompt(systemMessage string, maxPerReview int) string {
	return strings.Replace(systemMessage, config.MaxPerReviewPlaceholder, strconv.Itoa(maxPerReview), -1)
}

// buildUserPrompt substitutes the diff, commit messages and limit into the configured templates.
func buildUserPrompt(userPromptTemplate, changes, commitsComments string) (userPrompt string) {
	userPrompt = strings.Replace(userPromptTemplate, config.DiffsPlaceholder, changes, -1)
	userPrompt = strings.Replace(userPrompt, config.MessagesPlaceholder, commitsComments, -1)

	return userPrompt
}

func (c *commentGenerator) complete(userPrompt, systemPrompt string) (string, error) {
	return c.llmProvider.Completion(userPrompt, systemPrompt)
}

// parsLLMResponse processes the LLM response and returns the review comments.
func parsLLMResponse(llmResponse string) (comments []*reviewComment, err error) {
	// Try to extract JSON from the assistant content
	extracted := cleanLLMResponse(llmResponse)

	// Unmarshal the JSON response from the LLM
	if err = json.Unmarshal([]byte(extracted), &comments); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	if len(comments) == 0 {
		log.Println("AI commentGenerator found no issues to comment on.")
		return nil, nil
	}

	return comments, nil
}

// cleanLLMResponse extracts the JSON response from the LLM assistant content.
func cleanLLMResponse(content string) string {
	// Try to extract JSON from the assistant content
	// find first '{' and last '}'
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return ""
}
