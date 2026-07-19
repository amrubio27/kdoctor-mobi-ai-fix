// Package sarif parses SARIF 2.1.0 produced by Detekt into []types.Finding.
//
// El parser es ESTRICTO: rechaza cualquier versión != "2.1.0".
// MapSARIFLevel convierte el level SARIF a types.Severity, aplicado ANTES
// del rulemap (que puede sobrescribir según metadata.json).
package sarif

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/adkd/adkd/internal/core/types"
)

type doc struct {
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool struct {
		Driver struct {
			Name string `json:"name"`
		} `json:"driver"`
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	} `json:"tool"`
	Results []struct {
		RuleID  string `json:"ruleId"`
		Level   string `json:"level"` // "error"|"warning"|"note"|"none"
		Message struct {
			Text string `json:"text"`
		} `json:"message"`
		Locations []struct {
			Phys struct {
				ArtLoc struct {
					URI string `json:"uri"`
				} `json:"artifactLocation"`
				Region struct {
					StartLine   int `json:"startLine"`
					StartColumn int `json:"startColumn"`
				} `json:"region"`
			} `json:"physicalLocation"`
		} `json:"locations"`
	} `json:"results"`
}

// MapSARIFLevel convierte el level SARIF a types.Severity.
// "" o "none" devuelve "" (rulemap sobrescribe).
func MapSARIFLevel(level string) types.Severity {
	switch level {
	case "error":
		return types.SeverityError
	case "warning":
		return types.SeverityWarning
	case "note":
		return types.SeverityInfo
	}
	return ""
}

// Parse lee el JSON SARIF y devuelve Finding normalizados.
// Rechaza versiones distintas a 2.1.0.
func Parse(r io.Reader) ([]types.Finding, error) {
	var d doc
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return nil, err
	}
	if d.Version != "2.1.0" {
		return nil, fmt.Errorf("unsupported SARIF version %q (expected 2.1.0)", d.Version)
	}
	var out []types.Finding
	for _, run := range d.Runs {
		for _, res := range run.Results {
			f := types.Finding{
				Rule:     res.RuleID,
				Message:  res.Message.Text,
				Severity: MapSARIFLevel(res.Level),
			}
			if len(res.Locations) > 0 {
				f.File = res.Locations[0].Phys.ArtLoc.URI
				f.Line = res.Locations[0].Phys.Region.StartLine
				f.Column = res.Locations[0].Phys.Region.StartColumn
			}
			out = append(out, f)
		}
	}
	return out, nil
}
