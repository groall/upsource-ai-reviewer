package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const unknownLLMProvider = "unknown"

const (
	// ProviderAgent is the provider id for the external command-based provider.
	ProviderAgent = "agent"
	// ProviderOpenAI is the provider id for OpenAI API.
	ProviderOpenAI = "openai"
	// ProviderGemini is the provider id for Google Gemini API.
	ProviderGemini = "gemini"
	// ProviderAnthropic is the provider id for Anthropic API.
	ProviderAnthropic = "anthropic"
)

// Providers holds configuration for all supported LLM providers.
type Providers struct {
	Agent     Agent     `yaml:"agent"`
	OpenAI    OpenAI    `yaml:"openai"`
	Gemini    Gemini    `yaml:"gemini"`
	Anthropic Anthropic `yaml:"anthropic"`
}

// OpenAI configures the OpenAI API provider.
type OpenAI struct {
	Endpoint       string        `yaml:"endpoint"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	Temperature    float64       `yaml:"temperature"`
	APIKey         string        `yaml:"apiKey"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

// Agent configures the external command-based provider.
type Agent struct {
	Command        string        `yaml:"command"`
	CloneDir       string        `yaml:"cloneDir"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

// ValidateAndNormalizeCloneDir checks that CloneDir exists and normalizes it to an absolute path.
func (a *Agent) ValidateAndNormalizeCloneDir() error {
	if strings.TrimSpace(a.CloneDir) == "" {
		return fmt.Errorf("cloneDir is required")
	}

	cloneDir := strings.TrimSpace(a.CloneDir)
	var absPath string
	var err error

	if !filepath.IsAbs(cloneDir) {
		absPath, err = filepath.Abs(cloneDir)
		if err != nil {
			return fmt.Errorf("failed to convert cloneDir to absolute path: %w", err)
		}
	} else {
		absPath = cloneDir
	}

	// Check if the directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cloneDir does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to access cloneDir: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("cloneDir is not a directory: %s", absPath)
	}

	a.CloneDir = absPath
	return nil
}

// Anthropic configures the Anthropic API provider.
type Anthropic struct {
	APIKey         string        `yaml:"apiKey"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

// Gemini configures the Google Gemini API provider.
type Gemini struct {
	APIKey         string        `yaml:"apiKey"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

// Validate validates the provider configuration for the providers used by the given review and replies modes.
func (p *Providers) Validate(reviewProvider, repliesProvider string) error {
	openAIEnabled := reviewProvider == ProviderOpenAI || repliesProvider == ProviderOpenAI
	if openAIEnabled && strings.TrimSpace(p.OpenAI.Model) == "" {
		return fmt.Errorf("providers.openai.model is required when openai provider is active")
	}
	if openAIEnabled && strings.TrimSpace(p.OpenAI.APIKey) == "" {
		return fmt.Errorf("providers.openai.apiKey is required when openai provider is active")
	}

	geminiEnabled := reviewProvider == ProviderGemini || repliesProvider == ProviderGemini
	if geminiEnabled && strings.TrimSpace(p.Gemini.Model) == "" {
		return fmt.Errorf("providers.gemini.model is required when gemini provider is active")
	}
	if geminiEnabled && strings.TrimSpace(p.Gemini.APIKey) == "" {
		return fmt.Errorf("providers.gemini.apiKey is required when gemini provider is active")
	}

	anthropicEnabled := reviewProvider == ProviderAnthropic || repliesProvider == ProviderAnthropic
	if anthropicEnabled && strings.TrimSpace(p.Anthropic.Model) == "" {
		return fmt.Errorf("providers.anthropic.model is required when gemini provider is active")
	}
	if anthropicEnabled && strings.TrimSpace(p.Anthropic.APIKey) == "" {
		return fmt.Errorf("providers.anthropic.apiKey is required when gemini provider is active")
	}

	if reviewProvider == ProviderAgent {
		if strings.TrimSpace(p.Agent.Command) == "" {
			return fmt.Errorf("providers.agent.command is required when agent provider is active")
		}
		if err := p.Agent.ValidateAndNormalizeCloneDir(); err != nil {
			return fmt.Errorf("providers.agent.%w", err)
		}
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

// CheckProviderByID validates a provider id.
func CheckProviderByID(id string) error {
	switch id {
	case ProviderAgent, ProviderOpenAI, ProviderGemini, ProviderAnthropic:
		return nil
	default:
		return fmt.Errorf("unknown provider ID: %s", id)
	}
}

// ActiveLLMProvider returns the first configured provider in priority order.
func (p *Providers) ActiveLLMProvider() string {
	if strings.TrimSpace(p.Agent.Command) != "" {
		return ProviderAgent
	}
	if strings.TrimSpace(p.OpenAI.APIKey) != "" {
		return ProviderOpenAI
	}
	if strings.TrimSpace(p.Gemini.APIKey) != "" {
		return ProviderGemini
	}
	if strings.TrimSpace(p.Anthropic.APIKey) != "" {
		return ProviderAnthropic
	}
	return unknownLLMProvider
}
