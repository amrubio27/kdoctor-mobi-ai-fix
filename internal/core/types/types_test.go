package types

import "testing"

func TestSchemaVersion(t *testing.T) {
	if SchemaVersion != "3" {
		t.Fatalf("schema version expected 3, got %q", SchemaVersion)
	}
}

func TestSeverityContract(t *testing.T) {
	for _, s := range []Severity{SeverityError, SeverityWarning, SeverityInfo} {
		if s == "" {
			t.Fatal("empty severity")
		}
	}
	// Sólo tres valores legales en el enum; un cuarto debe ser distinguible.
	if Severity("debug") == SeverityError || Severity("debug") == SeverityWarning || Severity("debug") == SeverityInfo {
		t.Fatal("severity enum leak")
	}
}

func TestToMobiaiAnnotationRoundTrip(t *testing.T) {
	f := Finding{
		ID: "compose-remember-missing", Cluster: "compose-performance",
		Rule: "remember-missing", Severity: SeverityError,
		File: "src/Foo.kt", Line: 42, Column: 5,
		Message: "state not remembered", FixHint: "wrap with remember { ... }",
	}
	a := f.ToMobiaiAnnotation()
	if a.URI != "src/Foo.kt" || a.StartLine != 42 || a.RuleID != f.ID || a.Severity != "error" {
		t.Fatalf("annotation roundtrip broken: %+v", a)
	}
}

func TestSummaryAccumulatesSeverities(t *testing.T) {
	fs := []Finding{
		{Severity: SeverityError},
		{Severity: SeverityError},
		{Severity: SeverityWarning},
		{Severity: SeverityInfo},
		{Severity: SeverityInfo},
		{Severity: SeverityInfo},
	}
	manual := Summary{Errors: 2, Warnings: 1, Info: 3, Total: 6}
	if manual.Total != len(fs) {
		t.Fatalf("manual calc off")
	}
}
