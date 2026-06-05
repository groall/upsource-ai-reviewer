package config

import (
	"fmt"
	"strings"
	"time"
)

const unknownLLMProvider = "unknown"

type Providers struct {
	Agent     Agent     `yaml:"agent"`
	OpenAI    OpenAI    `yaml:"openai"`
	Gemini    Gemini    `yaml:"gemini"`
	Anthropic Anthropic `yaml:"anthropic"`
}

type OpenAI struct {
	Endpoint       string        `yaml:"endpoint"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	Temperature    float64       `yaml:"temperature"`
	APIKey         string        `yaml:"apiKey"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

type Agent struct {
	Command        string        `yaml:"command"`
	Workdir        string        `yaml:"workdir"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

type Anthropic struct {
	APIKey         string        `yaml:"apiKey"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

type Gemini struct {
	APIKey    string `yaml:"apiKey"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"maxTokens"`
}

func (p *Providers) Validate() error {
	if p.ActiveLLMProvider() == unknownLLMProvider {
		return fmt.Errorf("either providers.openai.apiKey, providers.gemini.apiKey, providers.anthropic.apiKey, or providers.agent.command is required")
	}
	return nil
}

func (p *Providers) AgentEnabled() bool {
	return strings.TrimSpace(p.Agent.Command) != ""
}

func (p *Providers) ActiveLLMProvider() string {
	if p.AgentEnabled() {
		return "agent"
	}
	if p.OpenAI.APIKey != "" {
		return "openai"
	}
	if p.Gemini.APIKey != "" {
		return "gemini"
	}
	if p.Anthropic.APIKey != "" {
		return "anthropic"
	}

	return unknownLLMProvider
}
