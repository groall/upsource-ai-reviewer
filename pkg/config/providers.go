package config

import (
	"fmt"
	"strings"
	"time"
)

const unknownLLMProvider = "unknown"

const (
	ProviderAgent     = "agent"
	ProviderOpenAI    = "openai"
	ProviderGemini    = "gemini"
	ProviderAnthropic = "anthropic"
)

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
	APIKey         string        `yaml:"apiKey"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

func (p *Providers) Validate() error {
	if p.ActiveLLMProvider() == unknownLLMProvider {
		return fmt.Errorf("either providers.openai.apiKey, providers.gemini.apiKey, providers.anthropic.apiKey, or providers.agent.command is required")
	}

	if p.OpenAIEnabled() && strings.TrimSpace(p.OpenAI.Model) == "" {
		return fmt.Errorf("providers.openai.model is required when providers.openai.apiKey is set")
	}

	if p.GeminiEnabled() && strings.TrimSpace(p.Gemini.Model) == "" {
		return fmt.Errorf("providers.gemini.model is required when providers.gemini.apiKey is set")
	}

	if p.AnthropicEnabled() && strings.TrimSpace(p.Anthropic.Model) == "" {
		return fmt.Errorf("providers.anthropic.model is required when providers.anthropic.apiKey is set")
	}

	return nil
}

func (p *Providers) AgentEnabled() bool {
	return strings.TrimSpace(p.Agent.Command) != ""
}

func (p *Providers) OpenAIEnabled() bool {
	return strings.TrimSpace(p.OpenAI.APIKey) != ""
}

func (p *Providers) GeminiEnabled() bool {
	return strings.TrimSpace(p.Gemini.APIKey) != ""
}

func (p *Providers) AnthropicEnabled() bool {
	return strings.TrimSpace(p.Anthropic.APIKey) != ""
}

func (p *Providers) ActiveLLMProvider() string {
	if p.AgentEnabled() {
		return ProviderAgent
	}
	if p.OpenAIEnabled() {
		return ProviderOpenAI
	}
	if p.GeminiEnabled() {
		return ProviderGemini
	}
	if p.AnthropicEnabled() {
		return ProviderAnthropic
	}

	return unknownLLMProvider
}
