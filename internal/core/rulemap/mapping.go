package rulemap

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/adkd/adkd/internal/core/types"
)

// Index es un índice in-memory construido desde []types.Rule.
// Permite lookup O(1) por DetektRule o ID. Sólo reglas "live" se indexan.
type Index struct {
	byDetekt map[string]types.Rule
	byID     map[string]types.Rule
}

// BuildIndex construye el índice a partir del catálogo.
// Reglas con status != "live" se omiten del índice.
func BuildIndex(rules []types.Rule) *Index {
	idx := &Index{
		byDetekt: make(map[string]types.Rule),
		byID:     make(map[string]types.Rule),
	}
	for _, r := range rules {
		if r.Status != "live" {
			continue
		}
		idx.byID[r.ID] = r
		if r.DetektRule != "" {
			idx.byDetekt[r.DetektRule] = r
		}
	}
	return idx
}

// Map enriquece cada Finding con id/cluster/severity/fixHint del catálogo.
// Devuelve una NUEVA lista; el input queda intacto. Las reglas sin matchear
// quedan con id="unmapped:<orig>" y cluster="unknown".
//
// Detekt 1.23.x SARIF outputs qualified ruleIds como "detekt.complexity.TooManyFunctions".
// El catálogo solo guarda el nombre corto ("TooManyFunctions"). Normalizamos
// queryID quitando cualquier prefijo "<vendor>.<ruleset>." antes del lookup.
// Si no se encuentra por DetektRule, intentamos buscar por ID nativo directamente.
func (idx *Index) Map(findings []types.Finding) []types.Finding {
	out := make([]types.Finding, 0, len(findings))
	for _, f := range findings {
		queryID := f.Rule
		if lastDot := strings.LastIndex(queryID, "."); lastDot != -1 {
			queryID = queryID[lastDot+1:]
		}

		var matched bool
		var r types.Rule

		if rule, ok := idx.byDetekt[queryID]; ok {
			r = rule
			matched = true
		} else if queryID == "UnusedImport" && idx.byDetekt["UnusedImports"].ID != "" {
			r = idx.byDetekt["UnusedImports"]
			matched = true
		} else if queryID == "UnusedImports" && idx.byDetekt["UnusedImport"].ID != "" {
			r = idx.byDetekt["UnusedImport"]
			matched = true
		} else if rule, ok := idx.byID[queryID]; ok {
			r = rule
			matched = true
		}

		if matched {
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

// ApplyOverrides aplica la configuración del equipo (kdoctor.config.yaml).
// - Si f.File hace match con algún patrón de cfg.Excludes, se descarta.
// - Si cfg.Rules tiene un override para f.ID o f.Cluster, se aplica.
// - Si la severidad resultante es "off", "disabled" o "none", se descarta.
// Requiere importar filepath y config.
func ApplyOverrides(findings []types.Finding, excludes []string, overrides map[string]string) []types.Finding {
	if len(excludes) == 0 && len(overrides) == 0 {
		return findings
	}

	var out []types.Finding
	for _, f := range findings {
		// 1. Excludes
		excluded := false
		for _, pattern := range excludes {
			match, err := filepath.Match(pattern, f.File)
			// filepath.Match sólo funciona con un directorio o nombre base si no hay slashes.
			// Para globs más robustos se usaría filepath.Match recursivo o strings.Contains,
			// pero para Fase 1 asumiremos que pueden usar strings.Contains o filepath.Match
			if err == nil && match {
				excluded = true
				break
			}
			// Fallback manual para globs tipo "**/*"
			if strings.Contains(pattern, "**") {
				if strings.Contains(f.File, strings.ReplaceAll(pattern, "**", "")) {
					excluded = true
					break
				}
			}
		}
		if excluded {
			continue
		}

		// 2. Severity Overrides
		effectiveSeverity := string(f.Severity)

		// Rule-level override wins over Cluster-level
		if sev, ok := overrides[f.ID]; ok {
			effectiveSeverity = sev
		} else if sev, ok := overrides[f.Cluster]; ok {
			effectiveSeverity = sev
		}

		// 3. Drop "off"
		lowerSev := strings.ToLower(effectiveSeverity)
		if lowerSev == "off" || lowerSev == "disabled" || lowerSev == "none" {
			continue
		}

		f.Severity = types.Severity(effectiveSeverity)
		out = append(out, f)
	}
	return out
}

// Len devuelve el número de reglas indexadas (para diagnóstico).
func (idx *Index) Len() int { return len(idx.byID) }
