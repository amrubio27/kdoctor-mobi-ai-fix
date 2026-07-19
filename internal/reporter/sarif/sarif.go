// Package sarif provides a SARIF 2.1.0 writer for adkd reports.
//
// Es el inverso del parser de internal/core/sarif: dado un types.Report,
// emite un documento SARIF apto para GitHub Code Scanning, Azure DevOps,
// cualquier consumer OASIS SARIF 2.1.0 estándar.
package sarif

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/adkd/adkd/internal/core/types"
)

const schemaURL = "https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0-rtm.5.json"

type log struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string     `json:"name"`
	InformationURI string     `json:"informationUri"`
	Rules          []ruleDecl `json:"rules"`
}

type ruleDecl struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   messageT   `json:"message"`
	Locations []location `json:"locations"`
}

type messageT struct {
	Text string `json:"text"`
}

type location struct {
	PhysicalLocation physicalLocation `json:"physicalLocation"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           region           `json:"region"`
}

type artifactLocation struct {
	URI string `json:"uri"`
}

type region struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

// Write serializa el Report a SARIF 2.1.0.
func Write(r types.Report, w io.Writer) error {
	out := log{Version: "2.1.0", Schema: schemaURL, Runs: []run{buildRun(r)}}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func buildRun(r types.Report) run {
	// Index único de reglas — emitimos una `ruleDecl` por regla distinta.
	unique := map[string]bool{}
	ordered := []string{}
	for _, f := range r.Findings {
		if f.Rule == "" {
			continue
		}
		if !unique[f.Rule] {
			unique[f.Rule] = true
			ordered = append(ordered, f.Rule)
		}
	}
	sort.Strings(ordered)

	rules := make([]ruleDecl, 0, len(ordered))
	for _, id := range ordered {
		rd := ruleDecl{ID: id, Name: id}
		rd.ShortDescription.Text = "adkd: " + id
		rules = append(rules, rd)
	}

	results := make([]result, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Rule == "" {
			continue
		}
		results = append(results, result{
			RuleID:  f.ID,
			Level:   levelFromSeverity(f.Severity),
			Message: messageT{Text: f.Message},
			Locations: []location{{
				PhysicalLocation: physicalLocation{
					ArtifactLocation: artifactLocation{URI: f.File},
					Region:           region{StartLine: f.Line, StartColumn: f.Column},
				},
			}},
		})
	}
	return run{
		Tool: tool{Driver: driver{
			Name:           "adkd",
			InformationURI: "https://github.com/adkd/adkd",
			Rules:          rules,
		}},
		Results: results,
	}
}

func levelFromSeverity(s types.Severity) string {
	switch s {
	case types.SeverityError:
		return "error"
	case types.SeverityWarning:
		return "warning"
	case types.SeverityInfo:
		return "note"
	}
	return "warning"
}
