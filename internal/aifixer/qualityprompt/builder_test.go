package qualityprompt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

// TestBuildPrompt is preserved verbatim from round-1 so callers and the
// legacy test path keep working.
func TestBuildPrompt(t *testing.T) {
	finding := types.Finding{
		File:    "src/main/kotlin/App.kt",
		Line:    12,
		Column:  5,
		ID:      "compose-modifier-missing",
		Rule:    "ComposeModifierMissing",
		Message: "Composable function is missing a modifier",
		FixHint: "Add a modifier parameter with default Modifier",
	}

	sourceCode := "fun App() {\n    Text(\"Hello\")\n}"

	prompt, err := BuildPrompt(finding, sourceCode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "You are kdoctor AI Fixer") {
		t.Error("missing system prompt")
	}

	if !strings.Contains(prompt, "File: src/main/kotlin/App.kt") {
		t.Error("missing file name")
	}

	if !strings.Contains(prompt, "Fix Hint: Add a modifier parameter with default Modifier") {
		t.Error("missing fix hint")
	}

	if !strings.Contains(prompt, sourceCode) {
		t.Error("missing source code")
	}
}

// helper: build a stable 30-line Kotlin file with one line per integer.
func fixtureSource(lines int) string {
	var b strings.Builder
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(&b, "line %02d", i)
		if i < lines {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// TestBuildPromptWithContext_HappyPathAtMiddle: finding at line 12 of a
// 30-line file with contextLines=10 → rendered block is lines 2-22, the
// marker sits on line 12, lines outside the block do not appear.
func TestBuildPromptWithContext_HappyPathAtMiddle(t *testing.T) {
	src := fixtureSource(30)
	finding := types.Finding{
		File:     "/abs/proj/src/main/App.kt",
		Line:     12,
		ID:       "compose-missing-key",
		Cluster:  "compose-performance",
		Severity: types.SeverityError,
		Message:  "items() requires key parameter",
		FixHint:  "Add key = { it.id } or use itemsIndexed",
	}

	prompt, err := BuildPromptWithContext(finding, src, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Header: file is rendered as basename only (OS-agnostic via path.Base).
	for _, want := range []string{
		"You are kdoctor AI Fixer",
		"File: App.kt",
		"Issue at line 12",
		"Rule ID:    compose-missing-key",
		"Cluster:    compose-performance",
		"Severity:   error",
		"Message:    items() requires key parameter",
		"Change Hint",
		"Lines 2-22 of App.kt",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing required fragment %q", want)
		}
	}

	// Block should contain lines 2..22; marker on line 12.
	for _, ln := range []int{2, 5, 12, 15, 22} {
		w := fmt.Sprintf("line %02d", ln)
		if !strings.Contains(prompt, w) {
			t.Errorf("expected line %d (%q) to appear in rendered prompt", ln, w)
		}
	}

	// Off-range lines (1 and 23-30) must NOT appear.
	for _, ln := range []int{1, 23, 24, 25, 30} {
		w := fmt.Sprintf("line %02d", ln)
		if strings.Contains(prompt, w) {
			t.Errorf("off-range line %d (%q) leaked into rendered prompt", ln, w)
		}
	}

	// Marker must be appended to the finding line. The marker literal is
	// findingMarkerInline = "  <-- FINDING" (two leading spaces), so the
	// line 12 entry becomes "12: line 12  <-- FINDING" (one space from
	// "12: ", no space between "12" and the marker prefix). We use the
	// marker constant directly to avoid future drift.
	expectedLine := fmt.Sprintf("12: line 12%s", findingMarkerInline)
	if !strings.Contains(prompt, expectedLine) {
		t.Errorf("expected line 12 entry to be %q; got prompt missing the FINDING marker", expectedLine)
	}
}

// TestBuildPromptWithContext_ClampsAtFirstLine: finding.Line=1 clamps start
// to 0 so lines 1..11 are emitted.
func TestBuildPromptWithContext_ClampsAtFirstLine(t *testing.T) {
	src := fixtureSource(20)
	finding := types.Finding{File: "App.kt", Line: 1, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, err := BuildPromptWithContext(finding, src, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "Lines 1-11 of App.kt") {
		t.Errorf("expected clamp at top; got block header fragment missing")
	}
	if !strings.Contains(prompt, "line 01") {
		t.Error("expected line 1 to appear in rendered prompt")
	}
	if strings.Contains(prompt, "line %!") || strings.Contains(prompt, "line\n") {
		t.Error("control characters leaked into output")
	}
}

// TestBuildPromptWithContext_ClampsAtLastLine: finding.Line=20 (last) of
// a 20-line file with N=10 → block lines 10..20.
func TestBuildPromptWithContext_ClampsAtLastLine(t *testing.T) {
	src := fixtureSource(20)
	finding := types.Finding{File: "App.kt", Line: 20, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, _ := BuildPromptWithContext(finding, src, 10)

	if !strings.Contains(prompt, "Lines 10-20 of App.kt") {
		t.Errorf("expected clamp at bottom; got block header fragment missing")
	}
	if !strings.Contains(prompt, "line 20") {
		t.Error("expected last line 20 to appear in rendered prompt")
	}
	if strings.Contains(prompt, "line 21") {
		t.Error("line 21 should not appear; file has only 20 lines")
	}
}

// TestBuildPromptWithContext_SingleLineFile: file has exactly one line and
// finding.Line=1 → block contains just that one line, marker on it.
func TestBuildPromptWithContext_SingleLineFile(t *testing.T) {
	src := "line 01"
	finding := types.Finding{File: "App.kt", Line: 1, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, _ := BuildPromptWithContext(finding, src, 10)

	if !strings.Contains(prompt, "Lines 1-1 of App.kt") {
		t.Errorf("single-line file: expected block header 'Lines 1-1 of App.kt'; got prompt %q", prompt)
	}
	if !strings.Contains(prompt, "line 01") {
		t.Error("single-line file: source line should appear")
	}
}

// TestBuildPromptWithContext_DefensiveClampOutOfRangeLine: finding.Line is
// larger than numLines — the helper clamps and still emits the last lines.
func TestBuildPromptWithContext_DefensiveClampOutOfRangeLine(t *testing.T) {
	src := fixtureSource(10)
	finding := types.Finding{File: "App.kt", Line: 999, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, _ := BuildPromptWithContext(finding, src, 3)

	// Slice clamped to last 3 lines, marker on line 10 (the real last line).
	if !strings.Contains(prompt, "Lines 7-10 of App.kt") {
		t.Errorf("expected 'Lines 7-10 of App.kt'; clamping to last should still render")
	}
	if !strings.Contains(prompt, "line 10") {
		t.Error("last line should appear even when finding.Line is out of range")
	}
}

// TestBuildPromptWithContext_DefensiveClampZeroOrNegativeLine: finding.Line=0
// (data-quality issue from upstream SARIF) shouldn't crash; treats as 1.
func TestBuildPromptWithContext_DefensiveClampZeroOrNegativeLine(t *testing.T) {
	src := fixtureSource(20)
	finding := types.Finding{File: "App.kt", Line: 0, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, err := BuildPromptWithContext(finding, src, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "Issue at line 0") {
		t.Error("expected header to echo the original (zero) line so the LLM sees the malformed data")
	}
	// sliceRange(0, 10, 20) → target=0, start=0, end=min(11,20)=11 → block has 11 lines.
	if !strings.Contains(prompt, "Lines 1-11 of App.kt") {
		t.Errorf("expected clamped first block 'Lines 1-11 of App.kt'; got %q", prompt)
	}
}

// TestBuildPromptWithContext_EmptySource: sourceCode is "" — emits the
// "// (empty source)" placeholder instead of crashing.
func TestBuildPromptWithContext_EmptySource(t *testing.T) {
	finding := types.Finding{File: "App.kt", Line: 5, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, err := BuildPromptWithContext(finding, "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "(empty source)") {
		t.Errorf("expected placeholder for empty source; got %q", prompt)
	}
	if !strings.Contains(prompt, "Issue at line 5") {
		t.Error("header should still render Line=5 in the empty-source case")
	}
}

// TestBuildPromptWithContext_FallbackHintWhenFixHintEmpty: missing FixHint
// triggers the generic hint so the LLM still has guidance.
func TestBuildPromptWithContext_FallbackHintWhenFixHintEmpty(t *testing.T) {
	src := fixtureSource(15)
	finding := types.Finding{File: "App.kt", Line: 7, ID: "r", Cluster: "c", Severity: types.SeverityInfo,
		FixHint: ""}

	prompt, _ := BuildPromptWithContext(finding, src, 10)
	if !strings.Contains(prompt, genericChangeHint) {
		t.Error("expected generic fallback hint when FixHint is empty")
	}
	if strings.Contains(prompt, "Fix Hint:           ") { // FixHint line should not appear empty
		t.Errorf("FixHint empty should not introduce an empty 'Fix Hint' line")
	}
}

// TestBuildPromptWithContext_DefaultContextLinesWhenNonPositivePass: passing
// contextLines=0 (or negative) falls back to DefaultContextLines silently.
func TestBuildPromptWithContext_DefaultContextLinesWhenNonPositivePass(t *testing.T) {
	src := fixtureSource(60)
	finding := types.Finding{File: "App.kt", Line: 30, ID: "r", Cluster: "c", Severity: types.SeverityInfo}

	prompt, _ := BuildPromptWithContext(finding, src, 0)
	if !strings.Contains(prompt, "Lines 20-40 of App.kt") {
		t.Errorf("contextLines=0 should fall back to DefaultContextLines=10; expected 'Lines 20-40 of App.kt'")
	}

	prompt2, _ := BuildPromptWithContext(finding, src, -5)
	if !strings.Contains(prompt2, "Lines 20-40 of App.kt") {
		t.Errorf("contextLines=-5 should fall back to DefaultContextLines=10")
	}
}

// TestBuildPromptWithContext_HeaderAbsolutePathToBasename: full absolute
// paths in finding.File become just the basename — keeps the prompt
// focused and OS-agnostic.
func TestBuildPromptWithContext_HeaderAbsolutePathToBasename(t *testing.T) {
	src := fixtureSource(15)
	cases := []struct{ in, want string }{
		{"/home/user/proj/src/main/App.kt", "App.kt"},
		{"./src/main/App.kt", "App.kt"},
		{`C:\Users\X\proj\src\App.kt`, "App.kt"}, // Windows path
		{"src/main/App.kt", "App.kt"},
	}
	for _, c := range cases {
		finding := types.Finding{File: c.in, Line: 5, ID: "r", Cluster: "c", Severity: types.SeverityInfo}
		prompt, err := BuildPromptWithContext(finding, src, 10)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", c.in, err)
		}
		want := "File: " + c.want
		if !strings.Contains(prompt, want) {
			t.Errorf("for input %q, expected header fragment %q", c.in, want)
		}
		// Sanity: directory components must be stripped.
		if strings.Contains(prompt, "File: /") ||
			strings.Contains(prompt, "File: C:\\") ||
			strings.Contains(prompt, "File: ./") {
			t.Errorf("absolute/path-style File: leaked into prompt for input %q", c.in)
		}
	}
}

// TestBuildPromptWithContext_RoundTripsAndDoesNotLeakOffRangeSourceLines is
// a larger regression test that verifies off-range code is omitted.
func TestBuildPromptWithContext_RoundTripsAndDoesNotLeakOffRangeSourceLines(t *testing.T) {
	src := fixtureSource(50)
	finding := types.Finding{File: "App.kt", Line: 25, ID: "r", Cluster: "c", Severity: types.SeverityInfo,
		FixHint: "Apply hint."}

	prompt, _ := BuildPromptWithContext(finding, src, 5)

	// Block is lines 20-30.
	if !strings.Contains(prompt, "Lines 20-30 of App.kt") {
		t.Errorf("expected 'Lines 20-30 of App.kt'")
	}
	for _, ln := range []int{20, 25, 30} {
		if !strings.Contains(prompt, fmt.Sprintf("line %02d", ln)) {
			t.Errorf("expected line %d in block", ln)
		}
	}
	for _, ln := range []int{1, 15, 19, 31, 50} {
		if strings.Contains(prompt, fmt.Sprintf("line %02d", ln)) {
			t.Errorf("off-range line %d leaked", ln)
		}
	}
}

// TestSliceRange is a unit test of the pure slice helper, isolated from
// BuildPromptWithContext so future changes don't accidentally regress the
// edge-clamping math.
func TestSliceRange(t *testing.T) {
	tests := []struct {
		name               string
		line, n, total     int
		wantStart, wantEnd int
	}{
		{"first line + n + total", 1, 5, 20, 0, 6},           // lines 1..6
		{"middle line + n", 12, 5, 20, 6, 17},                // lines 7..17
		{"last line + n", 20, 5, 20, 14, 20},                 // lines 15..20
		{"line past end clamps to total", 999, 5, 10, 4, 10}, // target clamps to 9 → 4..10
		{"line 0 treated as first", 0, 5, 20, 0, 6},          // target clamps to 0
		{"line negative treated as first", -3, 5, 20, 0, 6},
		{"empty source", 5, 5, 0, 0, 0},
		{"n=10 at line 10 of 100 lines", 10, 10, 100, 0, 20}, // target=9 → [0,20) → 20 elements
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := sliceRange(tt.line, tt.n, tt.total)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("sliceRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.line, tt.n, tt.total, gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

// TestSplitLines is a unit test of the line-splitter edge cases.
func TestSplitLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"only newlines", "\n\n\n", nil},
		{"one line", "a", []string{"a"}},
		{"two lines + trailing nl", "a\nb\n", []string{"a", "b"}},
		{"two lines no trailing nl", "a\nb", []string{"a", "b"}},
		{"three lines", "a\nb\nc", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("%s: len got %d want %d (got=%v)", tt.name, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("%s: [%d] got %q want %q", tt.name, i, got[i], tt.want[i])
				}
			}
		})
	}
}
