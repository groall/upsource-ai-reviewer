package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
	"github.com/groall/upsource-go-client/client"
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

type replierMockGitProvider struct {
	changes string
	commits string
	err     error
	calls   int
}

func (p *replierMockGitProvider) GetReviewChanges(review git.Review) (string, string, error) {
	p.calls++
	return p.changes, p.commits, p.err
}

func TestReplyFallsBackWhenPrefixCacheFails(t *testing.T) {
	provider := &prefixCacheMockProvider{
		prefixErr:        errors.New("prefix cache failed"),
		completionResult: `{"comment":"done","close":true}`,
	}

	gitProvider := &replierMockGitProvider{changes: "code context"}

	reviewer := &Reviewer{
		llmProvider: provider,
		gitProvider: gitProvider,
		config: &config.Config{
			Replies: config.Replies{
				SystemMessage: "reply system",
			},
		},
		ctx: context.Background(),
	}

	replier := NewReplier(reviewer).ForReview(&upsource.Review{})

	result, err := replier.Reply(
		client.DiscussionInFileDTO{
			Comments: []client.CommentDTO{{AuthorID: "dev", Text: "ok"}},
		},
		"bot",
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

	gitProvider := &replierMockGitProvider{changes: "code context"}

	reviewer := &Reviewer{
		llmProvider: provider,
		gitProvider: gitProvider,
		config: &config.Config{
			Replies: config.Replies{
				SystemMessage: "reply system",
			},
		},
		ctx: context.Background(),
	}

	replier := NewReplier(reviewer).ForReview(&upsource.Review{})

	result, err := replier.Reply(
		client.DiscussionInFileDTO{
			Comments: []client.CommentDTO{{AuthorID: "dev", Text: "ok"}},
		},
		"bot",
	)
	require.Nil(t, result)
	require.ErrorContains(t, err, "LLM reply request failed")
	require.Equal(t, 1, provider.prefixCalls)
	require.Equal(t, 1, provider.completionCalls)
}
