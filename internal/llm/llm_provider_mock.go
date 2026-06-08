package llm

type mockProvider struct {
	CompletionFunc func(userPrompt, systemPrompt string) (string, error)
}

func (m *mockProvider) Completion(userPrompt, systemPrompt string) (string, error) {
	return m.CompletionFunc(userPrompt, systemPrompt)
}
