package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestWriteIncludesSummary(t *testing.T) {
	report := types.Report{
		SchemaVersion: "3",
		ProjectType:   "android",
		HealthScore:   82,
		Summary:       types.Summary{Errors: 4, Warnings: 2, Info: 1, Total: 7},
		Findings: []types.Finding{
			{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityError, File: "Foo.kt", Line: 10, Message: "missing key", FixHint: "add key"},
			{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo, File: "Bar.kt", Line: 20, Message: "hardcoded dispatcher"},
		},
	}

	var buf bytes.Buffer
	if err := Write(report, &buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "KDoctor Report") {
		t.Error("missing report title")
	}
	if !strings.Contains(out, "Health Score:") {
		t.Error("missing health score")
	}
	if !strings.Contains(out, "| Errors | Warnings | Info | Total |") {
		t.Error("missing summary table")
	}
	if !strings.Contains(out, "Top Clusters") {
		t.Error("missing top clusters section")
	}
	if !strings.Contains(out, "### compose-performance") {
		t.Error("missing cluster heading")
	}
	if !strings.Contains(out, "compose-missing-key") {
		t.Error("missing finding id")
	}
}

func TestWriteEmptyFindings(t *testing.T) {
	report := types.Report{
		SchemaVersion: "3",
		ProjectType:   "android",
		HealthScore:   100,
		Summary:       types.Summary{Total: 0},
		Findings:      []types.Finding{},
	}

	var buf bytes.Buffer
	if err := Write(report, &buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "_No issues found._") {
		t.Error("missing empty findings message")
	}
	if strings.Contains(out, "Top Clusters") {
		t.Error("should not show top clusters when there are no findings")
	}
}

func TestWriteSummary(t *testing.T) {
	report := types.Report{
		SchemaVersion: "3",
		ProjectType:   "android",
		HealthScore:   82,
		Summary:       types.Summary{Errors: 4, Warnings: 2, Info: 1, Total: 7},
		Findings: []types.Finding{
			{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityError, File: "Foo.kt", Line: 10, Message: "missing key"},
		},
	}

	var buf bytes.Buffer
	if err := WriteSummary(report, &buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Top Clusters") {
		t.Error("missing top clusters section")
	}
	if strings.Contains(out, "## Findings") {
		t.Error("summary report should not contain full findings section")
	}
}

func TestEscapeMD(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"a | b", "a \\| b"},
		{"`code`", "\\`code\\`"},
		{"line1\nline2", "line1 line2"},
		{"*bold*", "\\*bold\\*"},
		{"under_score", "under\\_score"},
		{"~strike~", "\\~strike\\~"},
	}
	for _, c := range cases {
		got := escapeMD(c.input)
		if got != c.want {
			t.Errorf("escapeMD(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSeverityBreakdown(t *testing.T) {
	findings := []types.Finding{
		{Severity: types.SeverityError},
		{Severity: types.SeverityError},
		{Severity: types.SeverityWarning},
		{Severity: types.SeverityInfo},
	}
	got := severityBreakdown(findings)
	want := "2 errors, 1 warnings, 1 info"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
