package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Upsource Upsource `yaml:"upsource"`
	Gitlab   Gitlab   `yaml:"gitlab"`
	Promts   Promts   `yaml:"promts"`
	OpenAI   OpenAI   `yaml:"openai"`
	Gemini   Gemini   `yaml:"gemini"`
	Polling  Polling  `yaml:"polling"`
	Comments Comments `yaml:"comments"`
}

type Comments struct {
	MaxPerReview      int  `yaml:"maxPerReview"`
	HighSeveritySplit bool `yaml:"highSeveritySplit"`
}

type Gemini struct {
	APIKey    string `yaml:"apiKey"`
	Endpoint  string `yaml:"endpoint"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"maxTokens"`
}

type Polling struct {
	IntervalSeconds int `yaml:"intervalSeconds"`
}

type Upsource struct {
	BaseURL       string `yaml:"baseUrl"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Query         string `yaml:"query"`
	ReviewedLabel string `yaml:"reviewedLabel"`
}

type Gitlab struct {
	BaseURL     string `yaml:"baseUrl"`
	AccessToken string `yaml:"accessToken"`
}

type OpenAI struct {
	Endpoint       string        `yaml:"endpoint"`
	Model          string        `yaml:"model"`
	MaxTokens      int           `yaml:"maxTokens"`
	Temperature    float64       `yaml:"temperature"`
	APIKey         string        `yaml:"apiKey"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

type Promts struct {
	SystemMessage      string `yaml:"systemMessage"`
	UserPromptTemplate string `yaml:"userPromptTemplate"`
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
	if config.Upsource.BaseURL == "" {
		return fmt.Errorf("upsource.baseUrl is required")
	}
	if config.Upsource.Username == "" {
		return fmt.Errorf("upsource.username is required")
	}
	if config.Upsource.Password == "" {
		return fmt.Errorf("upsource.password is required")
	}
	if config.Upsource.Query == "" {
		return fmt.Errorf("upsource.query is required")
	}
	if config.Upsource.ReviewedLabel == "" {
		return fmt.Errorf("upsource.reviewedLabel is required")
	}
	if config.Gitlab.BaseURL == "" {
		return fmt.Errorf("gitlab.baseUrl is required")
	}
	if config.Gitlab.AccessToken == "" {
		return fmt.Errorf("gitlab.accessToken is required")
	}
	if config.OpenAI.APIKey == "" && config.Gemini.APIKey == "" {
		return fmt.Errorf("either openai.apiKey or gemini.apiKey is required")
	}
	if config.Promts.SystemMessage == "" {
		return fmt.Errorf("promts.systemMessage is required")
	}
	if config.Promts.UserPromptTemplate == "" {
		return fmt.Errorf("promts.userPromptTemplate is required")
	}
	if config.Polling.IntervalSeconds == 0 {
		return fmt.Errorf("polling.intervalSeconds is required")
	}

	return nil
}
