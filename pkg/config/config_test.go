package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		content := `
upsource:
  baseUrl: https://upsource.example
  username: user
  password: password
  query: "state: open"
  reviewedLabel: AI-Reviewed
gitlab:
  baseUrl: https://gitlab.example
  accessToken: token
review:
  maxPerReview: 10
  postInLine: high
  systemMessageIntro: "intro"
  systemMessageGuidelines: "max {{max_per_review}}"
  systemMessageOutputFormat: "output"
  userPromptTemplate: "diffs: {{diffs}}\nmessages: {{messages}}"
providers:
  openai:
    apiKey: key
polling:
  intervalSeconds: 60
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

		cfg, err := LoadConfig(configPath)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, "https://upsource.example", cfg.Upsource.BaseURL)
		require.Equal(t, "token", cfg.Gitlab.AccessToken)
		require.Equal(t, "key", cfg.Providers.OpenAI.APIKey)
		require.Equal(t, 60, cfg.Polling.IntervalSeconds)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))

		require.Error(t, err)
		require.Nil(t, cfg)
		require.ErrorContains(t, err, "failed to read config file")
	})

	t.Run("returns error for malformed yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("upsource: ["), 0o600))

		cfg, err := LoadConfig(configPath)

		require.Error(t, err)
		require.Nil(t, cfg)
		require.ErrorContains(t, err, "failed to parse config YAML")
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("rejects nil config", func(t *testing.T) {
		err := ValidateConfig(nil)
		require.EqualError(t, err, "config is nil")
	})

	t.Run("fails when upsource config is invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.Upsource.BaseURL = ""

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "upsource config is invalid: upsource.baseUrl is required")
	})

	t.Run("fails when providers config is invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderOpenAI
		cfg.Providers = Providers{}

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "providers config is invalid: providers.openai.model is required when openai provider is active")
	})

	t.Run("fails when gitlab base url is missing", func(t *testing.T) {
		cfg := validConfig()
		cfg.Gitlab.BaseURL = ""

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "gitlab.baseUrl is required")
	})

	t.Run("fails when gitlab access token is missing", func(t *testing.T) {
		cfg := validConfig()
		cfg.Gitlab.AccessToken = ""

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "gitlab.accessToken is required")
	})

	t.Run("fails when polling interval is missing", func(t *testing.T) {
		cfg := validConfig()
		cfg.Polling.IntervalSeconds = 0

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "polling.intervalSeconds is required")
	})

	t.Run("sets metrics defaults when metrics are enabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics = Metrics{Enabled: true}

		err := ValidateConfig(cfg)

		require.NoError(t, err)
		require.Equal(t, ":2112", cfg.Metrics.ListenAddress)
		require.Equal(t, "/metrics", cfg.Metrics.Path)
	})

	t.Run("fails when replies enabled and max per thread is invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 0
		cfg.Replies.SystemMessage = "reply"

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "replies config is invalid: replies.maxPerThread must be > 0 when replies.enabled is true")
	})

	t.Run("fails when replies enabled and system message is missing", func(t *testing.T) {
		cfg := validConfig()
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 1
		cfg.Replies.SystemMessage = ""

		err := ValidateConfig(cfg)

		require.EqualError(t, err, "replies config is invalid: replies.systemMessage is required when replies.enabled is true")
	})

	t.Run("allows openai provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderOpenAI
		cfg.Providers = Providers{OpenAI: OpenAI{APIKey: "openai", Model: "gpt-5-mini"}}
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("allows gemini provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderGemini
		cfg.Providers = Providers{Gemini: Gemini{APIKey: "gemini", Model: "gemini-2.5-flash"}}
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("allows anthropic provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderAnthropic
		cfg.Providers = Providers{Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"}}
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("allows agent provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderAgent
		cfg.Providers = Providers{Agent: Agent{Command: "codex", CloneDir: "/tmp"}}
		require.NoError(t, ValidateConfig(cfg))
	})
}

func TestValidateConfigWithReplies(t *testing.T) {
	t.Run("succeeds with replies disabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.Replies.Enabled = false
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("succeeds with replies enabled and valid config", func(t *testing.T) {
		cfg := validConfig()
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 1
		cfg.Replies.SystemMessage = "reply template"
		cfg.Replies.ActiveProvider = ProviderOpenAI
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("fails when replies enabled with agent provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 1
		cfg.Replies.SystemMessage = "reply"
		cfg.Replies.ActiveProvider = ProviderAgent
		err := ValidateConfig(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be agent")
	})
}

func TestValidateConfigWithMetrics(t *testing.T) {
	t.Run("succeeds with metrics disabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics.Enabled = false
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("sets default metrics address when enabled without address", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics.Enabled = true
		cfg.Metrics.ListenAddress = ""
		cfg.Metrics.Path = ""

		require.NoError(t, ValidateConfig(cfg))
		require.Equal(t, ":2112", cfg.Metrics.ListenAddress)
		require.Equal(t, "/metrics", cfg.Metrics.Path)
	})

	t.Run("preserves custom metrics address when set", func(t *testing.T) {
		cfg := validConfig()
		cfg.Metrics.Enabled = true
		cfg.Metrics.ListenAddress = ":9090"
		cfg.Metrics.Path = "/prom"

		require.NoError(t, ValidateConfig(cfg))
		require.Equal(t, ":9090", cfg.Metrics.ListenAddress)
		require.Equal(t, "/prom", cfg.Metrics.Path)
	})
}

func TestValidateConfigErrors(t *testing.T) {
	t.Run("fails when polling interval is exactly 0", func(t *testing.T) {
		cfg := validConfig()
		cfg.Polling.IntervalSeconds = 0
		err := ValidateConfig(cfg)
		require.EqualError(t, err, "polling.intervalSeconds is required")
	})

	t.Run("succeeds with polling interval of 1", func(t *testing.T) {
		cfg := validConfig()
		cfg.Polling.IntervalSeconds = 1
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("succeeds with large polling interval", func(t *testing.T) {
		cfg := validConfig()
		cfg.Polling.IntervalSeconds = 3600
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("accepts gitlab URL with whitespace", func(t *testing.T) {
		cfg := validConfig()
		cfg.Gitlab.BaseURL = "   https://gitlab.example   "
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("accepts gitlab token with whitespace", func(t *testing.T) {
		cfg := validConfig()
		cfg.Gitlab.AccessToken = "   token   "
		require.NoError(t, ValidateConfig(cfg))
	})
}

func TestValidateConfigMultipleProviders(t *testing.T) {
	t.Run("allows different review and replies providers", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderOpenAI
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 1
		cfg.Replies.SystemMessage = "reply"
		cfg.Replies.ActiveProvider = ProviderGemini
		cfg.Providers = Providers{
			OpenAI: OpenAI{APIKey: "openai-key", Model: "gpt-5"},
			Gemini: Gemini{APIKey: "gemini-key", Model: "gemini-2.5"},
		}
		require.NoError(t, ValidateConfig(cfg))
	})

	t.Run("allows same provider for review and replies", func(t *testing.T) {
		cfg := validConfig()
		cfg.Review.ActiveProvider = ProviderOpenAI
		cfg.Replies.Enabled = true
		cfg.Replies.MaxPerThread = 1
		cfg.Replies.SystemMessage = "reply"
		cfg.Replies.ActiveProvider = ProviderOpenAI
		cfg.Providers = Providers{
			OpenAI: OpenAI{APIKey: "key", Model: "gpt-5"},
		}
		require.NoError(t, ValidateConfig(cfg))
	})
}

func validConfig() *Config {
	return &Config{
		Upsource: Upsource{
			BaseURL:       "https://upsource.example",
			Username:      "user",
			Password:      "password",
			Query:         "state: open",
			ReviewedLabel: "AI-Reviewed",
		},
		Gitlab: Gitlab{
			BaseURL:     "https://gitlab.example",
			AccessToken: "token",
		},
		Review: Review{
			ActiveProvider:            ProviderOpenAI,
			MaxPerReview:              10,
			SystemMessageIntro:        "intro",
			SystemMessageGuidelines:   "max {{max_per_review}}",
			SystemMessageOutputFormat: "output",
			UserPromptTemplate:        "diffs: {{diffs}}\nmessages: {{messages}}",
		},
		Providers: Providers{
			OpenAI: OpenAI{
				APIKey: "key",
				Model:  "gpt-5-mini",
			},
		},
		Polling: Polling{IntervalSeconds: 60},
		Metrics: Metrics{Enabled: false},
	}
}
