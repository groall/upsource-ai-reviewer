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
	RequestTimeout time.Duration
}

type AgentCompletion struct {
	ctx    context.Context
	config AgentConfig
}

func NewAgentRunner(ctx context.Context, cfg *AgentConfig) (*AgentCompletion, error) {
	if cfg == nil {
		return nil, fmt.Errorf("agent config is required")
	}

	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	return &AgentCompletion{
		ctx:    ctx,
		config: *cfg,
	}, nil
}

// Run executes a local CLI command, piping the prompts via STDIN.
func (c *AgentCompletion) Run(userPrompt, systemPrompt, pathToRepo string) (string, error) {
	execCtx, cancel := withRequestTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	combinedPrompt := strings.TrimSpace(systemPrompt + "\n\n" + userPrompt)

	cmd := exec.CommandContext(execCtx, "bash", "-lc", c.config.Command)
	cmd.Stdin = strings.NewReader(combinedPrompt)
	cmd.Dir = pathToRepo

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %w: %s (stdout: %s)", err, strings.TrimSpace(stderr.String()), stdout.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("empty command response")
	}

	return output, nil
}
