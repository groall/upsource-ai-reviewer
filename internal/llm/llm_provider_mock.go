package llm

// MockProvider is a mock of Provider interface.
type MockProvider struct {
	CompletionFunc func(userPrompt, systemPrompt string) (string, error)
}

// Completion calls CompletionFunc.
func (m *MockProvider) Completion(userPrompt, systemPrompt string) (string, error) {
	return m.CompletionFunc(userPrompt, systemPrompt)
}
