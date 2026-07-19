package diff

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/pathutil"
	"github.com/adkd/adkd/internal/core/types"
)

func TestParseDiff(t *testing.T) {
	diffOutput := []byte(`diff --git a/app/src/main/java/com/example/MyFile.kt b/app/src/main/java/com/example/MyFile.kt
index 83db48f..b2931a5 100644
--- a/app/src/main/java/com/example/MyFile.kt
+++ b/app/src/main/java/com/example/MyFile.kt
@@ -10,0 +11,2 @@
+import foo
+import bar
@@ -20,2 +22 @@
-val old = 1
-val old2 = 2
+val new = 3
diff --git a/OtherFile.kt b/OtherFile.kt
index 1234567..89abcdef 100644
--- a/OtherFile.kt
+++ b/OtherFile.kt
@@ -5 +5 @@
-val a = 1
+val b = 2
`)

	expected := map[string][]LineRange{
		"app/src/main/java/com/example/MyFile.kt": {
			{Start: 11, Count: 2},
			{Start: 22, Count: 1},
		},
		"OtherFile.kt": {
			{Start: 5, Count: 1},
		},
	}

	got := parseDiff(diffOutput)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("parseDiff() = %v, want %v", got, expected)
	}
}

// TestFilterFindingsByDiff is preserved verbatim from the round-1 test suite.
// The wrapper's behavior (projectRoot="") must continue to match the same
// round-1 expectations, so prior smoke tests and integration with scan.go
// remain intact.
func TestFilterFindingsByDiff(t *testing.T) {
	diffMap := map[string][]LineRange{
		"app/src/main/java/com/example/MyFile.kt": {
			{Start: 11, Count: 2},
		},
		"OtherFile.kt": {
			{Start: 5, Count: 1},
		},
	}

	findings := []types.Finding{
		{
			ID:   "finding1",
			File: "C:\\Users\\Miguel\\project\\app\\src\\main\\java\\com\\example\\MyFile.kt",
			Line: 10, // Not in range
		},
		{
			ID:   "finding2",
			File: "C:/Users/Miguel/project/app/src/main/java/com/example/MyFile.kt",
			Line: 11, // In range
		},
		{
			ID:   "finding3",
			File: "app/src/main/java/com/example/MyFile.kt",
			Line: 12, // In range
		},
		{
			ID:   "finding4",
			File: "OtherFile.kt",
			Line: 5, // In range
		},
		{
			ID:   "finding5",
			File: "OtherFile.kt",
			Line: 6, // Not in range
		},
		{
			ID:   "finding6",
			File: "UntouchedFile.kt",
			Line: 1, // File not in diff
		},
	}

	filtered := FilterFindingsByDiff(findings, diffMap)

	if len(filtered) != 3 {
		t.Errorf("expected 3 findings, got %d", len(filtered))
	}

	expectedIDs := map[string]bool{
		"finding2": true,
		"finding3": true,
		"finding4": true,
	}

	for _, f := range filtered {
		if !expectedIDs[f.ID] {
			t.Errorf("unexpected finding kept: %s", f.ID)
		}
	}
}

// TestFilterFindingsByDiffWithRoot_RelativeJoinedToAbsolute verifies that
// when projectRoot is passed, a relative finding.File is joined against the
// root before matching against diffMap keys that are themselves relative to
// the same root (typical `git diff` output).
func TestFilterFindingsByDiffWithRoot_RelativeJoinedToAbsolute(t *testing.T) {
	diffMap := map[string][]LineRange{
		"src/main/Foo.kt": {
			{Start: 1, Count: 5},
		},
	}

	projectRoot := "/home/user/proj"

	findings := []types.Finding{
		{ID: "relative-in-range", File: "src/main/Foo.kt", Line: 3},                // relative, in range
		{ID: "absolute-in-range", File: projectRoot + "/src/main/Foo.kt", Line: 3}, // absolute, in range
		{ID: "untouched-file", File: "src/main/Bar.kt", Line: 3},                   // not in diffMap
		{ID: "out-of-range", File: "src/main/Foo.kt", Line: 99},                    // in file but outside range
	}

	filtered := FilterFindingsByDiffWithRoot(findings, diffMap, projectRoot)

	if len(filtered) != 2 {
		t.Errorf("expected 2 findings, got %d", len(filtered))
	}
	gotIDs := map[string]bool{}
	for _, f := range filtered {
		gotIDs[f.ID] = true
	}
	if !gotIDs["relative-in-range"] || !gotIDs["absolute-in-range"] {
		t.Errorf("expected relative-in-range and absolute-in-range to survive, got %v", gotIDs)
	}
	if gotIDs["untouched-file"] || gotIDs["out-of-range"] {
		t.Errorf("untouched-file and out-of-range should be dropped, got %v", gotIDs)
	}
}

