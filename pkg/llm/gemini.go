package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

type GeminiCompletion struct {
	client     *genai.Client
	config     *GeminiConfig
	ctx        context.Context
	cacheMu    sync.Mutex
	cacheNames map[string]string
}

type GeminiConfig struct {
	APIKey         string
	Model          string
	MaxTokens      int32
	RequestTimeout time.Duration
}

func NewGeminiCompletion(ctx context.Context, cfg *GeminiConfig) (*GeminiCompletion, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("a Gemini API key is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiCompletion{
		client:     client,
		config:     cfg,
		ctx:        ctx,
		cacheNames: make(map[string]string),
	}, nil
}

// Completion calls Gemini Chat Completion API.
func (c *GeminiCompletion) Completion(userPrompt, systemPrompt string) (string, error) {
	cfg := newGeminiGenerateContentConfig(c.config.MaxTokens)
	cfg.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{{Text: systemPrompt}},
	}

	return c.generateContent(userPrompt, cfg)
}

// CompletionWithPrefixCache calls Gemini with explicit cached content support.
func (c *GeminiCompletion) CompletionWithPrefixCache(userPromptPrefix, userPromptSuffix, systemPrompt string) (string, error) {
	if strings.TrimSpace(userPromptPrefix) == "" {
		return c.Completion(userPromptSuffix, systemPrompt)
	}

	cacheKey := geminiCacheKey(c.config.Model, systemPrompt, userPromptPrefix)
	cacheName, err := c.getOrCreateCachedContent(cacheKey, userPromptPrefix, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to prepare Gemini cached content: %w", err)
	}

	output, err := c.generateWithCachedContent(cacheName, userPromptSuffix)
	if err == nil {
		return output, nil
	}

	if !isGeminiCachedContentInvalidError(err) {
		return "", fmt.Errorf("the Gemini request with cached content failed: %w", err)
	}

	// Drop only the mapping we actually used so a concurrent refresh is not removed.
	c.clearCachedContentName(cacheKey, cacheName)

	cacheName, err = c.createCachedContent(userPromptPrefix, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to recreate Gemini cached content: %w", err)
	}
	c.setCachedContentName(cacheKey, cacheName)

	output, err = c.generateWithCachedContent(cacheName, userPromptSuffix)
	if err != nil {
		return "", fmt.Errorf("the Gemini request with refreshed cached content failed: %w", err)
	}

	return output, nil
}

func (c *GeminiCompletion) getOrCreateCachedContent(cacheKey, userPromptPrefix, systemPrompt string) (string, error) {
	if cacheName, ok := c.getCachedContentName(cacheKey); ok {
		return cacheName, nil
	}

	cacheName, err := c.createCachedContent(userPromptPrefix, systemPrompt)
	if err != nil {
		return "", err
	}
	c.setCachedContentName(cacheKey, cacheName)

	return cacheName, nil
}

func (c *GeminiCompletion) createCachedContent(userPromptPrefix, systemPrompt string) (string, error) {
	requestCtx, cancel := withRequestTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	cfg := &genai.CreateCachedContentConfig{
		Contents: geminiUserContent(userPromptPrefix),
	}
	if strings.TrimSpace(systemPrompt) != "" {
		cfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		}
	}

	cachedContent, err := c.client.Caches.Create(requestCtx, c.config.Model, cfg)
	if err != nil {
		return "", fmt.Errorf("the Gemini cache creation request failed: %w", err)
	}

	cacheName := strings.TrimSpace(cachedContent.Name)
	if cacheName == "" {
		return "", fmt.Errorf("the Gemini cache creation response does not include cache name")
	}

	return cacheName, nil
}

func (c *GeminiCompletion) generateWithCachedContent(cacheName, userPromptSuffix string) (string, error) {
	cfg := newGeminiGenerateContentConfig(c.config.MaxTokens)
	cfg.CachedContent = cacheName

	return c.generateContent(userPromptSuffix, cfg)
}

func (c *GeminiCompletion) generateContent(userPrompt string, cfg *genai.GenerateContentConfig) (string, error) {
	requestCtx, cancel := withRequestTimeout(c.ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.Models.GenerateContent(requestCtx, c.config.Model, geminiUserContent(userPrompt), cfg)
	if err != nil {
		return "", fmt.Errorf("the Gemini request failed: %w", err)
	}

	output := strings.TrimSpace(resp.Text())
	if output == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return output, nil
}

func newGeminiGenerateContentConfig(maxOutputTokens int32) *genai.GenerateContentConfig {
	think := int32(0)
	return &genai.GenerateContentConfig{
		MaxOutputTokens: maxOutputTokens,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &think, // Disables thinking
		},
	}
}

func geminiUserContent(userPrompt string) []*genai.Content {
	return []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromText(userPrompt),
		}, genai.RoleUser),
	}
}

func (c *GeminiCompletion) getCachedContentName(cacheKey string) (string, bool) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	cacheName, ok := c.cacheNames[cacheKey]
	return cacheName, ok
}

func (c *GeminiCompletion) setCachedContentName(cacheKey, cacheName string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if c.cacheNames == nil {
		c.cacheNames = make(map[string]string)
	}
	c.cacheNames[cacheKey] = cacheName
}

func (c *GeminiCompletion) clearCachedContentName(cacheKey, expectedCacheName string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if cacheName, ok := c.cacheNames[cacheKey]; ok && cacheName == expectedCacheName {
		delete(c.cacheNames, cacheKey)
	}
}

func geminiCacheKey(model, systemPrompt, userPromptPrefix string) string {
	// Zero-byte separators keep tuple boundaries unambiguous before hashing.
	sum := sha256.Sum256([]byte(model + "\x00" + systemPrompt + "\x00" + userPromptPrefix))
	return hex.EncodeToString(sum[:])
}

func isGeminiCachedContentInvalidError(err error) bool {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code == http.StatusNotFound {
			return true
		}

		// Some cache-expiry cases are surfaced only as free-form messages.
		if looksLikeCachedContentInvalid(apiErr.Message) {
			return true
		}
	}

	// Preserve detection when the SDK wraps API errors in plain text.
	return looksLikeCachedContentInvalid(err.Error())
}

func looksLikeCachedContentInvalid(value string) bool {
	lower := strings.ToLower(value)
	if !strings.Contains(lower, "cachedcontent") && !strings.Contains(lower, "cached content") {
		return false
	}

	return strings.Contains(lower, "not found") ||
		strings.Contains(lower, "expired") ||
		strings.Contains(lower, "invalid")
}
