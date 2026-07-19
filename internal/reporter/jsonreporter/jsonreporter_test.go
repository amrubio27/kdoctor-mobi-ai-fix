package jsonreporter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestWriteRoundTrip(t *testing.T) {
	r := types.Report{
		SchemaVersion: types.SchemaVersion,
		ProjectType:   "android",
		HealthScore:   47,
		Summary:       types.Summary{Errors: 3, Warnings: 9, Info: 6, Total: 18},
		Findings: []types.Finding{
			{
				ID: "x", Cluster: "compose-performance", Rule: "remember-missing",
				Severity: types.SeverityError, File: "F.kt", Line: 12, Column: 4,
				Message: "msg", FixHint: "wrap",
			},
		},
	}
	var buf strings.Builder
	if err := Write(r, &buf); err != nil {
		t.Fatal(err)
	}
	var again types.Report
	if err := json.Unmarshal([]byte(buf.String()), &again); err != nil {
		t.Fatal(err)
	}
	if again.HealthScore != 47 {
		t.Fatalf("score %d", again.HealthScore)
	}
	if len(again.Findings) != 1 {
		t.Fatalf("findings %d", len(again.Findings))
	}
	if again.Findings[0].FixHint != "wrap" {
		t.Fatalf("fixHint lost: %q", again.Findings[0].FixHint)
	}
	if again.Findings[0].Severity != types.SeverityError {
		t.Fatalf("severity lost: %q", again.Findings[0].Severity)
	}
}

func TestSchemaVersionIsV3(t *testing.T) {
	r := types.Report{SchemaVersion: types.SchemaVersion, HealthScore: 99}
	b := MustMarshal(r)
	if !strings.Contains(string(b), `"schemaVersion": "3"`) {
		t.Fatalf("missing schemaVersion v3 in output:\n%s", string(b))
	}
}
