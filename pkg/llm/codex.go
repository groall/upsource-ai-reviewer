package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CodexConfig struct {
	Command        string
	Workdir        string
	RequestTimeout time.Duration
}

type CodexCompletion struct {
	ctx    context.Context
	config CodexConfig
}

func NewCodexCompletion(ctx context.Context, cfg *CodexConfig) (*CodexCompletion, error) {
	if cfg == nil {
		return nil, fmt.Errorf("codex config is required")
	}

	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("a Codex command is required")
	}

	return &CodexCompletion{
		ctx:    ctx,
		config: *cfg,
	}, nil
}

// Completion executes a local Codex CLI command, piping the prompts via STDIN.
func (c *CodexCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	execCtx := c.ctx
	if c.config.RequestTimeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(execCtx, c.config.RequestTimeout)
		defer cancel()
	}

	combinedPrompt := strings.TrimSpace(systemPrompt + "\n\n" + userPrompt)

	cmd := exec.CommandContext(execCtx, "bash", "-lc", c.config.Command)
	cmd.Stdin = strings.NewReader(combinedPrompt)
	if c.config.Workdir != "" {
		cmd.Dir = c.config.Workdir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codex command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("empty Codex response")
	}

	return output, nil
}
