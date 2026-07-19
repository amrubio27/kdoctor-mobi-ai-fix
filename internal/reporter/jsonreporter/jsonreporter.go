// Package jsonreporter serializa types.Report a JSON schema v3.
//
// Es el formato consumido por:
//   - GitHub Actions / workflows (--json para scripts)
//   - MobiAI Graph (--mobiai vuelca annotations con shape distinto)
//   - SARIF writer alternativo (Fase 2)
package jsonreporter

import (
	"encoding/json"
	"io"

	"github.com/adkd/adkd/internal/core/types"
)

// Write serializa el Report a w con indentado para legibilidad humana.
func Write(r types.Report, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// Marshal helper (para tests).
func MustMarshal(r types.Report) []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}
