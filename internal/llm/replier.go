package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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

// Reply asks the LLM to produce a follow-up reply for a discussion thread.
func (c *Reviewer) Reply(thread []CommentMsg, codeContext string) (*ReplyResult, error) {
	if c.config.Replies.SystemMessage == "" || c.config.Replies.UserPromptTemplate == "" {
		return nil, fmt.Errorf("replies prompt templates are not configured")
	}

	userPrompt := fmt.Sprintf(c.config.Replies.UserPromptTemplate, codeContext, formatThread(thread))

	log.Print("Sending reply prompt to LLM...")

	resp, err := c.llmProvider.Completion(userPrompt, c.config.Replies.SystemMessage)
	if err != nil {
		return nil, fmt.Errorf("LLM reply request failed: %w", err)
	}

	reply := strings.TrimSpace(resp)

	if reply == "" {
		return nil, errors.New("LLM chose silence for discussion")
	}

	var result ReplyResult
	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM reply for discussion: %w", err)
	}

	return &result, nil
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
		fmt.Fprintf(&b, "%s (%s):\n%s", role, m.Author, m.Text)
	}
	return b.String()
}
