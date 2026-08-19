package replies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	appConfig "github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/llm"
)

// commentMsg is a single message in a thread transcript passed to the LLM.
type commentMsg struct {
	author string
	isBot  bool
	text   string
}

type replyResult struct {
	Comment string `json:"comment"`
	Close   bool   `json:"close"`
}

type generator struct {
	llmProvider    llm.Provider
	logMessages    bool
	systemMessage  string
	activeProvider string
}

func newGenerator(ctx context.Context, cfg appConfig.Replies, providersCfg appConfig.Providers) (*generator, error) {
	if cfg.SystemMessage == "" {
		return nil, fmt.Errorf("replies.systemMessage is not configured")
	}
	provider, err := llm.CreateLLMProvider(ctx, providersCfg, cfg.ActiveProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create Replier LLM provider: %w", err)
	}

	return &generator{
		llmProvider:    provider,
		logMessages:    cfg.LogMessages,
		systemMessage:  cfg.SystemMessage,
		activeProvider: cfg.ActiveProvider,
	}, nil
}

func (g *generator) reply(prefix, suffix string) (*replyResult, error) {
	if g.logMessages {
		log.Print("Sending reply prompt to LLM...")
	}

	replyText, llmErr := g.completeWithCache(prefix, suffix)
	if llmErr != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReply, g.activeProvider)
		return nil, fmt.Errorf("LLM reply request failed: %w", llmErr)
	}

	reply := strings.TrimSpace(replyText)
	extracted := parseLLMDiscissionReply(reply)
	if extracted == "" {
		return nil, errors.New("LLM chose silence for discussion")
	}

	var result replyResult
	if err := json.Unmarshal([]byte(extracted), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM reply for discussion: %w", err)
	}

	return &result, nil
}

func (g *generator) complete(userPrompt string) (string, error) {
	return g.llmProvider.Completion(userPrompt, g.systemMessage)
}

func (g *generator) completeWithCache(prefix, suffix string) (string, error) {
	if prefix == "" {
		return g.complete(prefix + suffix)
	}

	p, ok := g.llmProvider.(llm.PrefixCacheProvider)
	if !ok {
		return g.complete(prefix + suffix)
	}

	replyText, llmErr := p.CompletionWithPrefixCache(prefix, suffix, g.systemMessage)
	if llmErr != nil {
		if g.logMessages {
			log.Printf("Prefix-cache reply failed, retrying without prefix cache: %v", llmErr)
		}
		return g.complete(prefix + suffix)
	}

	return replyText, nil
}

func parseLLMDiscissionReply(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return ""
}
