// Package sarif provides a SARIF 2.1.0 writer for kdoctor reports.
//
// Es el inverso del parser de internal/core/sarif: dado un types.Report,
// emite un documento SARIF apto para GitHub Code Scanning, Azure DevOps,
// cualquier consumer OASIS SARIF 2.1.0 estándar.
package sarif

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
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
	// ruleIdentity is the single identifier used BOTH in the rules[] table and
	// in results[].ruleId. They used to disagree: declarations were keyed on
	// f.Rule (the detekt rule name) while results emitted f.ID (the kdoctor id),
	// so for every detekt-sourced finding GitHub Code Scanning received a ruleId
	// that matched no declared rule. kdoctor ids are the canonical ones; f.Rule
	// is kept as the rule Name for traceability.
	unique := map[string]bool{}
	ordered := []string{}
	names := map[string]string{}
	for _, f := range r.Findings {
		id := ruleIdentity(f)
		if !unique[id] {
			unique[id] = true
			ordered = append(ordered, id)
			names[id] = f.Rule
		}
	}
	sort.Strings(ordered)

	rules := make([]ruleDecl, 0, len(ordered))
	for _, id := range ordered {
		name := names[id]
		if name == "" {
			name = id
		}
		rd := ruleDecl{ID: id, Name: name}
		rd.ShortDescription.Text = "kdoctor: " + id
		rules = append(rules, rd)
	}

	results := make([]result, 0, len(r.Findings))
	for _, f := range r.Findings {
		results = append(results, result{
			RuleID:  ruleIdentity(f),
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
			Name:           "kdoctor",
			InformationURI: "https://github.com/amrubio27/kdoctor-mobi-ai-fix",
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

// ruleIdentity returns the canonical SARIF rule id for a finding: the kdoctor
// rule id when present, else the upstream (detekt) rule name, else a stable
// placeholder. Findings are never dropped from the report just because one of
// those fields is empty.
func ruleIdentity(f types.Finding) string {
	if f.ID != "" {
		return f.ID
	}
	if f.Rule != "" {
		return f.Rule
	}
	return "kdoctor-unknown"
}