// TestFilterFindingsByDiffWithRoot_BoundaryRegression pins the fix for the
// round-1 strings.HasSuffix fragility. A finding in "OtherFoo.kt" must NOT
// match a diffMap entry of "Foo.kt" — previously HasSuffix would let it
// through as a false positive.
func TestFilterFindingsByDiffWithRoot_BoundaryRegression(t *testing.T) {
	diffMap := map[string][]LineRange{
		"Foo.kt": {
			{Start: 1, Count: 5},
		},
	}

	findings := []types.Finding{
		{ID: "true-positive", File: "OtherFiles/Foo.kt", Line: 3}, // suffix "Foo.kt" with boundary
		{ID: "false-positive", File: "OtherFoo.kt", Line: 3},      // bare "OtherFoo.kt" — NO boundary
		{ID: "different-file", File: "Bar.kt", Line: 3},           // unrelated
	}

	filtered := FilterFindingsByDiffWithRoot(findings, diffMap, "")

	if len(filtered) != 1 {
		t.Fatalf("expected 1 finding (true-positive), got %d", len(filtered))
	}
	if filtered[0].ID != "true-positive" {
		t.Errorf("expected true-positive, got %s", filtered[0].ID)
	}
}

// TestFilterFindingsByDiffWithRoot_WindowsPathMixedSlashes verifies that
// both pure-backslash and mixed-slash Windows-style paths normalize to the
// same canonical form before matching.
func TestFilterFindingsByDiffWithRoot_WindowsPathMixedSlashes(t *testing.T) {
	diffMap := map[string][]LineRange{
		"src/main/Foo.kt": {
			{Start: 1, Count: 5},
		},
	}

	findings := []types.Finding{
		{ID: "mixed", File: `C:/Users/X/proj/src/main/Foo.kt`, Line: 3},
		{ID: "backslash", File: `C:\Users\X\proj\src\main\Foo.kt`, Line: 3},
	}

	filtered := FilterFindingsByDiffWithRoot(findings, diffMap, "")
	if len(filtered) != 2 {
		t.Errorf("expected both Windows-style findings to survive after path normalization, got %d", len(filtered))
	}
}

// Helper for ensuring pathutil output is well-formed in cross-platform tests.
func TestNormalizePath_RoundTrip(t *testing.T) {
	cases := []struct{ in, root, want string }{
		{`C:\proj\src\Foo.kt`, `C:\proj`, `C:/proj/src/Foo.kt`},
		{`src/Foo.kt`, `/root/proj`, `/root/proj/src/Foo.kt`},
	}
	for _, c := range cases {
		got := pathutil.NormalizePath(c.in, c.root)
		if got != c.want {
			t.Errorf("NormalizePath(%q, %q) = %q, want %q", c.in, c.root, got, c.want)
		}
	}
}

// TestGetGitRoot_NotInGitRepo verifies that GetGitRoot returns a
// non-nil error when called from a directory that is not inside a git
// repository, AND that the error wraps a recognisable cause (the
// `git rev-parse` subprocess invocation). The integration in scan.go
// falls back to `wd` when this errors, so a stale or generic error
// message would silently mislead users on monorepo submodules.
func TestGetGitRoot_NotInGitRepo(t *testing.T) {
	tmp := t.TempDir() // not a git repo
	_, err := GetGitRoot(tmp)
	if err == nil {
		t.Fatal("GetGitRoot on a non-git directory should return error; got nil")
	}
	if !strings.Contains(err.Error(), "git rev-parse") {
		t.Errorf("error %q should mention 'git rev-parse'; guards against future refactors returning a generic error", err)
	}
}
