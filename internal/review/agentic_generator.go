package review

import (
	"context"
	"fmt"
	"log"

	"github.com/groall/upsource-ai-reviewer/internal/git"
	"github.com/groall/upsource-ai-reviewer/internal/metrics"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
	"github.com/groall/upsource-ai-reviewer/pkg/upsource"

	pkgllm "github.com/groall/upsource-ai-reviewer/pkg/llm"
)

type agenticCommentGenerator struct {
	agentRunner *pkgllm.AgentCompletion
	gitProvider git.RepoCloner
	cfg         generatorConfig
	cloneDir    string
	ctx         context.Context
}

// newAgenticGenerator creates a new agenticCommentGenerator instance.
func newAgenticGenerator(ctx context.Context, cfg generatorConfig, agentCfg config.Agent, gitProvider git.RepoCloner) (*agenticCommentGenerator, error) {
	generator := &agenticCommentGenerator{
		cfg:         cfg,
		ctx:         ctx,
		gitProvider: gitProvider,
		cloneDir:    agentCfg.CloneDir,
	}

	agent, err := pkgllm.NewAgentRunner(ctx, &pkgllm.AgentConfig{
		Command:        agentCfg.Command,
		RequestTimeout: agentCfg.RequestTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	generator.agentRunner = agent

	return generator, nil
}

// generate reviews a change in fully agentic mode: it clones the review's
// repository at its branch into a fresh directory and runs the agent CLI there,
// so the agent can explore the real code. The diff is still included in the
// prompt for focus; the agent returns the same ReviewComment JSON.
func (g *agenticCommentGenerator) generate(review *upsource.Review) ([]*reviewComment, error) {
	changes, commitsComments, err := g.gitProvider.GetReviewChanges(review)
	if err != nil {
		return nil, fmt.Errorf("error getting review changes for %s: %w", review.GetBranch(), err)
	}

	cloneDir, err := g.gitProvider.PrepareReviewRepo(review, g.cloneDir)
	if err != nil {
		return nil, fmt.Errorf("failed to clone review %s: %w", review.GetBranch(), err)
	}

	systemPrompt := buildSystemPrompt(g.cfg.systemMessage, g.cfg.maxPerReview)
	userPrompt := buildUserPrompt(g.cfg.userPromptTemplate, changes, commitsComments)

	log.Printf("Running agent in %s...", cloneDir)

	llmResponse, err := g.agentRunner.Run(userPrompt, systemPrompt, cloneDir)
	if err != nil {
		metrics.DefaultRecorder.RecordLLMError(metrics.OperationReview, g.cfg.activeProvider)
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	log.Printf("Received agent response: %s\n", llmResponse)

	comments, err := parsLLMResponse(llmResponse)
	if err != nil {
		return nil, err
	}

	comments = validateCommentsAgainstDiff(changes, comments)

	return comments, nil
}
