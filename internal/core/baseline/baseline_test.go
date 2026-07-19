package baseline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestLoadBaseline(t *testing.T) {
	xmlContent := `<?xml version="1.0" ?>
<SmellBaseline>
  <ManuallySuppressedIssues>
    <ID>MagicNumber:MyFile.kt:42</ID>
  </ManuallySuppressedIssues>
  <CurrentIssues>
    <ID>UnusedImports:app/src/main/java/com/example/MyFile.kt:import foo</ID>
    <ID>MaxLineLength:AnotherFile.kt</ID>
  </CurrentIssues>
</SmellBaseline>
`
	tmpFile := filepath.Join(t.TempDir(), "baseline.xml")
	err := os.WriteFile(tmpFile, []byte(xmlContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := LoadBaseline(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d", len(ids))
	}

	if !ids["MagicNumber:MyFile.kt:42"] {
		t.Errorf("missing MagicNumber id")
	}
	if !ids["UnusedImports:app/src/main/java/com/example/MyFile.kt:import foo"] {
		t.Errorf("missing UnusedImports id")
	}
}

// TestIsSuppressed is preserved verbatim from the round-1 test suite. The
// wrapper's behavior (projectRoot="") must continue to match the same
// round-1 expectations.
func TestIsSuppressed(t *testing.T) {
	baselineIDs := map[string]bool{
		"UnusedImports:app/src/main/java/com/example/MyFile.kt:import foo": true,
		"MagicNumber:MyFile.kt:42": true,
	}

	tests := []struct {
		name       string
		finding    types.Finding
		suppressed bool
	}{
		{
			name: "exact match path",
			finding: types.Finding{
				Rule: "detekt.style.UnusedImports",
				File: "app/src/main/java/com/example/MyFile.kt",
			},
			suppressed: true,
		},
		{
			name: "substring match path finding to baseline",
			finding: types.Finding{
				Rule: "detekt.style.UnusedImports",
				File: "C:\\Users\\Miguel\\project\\app\\src\\main\\java\\com\\example\\MyFile.kt",
			},
			suppressed: true,
		},
		{
			name: "different rule",
			finding: types.Finding{
				Rule: "detekt.style.WildcardImport",
				File: "app/src/main/java/com/example/MyFile.kt",
			},
			suppressed: false,
		},
		{
			name: "different path",
			finding: types.Finding{
				Rule: "detekt.style.UnusedImports",
				File: "app/src/main/java/com/example/OtherFile.kt",
			},
			suppressed: false,
		},
		{
			name: "MagicNumber match",
			finding: types.Finding{
				Rule: "detekt.style.MagicNumber",
				File: "C:/Users/Miguel/project/MyFile.kt",
			},
			suppressed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSuppressed(tt.finding, baselineIDs)
			if got != tt.suppressed {
				t.Errorf("IsSuppressed() = %v, want %v", got, tt.suppressed)
			}
		})
	}
}

// TestIsSuppressedWithRoot_BoundaryRegression pins the fix for the round-1
// strings.Contains fragility. A finding in "OtherMyFile.kt" must NOT match a
// baseline ID whose path is just "MyFile.kt" — previously Contains would
// let it through as a false positive.
func TestIsSuppressedWithRoot_BoundaryRegression(t *testing.T) {
	baselineIDs := map[string]bool{
		"UnusedImports:MyFile.kt:any-sig": true,
	}

	findings := []struct {
		file string
		want bool
	}{
		{"src/MyFile.kt", true},       // suffix "MyFile.kt" with directory boundary — matches
		{"/abs/MyFile.kt", true},      // absolute with boundary — matches
		{"src/OtherMyFile.kt", false}, // bare-string contains match without boundary — REJECT
		{"/abs/Unrelated.kt", false},  // unrelated
		{"MyFile.kt.bak", false},      // accidentally-adjacent name — REJECT
	}

	for _, f := range findings {
		got := IsSuppressedWithRoot(types.Finding{
			Rule: "detekt.style.UnusedImports",
			File: f.file,
		}, baselineIDs, "")
		if got != f.want {
			t.Errorf("IsSuppressedWithRoot(file=%q) = %v, want %v", f.file, got, f.want)
		}
	}
}

// TestIsSuppressedWithRoot_ProjectRoot verifies that when projectRoot is
// passed, the baseline's relative path is joined against the root. This is
// the realistic case in scan.go: --project-dir is provided.
func TestIsSuppressedWithRoot_ProjectRoot(t *testing.T) {
	baselineIDs := map[string]bool{
		"UnusedImports:src/main/Foo.kt:import statement": true,
	}

	projectRoot := "/home/user/proj"

	findings := []types.Finding{
		{Rule: "detekt.style.UnusedImports", File: projectRoot + "/src/main/Foo.kt"}, // absolutized — matches
		{Rule: "detekt.style.UnusedImports", File: "src/main/Foo.kt"},                // relative, joined via root — matches
		{Rule: "detekt.style.UnusedImports", File: projectRoot + "/src/main/Bar.kt"}, // unrelated file
		{Rule: "detekt.style.OtherRule", File: projectRoot + "/src/main/Foo.kt"},     // different rule
	}

	if !IsSuppressedWithRoot(findings[0], baselineIDs, projectRoot) {
		t.Error("absolute finding should be suppressed when projectRoot is provided")
	}
	if !IsSuppressedWithRoot(findings[1], baselineIDs, projectRoot) {
		t.Error("relative finding should be suppressed when projectRoot is provided")
	}
	if IsSuppressedWithRoot(findings[2], baselineIDs, projectRoot) {
		t.Error("unrelated file should NOT be suppressed")
	}
	if IsSuppressedWithRoot(findings[3], baselineIDs, projectRoot) {
		t.Error("different rule should NOT be suppressed even for matching path")
	}
}

// TestIsSuppressedWithRoot_EmptyRuleDefensive pins the defensive check
// added in round-2 task #5: if finding.Rule is empty, IsSuppressed must
// return false unconditionally (rather than allow a parts[0]=="" match
// against any baseline ID). Also covers the symmetric defensive check
// for empty-path baseline IDs ("Rule::no-path" entries are silently
// skipped). Regression guard for refactors that might silently remove
// either defensive guard.
func TestIsSuppressedWithRoot_EmptyRuleDefensive(t *testing.T) {
	baselineIDs := map[string]bool{
		"UnusedImports:src/Foo.kt:import stmt": true,
		"UnusedImports::no-path":               true, // edge case: empty path between the two colons
		":src/Foo.kt":                          true, // edge case: rule-empty baseline ID
	}
	// Empty finding.Rule must short-circuit to false regardless.
	if IsSuppressedWithRoot(types.Finding{Rule: "", File: "src/Foo.kt"}, baselineIDs, "") {
		t.Error("empty finding.Rule should NOT match any baseline ID (defensive check)")
	}
	// Non-empty rule with matching baseline must be suppressed.
	if !IsSuppressedWithRoot(types.Finding{Rule: "detekt.style.UnusedImports", File: "src/Foo.kt"}, baselineIDs, "") {
		t.Error("non-empty rule with matching baseline should be suppressed")
	}
	// Empty-path baseline ID must NOT suppress unrelated findings — the
	// `len(idParts) < 2` skip in IsSuppressedWithRoot handles this.
	if IsSuppressedWithRoot(types.Finding{Rule: "detekt.style.WildcardImport", File: "src/Any.kt"}, baselineIDs, "") {
		t.Error("empty-path baseline ID should be silently ignored, not match unrelated findings")
	}
}
