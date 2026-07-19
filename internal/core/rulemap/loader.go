// Package rulemap carga el catálogo de reglas y mapea IDs Detekt a IDs adkd.
//
// single source of truth = rules/metadata.json (ver Tarea 1.3).
// Este paquete NO debe hardcodear reglas. Siempre lee del JSON.
package rulemap

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adkd/adkd/internal/core/types"
)

// LoadRules lee rules/metadata.json y devuelve el catálogo.
// El path es por defecto <repo-root>/rules/metadata.json cuando se
// construye el release; el CLI resolverá la búsqueda en internal/cli/scan.go.
func LoadRules(path string) ([]types.Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules metadata %s: %w", path, err)
	}
	var rules []types.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules metadata: %w", err)
	}
	return rules, nil
}
