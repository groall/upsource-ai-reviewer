package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Upsource  Upsource  `yaml:"upsource"`
	Gitlab    Gitlab    `yaml:"gitlab"`
	Review    Review    `yaml:"review"`
	Providers Providers `yaml:"providers"`
	Polling   Polling   `yaml:"polling"`
	Replies   Replies   `yaml:"replies"`
	Metrics   Metrics   `yaml:"metrics"`
}

type Metrics struct {
	Enabled       bool   `yaml:"enabled"`
	ListenAddress string `yaml:"listenAddress"`
	Path          string `yaml:"path"`
}

type Replies struct {
	Enabled       bool   `yaml:"enabled"`
	MaxPerThread  int    `yaml:"maxPerThread"`
	SystemMessage string `yaml:"systemMessage"`
}

type Polling struct {
	IntervalSeconds int `yaml:"intervalSeconds"`
}

type Gitlab struct {
	BaseURL     string `yaml:"baseUrl"`
	AccessToken string `yaml:"accessToken"`
}

// LoadConfig reads and parses the configuration YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}

	if err = yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return config, nil
}

func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if err := config.Upsource.Validate(); err != nil {
		return fmt.Errorf("upsource config is invalid: %w", err)
	}

	if err := config.Providers.Validate(); err != nil {
		return fmt.Errorf("providers config is invalid: %w", err)
	}

	if config.Gitlab.BaseURL == "" {
		return fmt.Errorf("gitlab.baseUrl is required")
	}

	if config.Gitlab.AccessToken == "" {
		return fmt.Errorf("gitlab.accessToken is required")
	}

	if config.Polling.IntervalSeconds == 0 {
		return fmt.Errorf("polling.intervalSeconds is required")
	}

	if err := config.Review.Validate(); err != nil {
		return fmt.Errorf("review config is invalid: %w", err)
	}

	if config.Metrics.Enabled {
		if config.Metrics.ListenAddress == "" {
			config.Metrics.ListenAddress = ":2112"
		}
		if config.Metrics.Path == "" {
			config.Metrics.Path = "/metrics"
		}
	}

	if config.Replies.Enabled {
		if config.Replies.MaxPerThread <= 0 {
			return fmt.Errorf("replies.maxPerThread must be > 0 when replies.enabled is true")
		}
		if config.Replies.SystemMessage == "" {
			return fmt.Errorf("replies.systemMessage is required when replies.enabled is true")
		}
	}

	return nil
}
