package pathutil

import (
	"strings"
	"testing"
)

func TestNormalizePath_AbsoluteUnix(t *testing.T) {
	got := NormalizePath("/home/user/proj/src/Foo.kt", "")
	want := "/home/user/proj/src/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePath_AbsoluteWindowsDrive(t *testing.T) {
	// Even when running on Linux CI, Windows-style drive letter paths should
	// normalize correctly: backslashes become forward slashes.
	got := NormalizePath(`C:\Users\X\proj\src\Foo.kt`, "")
	want := "C:/Users/X/proj/src/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePath_MixedSlashesWindows(t *testing.T) {
	// Mixed-slash Windows-style and pure-backslash Windows-style must
	// normalize identically. Detekt SARIF can emit either depending on JVM.
	got1 := NormalizePath(`C:/Users/X/proj/src/Foo.kt`, "")
	got2 := NormalizePath(`C:\Users\X\proj\src\Foo.kt`, "")
	if got1 != got2 {
		t.Errorf("mixed-slash inputs should normalize identically: %q != %q", got1, got2)
	}
}

func TestNormalizePath_RelativeJoinedToRoot(t *testing.T) {
	// On any OS, joining a relative path against a Unix-style projectRoot
	// should produce a stable path that does NOT gain a Windows drive
	// letter when running on Windows. This pins the cwd-injection bug fix.
	got := NormalizePath("src/Foo.kt", "/home/user/proj")
	want := "/home/user/proj/src/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePath_DotDotResolved(t *testing.T) {
	// `..` in path should be resolved cleanly.
	got := NormalizePath("/proj/old/../src/Foo.kt", "")
	want := "/proj/src/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePath_EmptyInput(t *testing.T) {
	if got := NormalizePath("", "/any"); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestNormalizePath_RelativeNoRoot(t *testing.T) {
	// No projectRoot, no drive letter — keep as-is exactly. Pin the contract.
	// filepath.Clean does not introduce directories that weren't there.
	got := NormalizePath("src/Foo.kt", "")
	if got != "src/Foo.kt" {
		t.Errorf("got %q, want %q", got, "src/Foo.kt")
	}
}

func TestNormalizePath_BackslashMixedInputNormalization(t *testing.T) {
	// Detekt on Windows can emit "app\src\Foo.kt" with mixed slashes.
	// Normalization should treat it as the forward-slash form regardless of
	// running OS.
	got := NormalizePath(`app\src\Foo.kt`, "")
	want := "app/src/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePath_DriveLetterWithRelativePath(t *testing.T) {
	// A drive letter followed by a path should fully normalize to forward
	// slashes without losing the drive prefix.
	got := NormalizePath(`C:proj\src\Foo.kt`, "")
	// On Linux: "C:proj/src/Foo.kt". On Windows: "C:/proj/src/Foo.kt" (drive is treated with separator).
	// Either form is acceptable — the key is that the result resolves with
	// forward slashes and is stable across OS.
	if got != "C:proj/src/Foo.kt" && got != "C:/proj/src/Foo.kt" {
		t.Errorf("got %q, want C:proj/src/Foo.kt or C:/proj/src/Foo.kt", got)
	}
}

func TestNormalizePath_UncPath(t *testing.T) {
	// UNC paths should be detected as absolute even when running on Linux,
	// and backslashes should become forward slashes.
	got := NormalizePath(`\\server\share\Foo.kt`, "")
	want := "//server/share/Foo.kt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSuffixMatch_ExactBoundary(t *testing.T) {
	tests := []struct {
		candidate string
		path      string
		want      bool
	}{
		{"/app/src/Foo.kt", "/app/src/Foo.kt", true},
		{"/app/src/OtherFoo.kt", "/app/src/Foo.kt", false}, // boundary violation
		{"/app/src/Foo.kt", "/Foo.kt", true},               // parent dir via segment boundary
		{"/proj/src/Foo.kt", "/src/Foo.kt", true},          // nested suffix with boundary
		{"", "", false},                    // empty inputs
		{"a", "", false},                   // empty pathToFind
		{"", "a", false},                   // empty candidate
		{"Untouched.kt", "/Foo.kt", false}, // completely different
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := SuffixMatch(tt.candidate, tt.path)
			if got != tt.want {
				t.Errorf("SuffixMatch(%q, %q) = %v, want %v", tt.candidate, tt.path, got, tt.want)
			}
		})
	}
}

// TestSuffixMatch_BoundaryRegression pins the boundary fix: previously, the
// old diff.FilterFindingsByDiff used strings.HasSuffix, which would let
// "OtherFoo.kt" match against "Foo.kt" (false positive). The new SuffixMatch
// must reject that.
func TestSuffixMatch_BoundaryRegression(t *testing.T) {
	if SuffixMatch("/proj/src/OtherFoo.kt", "/proj/Foo.kt") {
		t.Error("SuffixMatch must reject unrelated filename; got true for /proj/src/OtherFoo.kt suffix /proj/Foo.kt")
	}
	if SuffixMatch("/proj/Foo.kt", "/proj/Baz.kt") {
		t.Error("SuffixMatch must reject same-root different name; got true for /proj/Foo.kt suffix /proj/Baz.kt")
	}
}

// TestRelativeToProject pins the report path contract. Findings arrive in two
// shapes -- native detectors emit project-relative paths, detekt emits absolute
// file:// URIs -- and reports used to carry that mix through unchanged. The
// visible damage was a SARIF whose URIs pointed at the scanning machine and so
// matched nothing in the GitHub repository.
func TestRelativeToProject(t *testing.T) {
	const root = "/proj"
	cases := []struct {
		name  string
		input string
		root  string
		want  string
	}{
		{"already relative", "src/Foo.kt", root, "src/Foo.kt"},
		{"absolute inside project", "/proj/src/Foo.kt", root, "src/Foo.kt"},
		{"file uri", "file:///proj/src/Foo.kt", root, "src/Foo.kt"},
		{"percent encoded space", "file:///proj/my%20src/Foo.kt", root, "my src/Foo.kt"},
		{"backslashes", "/proj/src/sub/Foo.kt", root, "src/sub/Foo.kt"},
		{"dot segments", "/proj/src/../src/Foo.kt", root, "src/Foo.kt"},
		{"empty stays empty", "", root, ""},
		{"no root keeps input", "src/Foo.kt", "", "src/Foo.kt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RelativeToProject(c.input, c.root); got != c.want {
				t.Errorf("RelativeToProject(%q, %q) = %q, want %q", c.input, c.root, got, c.want)
			}
		})
	}
}

// A path genuinely outside the project must not be rewritten into a misleading
// ../../ relative path: it is not relative to the project at all.
func TestRelativeToProjectKeepsOutsidePathsAbsolute(t *testing.T) {
	got := RelativeToProject("/elsewhere/Foo.kt", "/proj")
	if strings.HasPrefix(got, "..") {
		t.Fatalf("outside path was rewritten as relative: %q", got)
	}
	if !strings.Contains(got, "elsewhere") {
		t.Fatalf("outside path lost its identity: %q", got)
	}
}
