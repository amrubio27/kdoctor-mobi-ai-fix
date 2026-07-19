package provider

// Provider defines the interface for an AI Fixer provider (e.g., Claude, Cursor, Gemini).
type Provider interface {
	Fix(prompt string) (patch string, err error)
}
