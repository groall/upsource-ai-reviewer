package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfigAllowsOpenAIWhenMetricsEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.OpenAI.APIKey = "key"

	require.NoError(t, ValidateConfig(cfg))
}

func TestValidateConfigAllowsAnthropicWhenMetricsEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.Anthropic.APIKey = "key"

	require.NoError(t, ValidateConfig(cfg))
}

func TestValidateConfigAllowsGeminiWhenMetricsEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.Gemini.APIKey = "key"

	require.NoError(t, ValidateConfig(cfg))
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
			MaxPerReview:       10,
			PostInLine:         "high",
			SystemMessage:      "max %d",
			UserPromptTemplate: "%s %s",
		},
		Polling: Polling{IntervalSeconds: 60},
		Metrics: Metrics{Enabled: true},
	}
}
