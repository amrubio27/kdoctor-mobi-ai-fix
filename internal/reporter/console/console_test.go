package console

import (
	"strings"
	"testing"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

func sampleReport() types.Report {
	return types.Report{
		HealthScore: 76,
		Summary:     types.Summary{Errors: 1, Warnings: 1, Info: 1, Total: 3},
		Findings: []types.Finding{
			{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError,
				File: "src/Auth.kt", Line: 42, Column: 5, Message: "PII in log"},
			{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityWarning,
				File: "src/List.kt", Line: 10, Message: "missing key"},
			{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo,
				File: "src/Repo.kt", Line: 7, Message: "hardcoded dispatcher"},
		},
	}
}

// TestRenderReportWithoutTTYEmitsNoANSI is the regression guard for a bug that
// was visible in every piped run: the score line wrote ansiReset
// unconditionally, even though pickColor returns "" without a TTY, so a bare
// ESC[0m leaked into redirected output ("76/100\x1b[0m").
func TestRenderReportWithoutTTYEmitsNoANSI(t *testing.T) {
	var buf strings.Builder
	RenderReport(sampleReport(), &buf, false)

	if strings.Contains(buf.String(), "\x1b") {
		t.Fatalf("ANSI escape leaked into non-TTY output: %q", buf.String())
	}
}

func TestRenderSummaryWithoutTTYEmitsNoANSI(t *testing.T) {
	var buf strings.Builder
	RenderSummary(sampleReport(), &buf, false)

	if strings.Contains(buf.String(), "\x1b") {
		t.Fatalf("ANSI escape leaked into non-TTY summary: %q", buf.String())
	}
}

func TestRenderReportShowsScoreCountsAndLocations(t *testing.T) {
	var buf strings.Builder
	RenderReport(sampleReport(), &buf, false)
	out := buf.String()

	for _, want := range []string{
		"76/100",
		"1 errors", "1 warnings", "1 info",
		"sec-log-pii", "src/Auth.kt", "42",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("console output missing %q:\n%s", want, out)
		}
	}
}

// A summary that listed every finding would defeat --summary.
func TestRenderSummaryOmitsIndividualFindings(t *testing.T) {
	var buf strings.Builder
	RenderSummary(sampleReport(), &buf, false)
	out := buf.String()

	if !strings.Contains(out, "76/100") {
		t.Errorf("summary must still show the score:\n%s", out)
	}
	if strings.Contains(out, "src/Auth.kt") {
		t.Errorf("summary should not list individual findings:\n%s", out)
	}
}

func TestRenderReportWithNoFindings(t *testing.T) {
	var buf strings.Builder
	RenderReport(types.Report{HealthScore: 100}, &buf, false)
	if !strings.Contains(buf.String(), "No issues found") {
		t.Errorf("a clean project should say so:\n%s", buf.String())
	}
}

// With a TTY colours are expected; this pins that the flag actually switches
// behaviour rather than both paths being identical.
func TestRenderReportWithTTYEmitsColour(t *testing.T) {
	var buf strings.Builder
	RenderReport(sampleReport(), &buf, true)
	if !strings.Contains(buf.String(), "\x1b") {
		t.Fatal("expected ANSI colour when a TTY is reported")
	}
}
