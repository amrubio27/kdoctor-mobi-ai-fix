package provider

import "fmt"

// StubProvider provides a generic stub for not-yet-implemented providers
type StubProvider struct {
	Name string
}

func (p *StubProvider) Fix(prompt string) (string, error) {
	return "", fmt.Errorf("provider %q is a stub and not yet implemented", p.Name)
}

func NewCursorProvider() Provider {
	return &StubProvider{Name: "cursor"}
}

func NewGeminiProvider() Provider {
	return &StubProvider{Name: "gemini"}
}

func NewCodexProvider() Provider {
	return &StubProvider{Name: "codex"}
}

func NewMobiAIProvider() Provider {
	return &StubProvider{Name: "mobiai"}
}
