package replies

import (
	"fmt"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	appConfig "github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"
	"github.com/groall/upsource-go-client/client"
)

const replyUserPromptPrefixTemplate = `### Original code context
%s

### Discussion anchor
%s

### Discussion so far (oldest first)
`

type reviewReplyGenerator struct {
	generator   *generator
	gitProvider git.Provider
	review      *upsource.Review
	codeContext string
	botUserID   string
}

func (g *reviewReplyGenerator) setBotUserID(botUserID string) {
	g.botUserID = botUserID
}

func newReviewReplyGenerator(generator *generator, gitlabCfg *appConfig.Gitlab) (*reviewReplyGenerator, error) {
	gitlabProvider, err := git.NewGitlabProvider(gitlabCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab provider: %w", err)
	}

	return &reviewReplyGenerator{
		gitProvider: gitlabProvider,
		generator:   generator,
	}, nil
}

func (g *reviewReplyGenerator) prepareReview(review *upsource.Review) error {
	return g.loadCodeContext(review)
}

// Reply asks the LLM to produce a follow-up reply for a discussion thread.
func (g *reviewReplyGenerator) reply(d client.DiscussionInFileDTO) (*replyResult, error) {
	if g.codeContext == "" {
		return nil, fmt.Errorf("code context not loaded")
	}

	thread := buildThreadTranscript(d.Comments, g.botUserID)
	anchorText := buildReplyAnchorText(d.Anchor)
	threadText := formatThread(thread)

	prefix := fmt.Sprintf(replyUserPromptPrefixTemplate, g.codeContext, anchorText)
	suffix := threadText + "\n"

	return g.generator.reply(prefix, suffix)
}

func (g *reviewReplyGenerator) loadCodeContext(review *upsource.Review) error {
	codeContext, _, err := g.gitProvider.GetReviewChanges(review)
	if err != nil {
		return fmt.Errorf("get review changes: %w", err)
	}

	g.codeContext = codeContext

	return nil
}

func formatThread(thread []commentMsg) string {
	var b strings.Builder
	for i, m := range thread {
		role := "Human"
		if m.isBot {
			role = "AI Reviewer"
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		_, _ = fmt.Fprintf(&b, "%s (%s):\n%s", role, m.author, m.text)
	}

	return b.String()
}

func buildThreadTranscript(comments []client.CommentDTO, botUserID string) []commentMsg {
	out := make([]commentMsg, 0, len(comments))
	for _, c := range comments {
		out = append(out, commentMsg{
			author: c.AuthorID,
			isBot:  c.AuthorID == botUserID,
			text:   c.Text,
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
