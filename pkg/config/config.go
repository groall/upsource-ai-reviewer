package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Upsource  Upsource  `yaml:"upsource"`
	Gitlab    Gitlab    `yaml:"gitlab"`
	Review    Review    `yaml:"review"`
	OpenAI    OpenAI    `yaml:"openai"`
	Codex     Codex     `yaml:"codex"`
	Gemini    Gemini    `yaml:"gemini"`
	Anthropic Anthropic `yaml:"anthropic"`
	Polling   Polling   `yaml:"polling"`
	Replies   Replies   `yaml:"replies"`
}

type Replies struct {
	Enabled       bool   `yaml:"enabled"`
	MaxPerThread  int    `yaml:"maxPerThread"`
	SystemMessage string `yaml:"systemMessage"`
}

type Review struct {
	MaxPerReview       int    `yaml:"maxPerReview"`
	PostInLine         string `yaml:"postInLine"` // high, mid, low, none
	SystemMessage      string `yaml:"systemMessage"`
	UserPromptTemplate string `yaml:"userPromptTemplate"`
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
	BaseURL         string `yaml:"baseUrl"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Query           string `yaml:"query"`
	ReviewedLabel   string `yaml:"reviewedLabel"`
	InvitationLabel string `yaml:"invitationLabel"`
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

type Codex struct {
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
	if config.OpenAI.APIKey == "" && config.Gemini.APIKey == "" && config.Codex.Command == "" && config.Anthropic.APIKey == "" {
		return fmt.Errorf("either openai.apiKey, gemini.apiKey, anthropic.apiKey, or codex.command is required")
	}
	// Allow Codex without API keys; no additional validation needed beyond a command being present.
	if config.Polling.IntervalSeconds == 0 {
		return fmt.Errorf("polling.intervalSeconds is required")
	}

	if config.Review.MaxPerReview == 0 {
		return fmt.Errorf("review.maxPerReview is required")
	}
	if config.Review.PostInLine == "" {
		return fmt.Errorf("review.postInLine is required")
	}
	if config.Review.PostInLine != "high" && config.Review.PostInLine != "mid" && config.Review.PostInLine != "low" && config.Review.PostInLine != "none" {
		return fmt.Errorf("review.postInLine must be one of: high, mid, low, none")
	}
	if config.Review.SystemMessage == "" {
		return fmt.Errorf("review.systemMessage is required")
	}
	// Guard against accidental missing fmt args in templates.
	if s := fmt.Sprintf(config.Review.SystemMessage, config.Review.MaxPerReview); s == "" || containsFmtError(s) {
		return fmt.Errorf("review.systemMessage is not a valid fmt template (expected maxPerReview placeholder like %%d)")
	}
	if config.Review.UserPromptTemplate == "" {
		return fmt.Errorf("review.userPromptTemplate is required")
	}
	if s := fmt.Sprintf(config.Review.UserPromptTemplate, "diff", "commits"); s == "" || containsFmtError(s) {
		return fmt.Errorf("review.userPromptTemplate is not a valid fmt template (expected placeholders for diff and commits comments)")
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

func containsFmtError(s string) bool {
	// fmt.Sprintf reports formatting problems as "%!<verb>(...)".
	return strings.Contains(s, "%!")
}
