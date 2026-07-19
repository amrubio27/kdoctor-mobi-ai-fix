package rulemap

import (
	"sort"

	"github.com/adkd/adkd/internal/core/types"
)

// Index es un índice in-memory construido desde []types.Rule.
// Permite lookup O(1) por DetektRule. Sólo reglas "live" se indexan.
type Index struct {
	byDetekt map[string]types.Rule
}

// BuildIndex construye el índice a partir del catálogo.
// DetektRule vacío o status != "live" se omiten del índice.
func BuildIndex(rules []types.Rule) *Index {
	idx := &Index{byDetekt: make(map[string]types.Rule, len(rules))}
	for _, r := range rules {
		if r.Status != "live" || r.DetektRule == "" {
			continue
		}
		idx.byDetekt[r.DetektRule] = r
	}
	return idx
}

// Map enriquece cada Finding con id/cluster/severity/fixHint del catálogo.
// Devuelve una NUEVA lista; el input queda intacto. Las reglas sin matchear
// quedan con id="unmapped:<rule>" y cluster="unknown".
func (idx *Index) Map(findings []types.Finding) []types.Finding {
	out := make([]types.Finding, 0, len(findings))
	for _, f := range findings {
		if r, ok := idx.byDetekt[f.Rule]; ok {
			f.ID = r.ID
			f.Cluster = r.Cluster
			f.Severity = r.Severity
			f.FixHint = r.FixHint
			if f.Severity == "" && r.Severity != "" {
				f.Severity = r.Severity
			}
		} else {
			f.ID = "unmapped:" + f.Rule
			f.Cluster = "unknown"
			if f.Severity == "" {
				f.Severity = types.SeverityInfo
			}
		}
		out = append(out, f)
	}
	// orden estable (file, line, col) — útil para reporter.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Column < out[j].Column
	})
	return out
}

// Len devuelve el número de reglas indexadas (para diagnóstico).
func (idx *Index) Len() int { return len(idx.byDetekt) }
