package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
	"github.com/groall/upsource-go-client/client"
)

type Replier struct {
	llmProvider Provider
	gitProvider git.Provider
	config      *config.Config
}

type reviewReplier struct {
	replier     *Replier
	review      *upsource.Review
	codeContext string
	loaded      bool
}

func NewReplier(reviewer *Reviewer) *Replier {
	return &Replier{
		llmProvider: reviewer.llmProvider,
		gitProvider: reviewer.gitProvider,
		config:      reviewer.config,
	}
}

func (r *Replier) ForReview(review *upsource.Review) *reviewReplier {
	return &reviewReplier{
		replier: r,
		review:  review,
	}
}

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
func (r *Replier) Reply(review *upsource.Review, d client.DiscussionInFileDTO, botUserID string) (*ReplyResult, error) {
	return r.ForReview(review).Reply(d, botUserID)
}

func (rr *reviewReplier) Reply(d client.DiscussionInFileDTO, botUserID string) (*ReplyResult, error) {
	if rr.replier.config.Replies.SystemMessage == "" {
		return nil, fmt.Errorf("replies.systemMessage is not configured")
	}

	thread := buildThreadTranscript(d.Comments, botUserID)

	codeContext, err := rr.loadCodeContext()
	if err != nil {
		return nil, err
	}
	anchorText := buildReplyAnchorText(d.Anchor)

	threadText := formatThread(thread)
	userPrompt, prefix, suffix := buildReplyPromptParts(codeContext, anchorText, threadText)

	log.Print("Sending reply prompt to LLM...")

	var replyText string
	var llmErr error
	if prefix != "" {
		if p, ok := rr.replier.llmProvider.(PrefixCacheProvider); ok {
			replyText, llmErr = p.CompletionWithPrefixCache(prefix, suffix, rr.replier.config.Replies.SystemMessage)
			if llmErr != nil {
				log.Printf("Prefix-cache reply failed, retrying without prefix cache: %v", llmErr)
				replyText, llmErr = rr.replier.complete(userPrompt, rr.replier.config.Replies.SystemMessage)
			}
		} else {
			replyText, llmErr = rr.replier.complete(userPrompt, rr.replier.config.Replies.SystemMessage)
		}
	} else {
		replyText, llmErr = rr.replier.complete(userPrompt, rr.replier.config.Replies.SystemMessage)
	}
	if llmErr != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReply, rr.replier.config.Providers.ActiveLLMProvider())
		return nil, fmt.Errorf("LLM reply request failed: %w", llmErr)
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

func (r *Replier) complete(userPrompt, systemPrompt string) (string, error) {
	return r.llmProvider.Completion(userPrompt, systemPrompt)
}

func (rr *reviewReplier) loadCodeContext() (string, error) {
	if rr.loaded {
		return rr.codeContext, nil
	}

	codeContext, _, err := rr.replier.gitProvider.GetReviewChanges(rr.review)
	if err != nil {
		return "", fmt.Errorf("get review changes: %w", err)
	}

	rr.codeContext = codeContext
	rr.loaded = true
	return rr.codeContext, nil
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

func buildThreadTranscript(comments []client.CommentDTO, botUserID string) []CommentMsg {
	out := make([]CommentMsg, 0, len(comments))
	for _, c := range comments {
		out = append(out, CommentMsg{
			Author: c.AuthorID,
			IsBot:  c.AuthorID == botUserID,
			Text:   c.Text,
		})
	}
	return out
}

func buildReplyAnchorText(anchor client.AnchorDTO) string {
	if anchor.FileID == "" {
		return ""
	}

	var rangeText string
	if anchor.Range != nil {
		rangeText = fmt.Sprintf(" range=[%d,%d]", anchor.Range.StartOffset, anchor.Range.EndOffset)
	}

	return fmt.Sprintf("fileId=%s revisionId=%s inlineInRevision=%s%s", anchor.FileID, anchor.RevisionID, anchor.InlineInRevision, rangeText)
}
