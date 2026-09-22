package html

import (
	"strings"
	"testing"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

func sampleReport() types.Report {
	return types.Report{
		HealthScore: 76,
		Summary:     types.Summary{Errors: 1, Warnings: 2, Info: 1, Total: 4},
		Findings: []types.Finding{
			{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError,
				File: "src/Auth.kt", Line: 42, Column: 5, Message: "PII in log",
				FixHint: "Mask the value"},
			{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityWarning,
				File: "src/List.kt", Line: 10, Message: "missing key"},
			{ID: "compose-modifier-missing", Cluster: "compose-performance", Severity: types.SeverityWarning,
				File: "src/Card.kt", Line: 3, Message: "no Modifier param"},
			{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo,
				File: "src/Repo.kt", Line: 7, Message: "hardcoded dispatcher"},
		},
	}
}

func render(t *testing.T, r types.Report) string {
	t.Helper()
	var buf strings.Builder
	if err := RenderHTML(r, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	return buf.String()
}

func TestRenderHTMLIncludesScoreAndFindings(t *testing.T) {
	out := render(t, sampleReport())
	for _, want := range []string{
		"<!DOCTYPE html>", "76", "sec-log-pii", "src/Auth.kt", "Mask the value",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML report missing %q", want)
		}
	}
}

// The report is meant to be opened offline and shared as one file, so anything
// fetched at runtime would break it exactly when it matters.
func TestRenderHTMLIsSelfContained(t *testing.T) {
	out := render(t, sampleReport())
	for _, forbidden := range []string{"<script src=", "<link rel=\"stylesheet\"", "http://", "https://"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("report is not self-contained, found %q", forbidden)
		}
	}
}

// TestRenderHTMLHasClusterFilters guards a feature the README promised and the
// template half-implemented: every card emitted data-cluster, but no control
// ever read it, so filtering by cluster silently did not exist.
func TestRenderHTMLHasClusterFilters(t *testing.T) {
	out := render(t, sampleReport())

	if !strings.Contains(out, `id="cluster-filters"`) {
		t.Fatal("no cluster filter row rendered")
	}
	for _, cluster := range []string{"security", "compose-performance", "coroutines"} {
		if !strings.Contains(out, `data-value="`+cluster+`"`) {
			t.Errorf("cluster %q has no filter button", cluster)
		}
	}
	if !strings.Contains(out, "data-cluster") {
		t.Error("finding cards must carry data-cluster for the filter to work")
	}
}

// The implicit global `event` is non-standard and throws under strict mode.
func TestRenderHTMLDoesNotUseImplicitGlobalEvent(t *testing.T) {
	out := render(t, sampleReport())
	if strings.Contains(out, "event.target.classList") {
		t.Error("script still relies on the implicit global event object")
	}
	if !strings.Contains(out, "addEventListener") {
		t.Error("expected handlers to be wired with addEventListener")
	}
}

func TestCountClustersOrdersByCountThenName(t *testing.T) {
	got := countClusters(sampleReport().Findings)
	if len(got) != 3 {
		t.Fatalf("want 3 clusters, got %d: %+v", len(got), got)
	}
	// Noisiest first, so the dominant category is one click away.
	if got[0].Name != "compose-performance" || got[0].Count != 2 {
		t.Errorf("expected compose-performance(2) first, got %+v", got[0])
	}
	// Ties break alphabetically for a stable report.
	if got[1].Name != "coroutines" || got[2].Name != "security" {
		t.Errorf("ties should break alphabetically, got %+v", got)
	}
}

func TestRenderHTMLWithNoFindings(t *testing.T) {
	out := render(t, types.Report{HealthScore: 100})
	if !strings.Contains(out, "100") {
		t.Error("a clean report should still render its score")
	}
	if strings.Contains(out, `id="cluster-filters"`) {
		t.Error("no findings means no cluster filter row")
	}
}
