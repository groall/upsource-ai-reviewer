package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/stretchr/testify/require"
)

type prefixCacheMockProvider struct {
	completionResult string
	completionErr    error
	prefixResult     string
	prefixErr        error
	completionCalls  int
	prefixCalls      int
}

func (m *prefixCacheMockProvider) Completion(userPrompt, systemPrompt string) (string, error) {
	m.completionCalls++
	return m.completionResult, m.completionErr
}

func (m *prefixCacheMockProvider) CompletionWithPrefixCache(userPromptPrefix, userPromptSuffix, systemPrompt string) (string, error) {
	m.prefixCalls++
	return m.prefixResult, m.prefixErr
}

func TestReplyFallsBackWhenPrefixCacheFails(t *testing.T) {
	provider := &prefixCacheMockProvider{
		prefixErr:        errors.New("prefix cache failed"),
		completionResult: `{"comment":"done","close":true}`,
	}

	reviewer := &Reviewer{
		llmProvider: provider,
		config: &config.Config{
			Replies: config.Replies{
				SystemMessage: "reply system",
			},
		},
		ctx: context.Background(),
	}

	result, err := reviewer.Reply(
		[]CommentMsg{{Author: "dev", Text: "ok"}},
		"code context",
		"anchor",
	)
	require.NoError(t, err)
	require.Equal(t, "done", result.Comment)
	require.True(t, result.Close)
	require.Equal(t, 1, provider.prefixCalls)
	require.Equal(t, 1, provider.completionCalls)
}

func TestReplyReturnsErrorWhenPrefixAndFallbackFail(t *testing.T) {
	provider := &prefixCacheMockProvider{
		prefixErr:     errors.New("prefix cache failed"),
		completionErr: errors.New("plain completion failed"),
	}

	reviewer := &Reviewer{
		llmProvider: provider,
		config: &config.Config{
			Replies: config.Replies{
				SystemMessage: "reply system",
			},
		},
		ctx: context.Background(),
	}

	result, err := reviewer.Reply(
		[]CommentMsg{{Author: "dev", Text: "ok"}},
		"code context",
		"anchor",
	)
	require.Nil(t, result)
	require.ErrorContains(t, err, "LLM reply request failed")
	require.Equal(t, 1, provider.prefixCalls)
	require.Equal(t, 1, provider.completionCalls)
}
