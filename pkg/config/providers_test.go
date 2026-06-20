package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProvidersValidate(t *testing.T) {
	t.Run("fails when openai is enabled without model", func(t *testing.T) {
		providers := &Providers{OpenAI: OpenAI{APIKey: "key"}}
		err := providers.Validate(ProviderOpenAI, "")
		require.EqualError(t, err, "providers.openai.model is required when openai provider is active")
	})

	t.Run("fails when gemini is enabled without model", func(t *testing.T) {
		providers := &Providers{Gemini: Gemini{APIKey: "key"}}
		err := providers.Validate(ProviderGemini, "")
		require.EqualError(t, err, "providers.gemini.model is required when gemini provider is active")
	})

	t.Run("fails when anthropic is enabled without model", func(t *testing.T) {
		providers := &Providers{Anthropic: Anthropic{APIKey: "key"}}
		err := providers.Validate(ProviderAnthropic, "")
		require.EqualError(t, err, "providers.anthropic.model is required when gemini provider is active")
	})

	t.Run("allows each provider", func(t *testing.T) {
		testCases := []struct {
			name     string
			provider string
			cfg      Providers
		}{
			{
				name:     "openai",
				provider: ProviderOpenAI,
				cfg:      Providers{OpenAI: OpenAI{APIKey: "key", Model: "gpt-5-mini"}},
			},
			{
				name:     "gemini",
				provider: ProviderGemini,
				cfg:      Providers{Gemini: Gemini{APIKey: "key", Model: "gemini-2.5-flash"}},
			},
			{
				name:     "anthropic",
				provider: ProviderAnthropic,
				cfg:      Providers{Anthropic: Anthropic{APIKey: "key", Model: "claude-opus-4-1"}},
			},
			{
				name:     "agent",
				provider: ProviderAgent,
				cfg:      Providers{Agent: Agent{Command: "codex", CloneDir: "/tmp"}},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, tc.cfg.Validate(tc.provider, ""))
			})
		}
	})
}

func TestProvidersAgentEnabled(t *testing.T) {
	t.Run("returns false for empty command", func(t *testing.T) {
		providers := Providers{}
		require.False(t, providers.AgentEnabled())
	})

	t.Run("returns false for whitespace command", func(t *testing.T) {
		providers := Providers{
			Agent: Agent{Command: "   "},
		}
		require.False(t, providers.AgentEnabled())
	})

	t.Run("returns true for non-empty command", func(t *testing.T) {
		providers := Providers{
			Agent: Agent{Command: "codex"},
		}
		require.True(t, providers.AgentEnabled())
	})
}

func TestProvidersActiveLLMProvider(t *testing.T) {
	t.Run("returns unknown when no providers are configured", func(t *testing.T) {
		providers := Providers{}
		require.Equal(t, unknownLLMProvider, providers.ActiveLLMProvider())
	})

	t.Run("prefers agent over all other providers", func(t *testing.T) {
		providers := Providers{
			Agent:     Agent{Command: "codex"},
			OpenAI:    OpenAI{APIKey: "openai"},
			Gemini:    Gemini{APIKey: "gemini"},
			Anthropic: Anthropic{APIKey: "anthropic"},
		}
		require.Equal(t, "agent", providers.ActiveLLMProvider())
	})

	t.Run("prefers openai over gemini and anthropic", func(t *testing.T) {
		providers := Providers{
			OpenAI:    OpenAI{APIKey: "openai", Model: "gpt-5-mini"},
			Gemini:    Gemini{APIKey: "gemini", Model: "gemini-2.5-flash"},
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "openai", providers.ActiveLLMProvider())
	})

	t.Run("prefers gemini over anthropic when openai is not configured", func(t *testing.T) {
		providers := Providers{
			Gemini:    Gemini{APIKey: "gemini", Model: "gemini-2.5-flash"},
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "gemini", providers.ActiveLLMProvider())
	})

	t.Run("returns anthropic when only anthropic is configured", func(t *testing.T) {
		providers := Providers{
			Anthropic: Anthropic{APIKey: "anthropic", Model: "claude-opus-4-1"},
		}
		require.Equal(t, "anthropic", providers.ActiveLLMProvider())
	})

	t.Run("ignores whitespace-only API keys", func(t *testing.T) {
		providers := Providers{
			OpenAI: OpenAI{
				APIKey: "   ",
				Model:  "gpt-5-mini",
			},
		}
		require.Equal(t, unknownLLMProvider, providers.ActiveLLMProvider())
	})
}

