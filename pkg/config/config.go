package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config is the root configuration loaded from YAML.
type Config struct {
	Upsource   Upsource   `yaml:"upsource"`
	Gitlab     Gitlab     `yaml:"gitlab"`
	Review     Review     `yaml:"review"`
	Providers  Providers  `yaml:"providers"`
	Polling    Polling    `yaml:"polling"`
	Replies    Replies    `yaml:"replies"`
	Metrics    Metrics    `yaml:"metrics"`
	LLMLogging LLMLogging `yaml:"llmLogging"`
}

// LLMLogging controls optional persistence of prompts and raw model responses.
type LLMLogging struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

// Metrics controls the optional Prometheus metrics endpoint.
type Metrics struct {
	Enabled       bool   `yaml:"enabled"`
	ListenAddress string `yaml:"listenAddress"`
	Path          string `yaml:"path"`
}

// Polling controls how often Upsource is polled for reviews.
type Polling struct {
	IntervalSeconds int `yaml:"intervalSeconds"`
}

// Gitlab contains credentials used to clone repositories for agentic reviews.
type Gitlab struct {
	BaseURL     string `yaml:"baseUrl"`
	AccessToken string `yaml:"accessToken"`
}

// LoadConfig reads a YAML file and unmarshals it into a Config.
//
// LoadConfig does not validate values; call ValidateConfig after loading.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}

	if err = yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}
	if err = config.loadPromptFiles(filepath.Dir(filename)); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) loadPromptFiles(configDir string) error {
	load := func(name, path string) (string, error) {
		if path == "" {
			return "", nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(configDir, path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read %s prompt file %q: %w", name, path, err)
		}
		return string(content), nil
	}

	for _, prompt := range []struct {
		name   string
		path   string
		target *string
	}{
		{"review.systemMessage", c.Review.SystemMessageFile, &c.Review.SystemMessage},
		{"review.userPromptTemplate", c.Review.UserPromptTemplateFile, &c.Review.UserPromptTemplate},
		{"replies.systemMessage", c.Replies.SystemMessageFile, &c.Replies.SystemMessage},
	} {
		content, err := load(prompt.name, prompt.path)
		if err != nil {
			return err
		}
		if prompt.path != "" {
			*prompt.target = content
		}
	}
	return nil
}

// ValidateConfig validates configuration values and applies defaults in-place.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if err := config.Upsource.Validate(); err != nil {
		return fmt.Errorf("upsource config is invalid: %w", err)
	}

	if config.Polling.IntervalSeconds == 0 {
		return fmt.Errorf("polling.intervalSeconds is required")
	}

	if err := config.Review.Validate(); err != nil {
		return fmt.Errorf("review config is invalid: %w", err)
	}

	if config.LLMLogging.Enabled && config.LLMLogging.Directory == "" {
		return fmt.Errorf("llmLogging.directory is required when llmLogging.enabled is true")
	}

	if config.Review.ActiveProvider == ProviderAgent {
		if config.Gitlab.BaseURL == "" {
			return fmt.Errorf("gitlab.baseUrl is required for agent repository cloning")
		}
		if config.Gitlab.AccessToken == "" {
			return fmt.Errorf("gitlab.accessToken is required for agent repository cloning")
		}
	}

	if err := config.Replies.Validate(); err != nil {
		return fmt.Errorf("replies config is invalid: %w", err)
	}

	if config.Metrics.Enabled {
		if config.Metrics.ListenAddress == "" {
			config.Metrics.ListenAddress = ":2112"
		}
		if config.Metrics.Path == "" {
			config.Metrics.Path = "/metrics"
		}
	}

	if err := config.Providers.Validate(config.Review.ActiveProvider, config.Replies.ActiveProvider); err != nil {
		return fmt.Errorf("providers config is invalid: %w", err)
	}

	return nil
}
