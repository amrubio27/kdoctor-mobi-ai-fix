package sarif

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

func TestWriteProducesValidSARIF(t *testing.T) {
	r := types.Report{
		HealthScore: 47,
		Summary:     types.Summary{Errors: 1, Warnings: 1, Info: 1, Total: 3},
		Findings: []types.Finding{
			{
				ID: "compose-remember-missing", Cluster: "compose-performance",
				Rule: "Compose:ReusedModifierInstance", Severity: types.SeverityError,
				File: "src/Foo.kt", Line: 42, Column: 5, Message: "state not remembered",
			},
			{
				ID: "dead-unused-import", Cluster: "dead-code",
				Rule: "UnusedImport", Severity: types.SeverityInfo,
				File: "src/Bar.kt", Line: 1, Column: 1, Message: "unused import",
			},
		},
	}
	var buf strings.Builder
	if err := Write(r, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"version": "2.1.0"`) {
		t.Fatalf("missing SARIF version 2.1.0:\n%s", out)
	}
	if !strings.Contains(out, `"name": "kdoctor"`) {
		t.Fatalf("missing tool name kdoctor")
	}
	if !strings.Contains(out, "compose-remember-missing") {
		t.Fatalf("missing finding id compose-remember-missing")
	}
}

func TestWriteDedupeRuleDecls(t *testing.T) {
	r := types.Report{
		Findings: []types.Finding{
			{Rule: "A", Severity: types.SeverityWarning, File: "a", Line: 1},
			{Rule: "A", Severity: types.SeverityWarning, File: "b", Line: 2},
			{Rule: "B", Severity: types.SeverityInfo, File: "c", Line: 3},
		},
	}
	var buf strings.Builder
	if err := Write(r, &buf); err != nil {
		t.Fatal(err)
	}
	var parsed log
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Runs) != 1 {
		t.Fatalf("runs %d", len(parsed.Runs))
	}
	if len(parsed.Runs[0].Tool.Driver.Rules) != 2 {
		t.Fatalf("expected 2 unique rules, got %d", len(parsed.Runs[0].Tool.Driver.Rules))
	}
	if len(parsed.Runs[0].Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(parsed.Runs[0].Results))
	}
}

func TestSeverityToSARIFLevel(t *testing.T) {
	cases := map[types.Severity]string{
		types.SeverityError:   "error",
		types.SeverityWarning: "warning",
		types.SeverityInfo:    "note",
		"":                    "warning",
	}
	for sev, want := range cases {
		if got := levelFromSeverity(sev); got != want {
			t.Errorf("levelFromSeverity(%q) = %q, want %q", sev, got, want)
		}
	}
}

// TestEveryResultRuleIDIsDeclared pins the SARIF 2.1.0 invariant that every
// results[].ruleId must resolve to an entry in tool.driver.rules[]. The writer
// used to key declarations on f.Rule (detekt's name) while emitting f.ID
// (kdoctor's id) in results, so GitHub Code Scanning silently dropped every
// detekt-sourced finding. It also checks that findings are never discarded just
// because one identifier field is empty.
func TestEveryResultRuleIDIsDeclared(t *testing.T) {
	r := types.Report{
		Findings: []types.Finding{
			// detekt-sourced: ID and Rule differ — the regression case.
			{ID: "complexity-too-many-functions", Rule: "TooManyFunctions",
				Severity: types.SeverityWarning, File: "a.kt", Line: 1},
			// native: ID == Rule.
			{ID: "sec-log-pii", Rule: "sec-log-pii",
				Severity: types.SeverityError, File: "b.kt", Line: 2},
			// no Rule at all: must still be emitted, not dropped.
			{ID: "arch-orphan", Severity: types.SeverityInfo, File: "c.kt", Line: 3},
		},
	}
	var buf strings.Builder
	if err := Write(r, &buf); err != nil {
		t.Fatal(err)
	}
	var parsed log
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(parsed.Runs))
	}
	run := parsed.Runs[0]

	if len(run.Results) != len(r.Findings) {
		t.Fatalf("findings were dropped: want %d results, got %d", len(r.Findings), len(run.Results))
	}

	declared := map[string]bool{}
	for _, rd := range run.Tool.Driver.Rules {
		declared[rd.ID] = true
	}
	for _, res := range run.Results {
		if !declared[res.RuleID] {
			t.Errorf("results[].ruleId %q is not declared in tool.driver.rules[] (declared: %v)",
				res.RuleID, run.Tool.Driver.Rules)
		}
	}
}