func TestProvidersValidateWithMultipleProviders(t *testing.T) {
	t.Run("succeeds when multiple providers are configured but only one is active", func(t *testing.T) {
		providers := &Providers{
			OpenAI:    OpenAI{APIKey: "key", Model: "gpt-5-mini"},
			Gemini:    Gemini{APIKey: "key", Model: "gemini-2.5-flash"},
			Anthropic: Anthropic{APIKey: "key", Model: "claude-opus-4-1"},
		}
		require.NoError(t, providers.Validate(ProviderOpenAI, ""))
	})

	t.Run("validates openai when used for review", func(t *testing.T) {
		providers := &Providers{
			OpenAI: OpenAI{APIKey: "key", Model: "gpt-5-mini"},
		}
		require.NoError(t, providers.Validate(ProviderOpenAI, ""))
	})

	t.Run("validates openai when used for replies", func(t *testing.T) {
		providers := &Providers{
			OpenAI: OpenAI{APIKey: "key", Model: "gpt-5-mini"},
		}
		require.NoError(t, providers.Validate("", ProviderOpenAI))
	})

	t.Run("validates both review and replies providers", func(t *testing.T) {
		providers := &Providers{
			OpenAI:    OpenAI{APIKey: "key", Model: "gpt-5-mini"},
			Anthropic: Anthropic{APIKey: "key", Model: "claude-opus-4-1"},
		}
		require.NoError(t, providers.Validate(ProviderOpenAI, ProviderAnthropic))
	})

	t.Run("fails when review provider missing required field", func(t *testing.T) {
		providers := &Providers{
			OpenAI: OpenAI{APIKey: "key"},
		}
		err := providers.Validate(ProviderOpenAI, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "model")
	})

	t.Run("fails when replies provider missing required field", func(t *testing.T) {
		providers := &Providers{
			Gemini: Gemini{APIKey: "key"},
		}
		err := providers.Validate("", ProviderGemini)
		require.Error(t, err)
		require.Contains(t, err.Error(), "model")
	})

	t.Run("ignores unconfigured providers", func(t *testing.T) {
		providers := &Providers{
			OpenAI: OpenAI{APIKey: "key", Model: "gpt-5-mini"},
		}
		require.NoError(t, providers.Validate(ProviderOpenAI, ""))
	})

	t.Run("fails when agent missing command", func(t *testing.T) {
		providers := &Providers{
			Agent: Agent{CloneDir: "/tmp"},
		}
		err := providers.Validate(ProviderAgent, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "command")
	})

	t.Run("fails when agent missing cloneDir", func(t *testing.T) {
		providers := &Providers{
			Agent: Agent{Command: "codex"},
		}
		err := providers.Validate(ProviderAgent, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cloneDir")
	})

	t.Run("fails when cloneDir does not exist", func(t *testing.T) {
		providers := &Providers{
			Agent: Agent{Command: "codex", CloneDir: "/nonexistent/path/xyz"},
		}
		err := providers.Validate(ProviderAgent, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when cloneDir is not a directory", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "testfile")
		require.NoError(t, err)
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				fmt.Printf("Error removing file: %v\n", err)
			}
		}(tmpFile.Name())

		providers := &Providers{
			Agent: Agent{Command: "codex", CloneDir: tmpFile.Name()},
		}
		err = providers.Validate(ProviderAgent, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a directory")
	})

	t.Run("succeeds when agent has both command and absolute cloneDir", func(t *testing.T) {
		providers := &Providers{
			Agent: Agent{Command: "codex", CloneDir: "/tmp"},
		}
		require.NoError(t, providers.Validate(ProviderAgent, ""))
	})

	t.Run("succeeds with relative cloneDir and converts to absolute", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		require.NoError(t, err)
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				fmt.Printf("Error removing files on path %s: %v\n", path, err)
			}
		}(tmpDir)

		providers := &Providers{
			Agent: Agent{Command: "codex", CloneDir: tmpDir},
		}
		err = providers.Validate(ProviderAgent, "")
		require.NoError(t, err)
		require.True(t, filepath.IsAbs(providers.Agent.CloneDir))
	})
}

