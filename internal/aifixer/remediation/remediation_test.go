package remediation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

const sample = "package demo\n" + // 1
	"\n" + // 2
	"class Repo {\n" + // 3
	"    fun load() {\n" + // 4
	"        val x = 1\n" + // 5
	"        log(secret)\n" + // 6
	"        val y = 2\n" + // 7
	"    }\n" + // 8
	"}\n" // 9

func staticReader(src string) SourceReader {
	return func(string) (string, error) { return src, nil }
}

func TestBuildProducesActionableItems(t *testing.T) {
	findings := []types.Finding{{
		ID: "sec-log-pii", Rule: "sec-log-pii", Cluster: "security",
		Severity: types.SeverityError, File: "Repo.kt", Line: 6, Column: 9,
		Message: "PII in log", FixHint: "Do not log secrets",
	}}

	plan, skipped := Build("/proj", 42, findings, 2, staticReader(sample))
	if len(skipped) != 0 {
		t.Fatalf("unexpected skips: %v", skipped)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(plan.Items))
	}
	it := plan.Items[0]

	// The range is what makes the plan actionable: without it the agent has to
	// infer which lines the context covers, which is how in-place patching went
	// wrong before.
	if it.ReplaceFrom != 4 || it.ReplaceTo != 8 {
		t.Errorf("finding on line 6 with 2 context lines should map to 4..8, got %d..%d",
			it.ReplaceFrom, it.ReplaceTo)
	}
	if it.FixHint != "Do not log secrets" {
		t.Errorf("fix hint lost: %q", it.FixHint)
	}
	if !strings.Contains(it.Context, "log(secret)") {
		t.Errorf("context does not include the offending line:\n%s", it.Context)
	}
	if plan.HealthScore != 42 || plan.SchemaVersion != SchemaVersion {
		t.Errorf("plan metadata wrong: score=%d schema=%q", plan.HealthScore, plan.SchemaVersion)
	}
}

// One unreadable file must not sink the whole run; the caller is told which.
func TestBuildSkipsUnreadableFilesWithoutFailing(t *testing.T) {
	read := func(path string) (string, error) {
		if path == "Gone.kt" {
			return "", errors.New("no such file")
		}
		return sample, nil
	}
	findings := []types.Finding{
		{ID: "a", File: "Gone.kt", Line: 1, Severity: types.SeverityError},
		{ID: "b", File: "Repo.kt", Line: 6, Severity: types.SeverityError},
	}

	plan, skipped := Build("/proj", 10, findings, 2, read)
	if len(plan.Items) != 1 || plan.Items[0].ID != "b" {
		t.Fatalf("readable finding should survive, got %+v", plan.Items)
	}
	if len(skipped) != 1 || skipped[0] != "Gone.kt" {
		t.Fatalf("skipped files must be reported, got %v", skipped)
	}
}

func TestBuildIgnoresFindingsWithoutFile(t *testing.T) {
	plan, _ := Build("/proj", 0, []types.Finding{{ID: "orphan"}}, 2, staticReader(sample))
	if len(plan.Items) != 0 {
		t.Fatalf("a finding with no file cannot be remediated, got %+v", plan.Items)
	}
}

func TestWriteJSONRoundTrips(t *testing.T) {
	plan, _ := Build("/proj", 55, []types.Finding{{
		ID: "sec-log-pii", File: "Repo.kt", Line: 6, Severity: types.SeverityError,
	}}, 2, staticReader(sample))

	var buf strings.Builder
	if err := WriteJSON(plan, &buf); err != nil {
		t.Fatal(err)
	}
	var back Plan
	if err := json.Unmarshal([]byte(buf.String()), &back); err != nil {
		t.Fatalf("emitted JSON does not parse: %v\n%s", err, buf.String())
	}
	if back.SchemaVersion != SchemaVersion || len(back.Items) != 1 {
		t.Fatalf("round trip lost data: %+v", back)
	}
	if back.Instructions == "" {
		t.Error("instructions must travel with the plan; the consuming model needs them")
	}
}

func TestWriteMarkdownIncludesRangeAndHint(t *testing.T) {
	plan, _ := Build("/proj", 55, []types.Finding{{
		ID: "sec-log-pii", File: "Repo.kt", Line: 6,
		Severity: types.SeverityError, FixHint: "Do not log secrets",
	}}, 2, staticReader(sample))

	var buf strings.Builder
	if err := WriteMarkdown(plan, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"sec-log-pii", "Replace lines 4-8", "Do not log secrets", "Health Score"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q:\n%s", want, out)
		}
	}
}
