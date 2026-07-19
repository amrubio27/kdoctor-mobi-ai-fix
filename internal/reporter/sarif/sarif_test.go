package sarif

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
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
	if !strings.Contains(out, `"name": "adkd"`) {
		t.Fatalf("missing tool name adkd")
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