func TestProvidersWhitespaceHandling(t *testing.T) {
	t.Run("openai enabled check with whitespace", func(t *testing.T) {
		providers := Providers{OpenAI: OpenAI{APIKey: "   "}}
		require.False(t, providers.OpenAIEnabled())
	})

	t.Run("gemini enabled check with whitespace", func(t *testing.T) {
		providers := Providers{Gemini: Gemini{APIKey: "   "}}
		require.False(t, providers.GeminiEnabled())
	})

	t.Run("anthropic enabled check with whitespace", func(t *testing.T) {
		providers := Providers{Anthropic: Anthropic{APIKey: "   "}}
		require.False(t, providers.AnthropicEnabled())
	})

	t.Run("agent enabled check with whitespace", func(t *testing.T) {
		providers := Providers{Agent: Agent{Command: "   "}}
		require.False(t, providers.AgentEnabled())
	})

	t.Run("openai enabled check with valid key", func(t *testing.T) {
		providers := Providers{OpenAI: OpenAI{APIKey: "key"}}
		require.True(t, providers.OpenAIEnabled())
	})

	t.Run("gemini enabled check with valid key", func(t *testing.T) {
		providers := Providers{Gemini: Gemini{APIKey: "key"}}
		require.True(t, providers.GeminiEnabled())
	})

	t.Run("anthropic enabled check with valid key", func(t *testing.T) {
		providers := Providers{Anthropic: Anthropic{APIKey: "key"}}
		require.True(t, providers.AnthropicEnabled())
	})

	t.Run("agent enabled check with valid command", func(t *testing.T) {
		providers := Providers{Agent: Agent{Command: "cmd"}}
		require.True(t, providers.AgentEnabled())
	})
}

func TestAgentValidateAndNormalizeCloneDir(t *testing.T) {
	t.Run("fails when cloneDir is empty", func(t *testing.T) {
		agent := &Agent{}
		err := agent.ValidateAndNormalizeCloneDir()
		require.Error(t, err)
		require.Contains(t, err.Error(), "cloneDir is required")
	})

	t.Run("fails when cloneDir does not exist", func(t *testing.T) {
		agent := &Agent{CloneDir: "/nonexistent/path/xyz"}
		err := agent.ValidateAndNormalizeCloneDir()
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when cloneDir is not a directory", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "testfile")
		require.NoError(t, err)
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				fmt.Printf("Error removing file: %v\n", err)
			}
		}(tmpFile.Name())

		agent := &Agent{CloneDir: tmpFile.Name()}
		err = agent.ValidateAndNormalizeCloneDir()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a directory")
	})

	t.Run("succeeds with absolute path", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		require.NoError(t, err)
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				fmt.Printf("Error removing files on path %s: %v\n", path, err)
			}
		}(tmpDir)

		agent := &Agent{CloneDir: tmpDir}
		err = agent.ValidateAndNormalizeCloneDir()
		require.NoError(t, err)
		require.Equal(t, tmpDir, agent.CloneDir)
		require.True(t, filepath.IsAbs(agent.CloneDir))
	})

	t.Run("trims whitespace from cloneDir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		require.NoError(t, err)
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				fmt.Printf("Error removing files on path %s: %v\n", path, err)
			}
		}(tmpDir)

		agent := &Agent{CloneDir: "  " + tmpDir + "  "}
		err = agent.ValidateAndNormalizeCloneDir()
		require.NoError(t, err)
		require.Equal(t, tmpDir, agent.CloneDir)
	})
}

func TestCheckProviderByID(t *testing.T) {
	t.Run("accepts valid provider ids", func(t *testing.T) {
		for _, provider := range []string{ProviderAgent, ProviderOpenAI, ProviderGemini, ProviderAnthropic} {
			require.NoError(t, CheckProviderByID(provider))
		}
	})

	t.Run("rejects unknown provider", func(t *testing.T) {
		err := CheckProviderByID("unknown")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown provider ID")
	})

	t.Run("rejects empty provider", func(t *testing.T) {
		err := CheckProviderByID("")
		require.Error(t, err)
	})

	t.Run("rejects whitespace provider", func(t *testing.T) {
		err := CheckProviderByID("   ")
		require.Error(t, err)
	})
}
