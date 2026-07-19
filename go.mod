module github.com/adkd/adkd

go 1.22

// Dependencias mínimas del runtime de Fase 1.
//   - cobra: entrypoint CLI (`cmd/adkd`) + subcomandos (`internal/cli/*`).
//   - yaml.v3: parseo de `adkd.config.yaml` (`internal/core/config`).
// Lipgloss se introduce en Fase 2 cuando entre el TUI real (charmbracelet/huh o lipgloss+bubbles).
require (
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)
