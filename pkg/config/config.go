package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config is the root configuration loaded from YAML.
type Config struct {
	Upsource  Upsource  `yaml:"upsource"`
	Gitlab    Gitlab    `yaml:"gitlab"`
	Review    Review    `yaml:"review"`
	Providers Providers `yaml:"providers"`
	Polling   Polling   `yaml:"polling"`
	Replies   Replies   `yaml:"replies"`
	Metrics   Metrics   `yaml:"metrics"`
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

// Gitlab contains connection details used to fetch diffs from GitLab.
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

	return config, nil
}

// ValidateConfig validates configuration values and applies defaults in-place.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if err := config.Upsource.Validate(); err != nil {
		return fmt.Errorf("upsource config is invalid: %w", err)
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
