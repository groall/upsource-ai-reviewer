package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/groall/upsource-go-client/client"

	"github.com/groall/upsource-ai-reviewer/internal/llmlog"
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
	llmLogger          *llmlog.Logger
}

type commentGenerator struct {
	llmProvider    llm.Provider
	upsourceClient *client.Client
	cfg            generatorConfig
	ctx            context.Context
}

// newGenerator creates a new LLM commentGenerator instance.
func newGenerator(ctx context.Context, cfg generatorConfig, providers config.Providers, upsourceClient *client.Client) (*commentGenerator, error) {
	reviewer := &commentGenerator{
		cfg:            cfg,
		ctx:            ctx,
		upsourceClient: upsourceClient,
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
	changes, commitsComments, err := upsource.GetReviewChanges(c.ctx, c.upsourceClient, review)
	if err != nil {
		return nil, fmt.Errorf("error getting review changes: %w", err)
	}

	systemPrompt := buildSystemPrompt(c.cfg.systemMessage, c.cfg.maxPerReview)
	userPrompt := buildUserPrompt(c.cfg.userPromptTemplate, changes, commitsComments)

	log.Printf("Sending prompt to LLM for review %s...", review.GetTitle())

	llmResponse, err := c.complete(userPrompt, systemPrompt)
	output := llmResponse
	if err != nil {
		output = "LLM request error: " + err.Error()
	}
	if logErr := c.cfg.llmLogger.Save(review.GetReviewID().ReviewID, formatLLMInput(systemPrompt, userPrompt), output); logErr != nil {
		log.Printf("Failed to save LLM input/output for review %s: %v", review.GetReviewID().ReviewID, logErr)
	}
	if err != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReview, c.cfg.activeProvider)
		return nil, fmt.Errorf("request LLM failed: %w", err)
	}
	log.Printf("Received LLM response: %s\n", llmResponse)

	comments, err := parsLLMResponse(llmResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing llm response: %w", err)
	}

	comments = validateCommentsAgainstDiff(changes, comments)

	return comments, nil
}

func formatLLMInput(systemPrompt, userPrompt string) string {
	return "System prompt:\n" + systemPrompt + "\n\nUser prompt:\n" + userPrompt
}

// buildSystemPrompt substitutes the limit into the configured templates.
func buildSystemPrompt(systemMessage string, maxPerReview int) string {
	return strings.Replace(systemMessage, config.MaxPerReviewPlaceholder, strconv.Itoa(maxPerReview), -1)
}

// buildUserPrompt substitutes the diff, commit messages and limit into the configured templates.
func buildUserPrompt(userPromptTemplate, changes, commitsComments string) (userPrompt string) {
	var locations strings.Builder
	forEachAddedLine(changes, func(file string, line int, content string) {
		fmt.Fprintf(&locations, "%s:%d: %s\n", file, line, content)
	})
	if locations.Len() > 0 {
		changes += "\n### Added lines (repository path:new-file line: content)\n\n" + locations.String()
	}
	userPrompt = strings.Replace(userPromptTemplate, config.DiffsPlaceholder, changes, -1)
	userPrompt = strings.Replace(userPrompt, config.MessagesPlaceholder, commitsComments, -1)

	return userPrompt
}

func (c *commentGenerator) complete(userPrompt, systemPrompt string) (string, error) {
	metrics.DefaultRecorder.RecordLLMRequest(metrics.OperationReview, c.cfg.activeProvider)
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
