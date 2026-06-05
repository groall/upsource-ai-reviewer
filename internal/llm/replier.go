package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/metrics"
)

// CommentMsg is a single message in a thread transcript passed to the LLM.
type CommentMsg struct {
	Author string
	IsBot  bool
	Text   string
}

type ReplyResult struct {
	Comment string `json:"comment"`
	Close   bool   `json:"close"`
}

const replyUserPromptPrefixTemplate = `### Original code context
%s

### Discussion anchor
%s

### Discussion so far (oldest first)
`

// Reply asks the LLM to produce a follow-up reply for a discussion thread.
func (c *Reviewer) Reply(thread []CommentMsg, codeContext string, anchorText string) (*ReplyResult, error) {
	if c.config.Replies.SystemMessage == "" {
		return nil, fmt.Errorf("replies.systemMessage is not configured")
	}

	threadText := formatThread(thread)
	userPrompt, prefix, suffix := buildReplyPromptParts(codeContext, anchorText, threadText)

	log.Print("Sending reply prompt to LLM...")

	var replyText string
	var err error
	if prefix != "" {
		if p, ok := c.llmProvider.(PrefixCacheProvider); ok {
			replyText, err = p.CompletionWithPrefixCache(prefix, suffix, c.config.Replies.SystemMessage)
		} else {
			replyText, err = c.complete(userPrompt, c.config.Replies.SystemMessage)
		}
	} else {
		replyText, err = c.complete(userPrompt, c.config.Replies.SystemMessage)
	}
	if err != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReply, c.config.Providers.ActiveLLMProvider())
		return nil, fmt.Errorf("LLM reply request failed: %w", err)
	}

	reply := strings.TrimSpace(replyText)
	extracted := parseLLMDiscissionReply(reply)
	if extracted == "" {
		return nil, errors.New("LLM chose silence for discussion")
	}

	var result ReplyResult
	if err := json.Unmarshal([]byte(extracted), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM reply for discussion: %w", err)
	}

	return &result, nil
}

func buildReplyPromptParts(codeContext, anchorText, threadText string) (fullPrompt, prefix, suffix string) {
	prefix = fmt.Sprintf(replyUserPromptPrefixTemplate, codeContext, anchorText)
	suffix = threadText + "\n"
	return prefix + suffix, prefix, suffix
}

func formatThread(thread []CommentMsg) string {
	var b strings.Builder
	for i, m := range thread {
		role := "Human"
		if m.IsBot {
			role = "AI Reviewer"
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		_, _ = fmt.Fprintf(&b, "%s (%s):\n%s", role, m.Author, m.Text)
	}

	return b.String()
}

func parseLLMDiscissionReply(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return ""
}
