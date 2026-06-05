package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type AgentConfig struct {
	Command        string
	Workdir        string
	RequestTimeout time.Duration
}

type AgentCompletion struct {
	ctx    context.Context
	config AgentConfig
}

func NewAgentCompletion(ctx context.Context, cfg *AgentConfig) (*AgentCompletion, error) {
	if cfg == nil {
		return nil, fmt.Errorf("command config is required")
	}

	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("a command is required")
	}

	return &AgentCompletion{
		ctx:    ctx,
		config: *cfg,
	}, nil
}

// Completion executes a local CLI command, piping the prompts via STDIN.
func (c *AgentCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
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
		return "", fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("empty command response")
	}

	return output, nil
}
