// Package pathutil centralizes path normalization helpers used across kdoctor
// packages (diff, baseline, scan) where findings emitted by different sources
// (detekt SARIF, git diff output, detekt baseline.xml) need to be matched
// against each other despite differing in absolute/relative form, separator
// style (forward vs back slashes), or which OS the path style represents.
//
// Why this exists: prior to round-2 task #5, FilterFindingsByDiff used
// strings.HasSuffix (which lets "OtherFoo.kt" falsely match against "Foo.kt")
// and IsSuppressed used strings.Contains on raw paths with no absolutization
// (which lets substrings act as false positives). This package introduces a
// single, testable normalization step that callers reuse.
package pathutil

import (
	"net/url"
	"path/filepath"
	"strings"
)

// NormalizePath canonicalizes `input` against `projectRoot` so paths from
// different sources can be compared as strings.
//
// Rules:
//   - Empty input returns empty output.
//   - If `input` is absolute (Unix-style "/x", Windows-style "C:\x" / "C:/x"
//     / UNC "\\server\share" / drive-only "C:"), it is kept.
//   - If `input` is relative and `projectRoot` is non-empty, they are joined
//     so the result is rooted against `projectRoot`.
//   - If `input` is relative and `projectRoot` is empty, the path is kept
//     as-is. This preserves git-root-relative behavior.
//   - `..` segments are resolved via filepath.Clean.
//   - filepath.Abs is invoked only when the host OS recognizes the result
//     as absolute (filepath.IsAbs true). This avoids cwd pollution when
//     cross-platform strings appear on a runner of a different OS — e.g.
//     "/foo" on Windows (where IsAbs returns false for it) stays literal
//     rather than being prepended with a drive letter.
//   - All separators become forward slashes for uniform comparison.
//
// No locale-specific case folding is performed. Windows-style paths
// (drive letter) on Linux CI are NOT lowercased — that decision belongs to
// the caller via separate comparison logic if cross-platform case
// insensitivity is required.
func NormalizePath(input, projectRoot string) string {
	if input == "" {
		return ""
	}

	path := input
	if !isAbsoluteLike(input) && projectRoot != "" {
		path = filepath.Join(projectRoot, input)
	}

	path = filepath.Clean(path)

	// Only invoke filepath.Abs when the host OS recognizes this path style
	// as absolute. Otherwise the engine would prepend the current working
	// directory, breaking cross-platform tests and silently mutating paths.
	if filepath.IsAbs(path) {
		if p, err := filepath.Abs(path); err == nil {
			path = p
		}
	}

	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

// SuffixMatch returns true if `candidate` ends with `pathToFind` at a
// directory boundary. Returns false if either argument is empty.
//
// This prevents false positives that plain strings.HasSuffix would allow:
//   - SuffixMatch("/proj/src/Foo.kt", "/Foo.kt")                    = true
//   - SuffixMatch("/proj/src/OtherFoo.kt", "/Foo.kt")               = false
//   - SuffixMatch("/proj/src/Foo.kt", "/proj/src/Foo.kt")           = true (exact)
//   - SuffixMatch("untouched.kt", "/proj/src/Foo.kt")                = false
//
// Both inputs must already be normalized via NormalizePath (forward slashes)
// for the boundary detection to behave correctly.
func SuffixMatch(candidate, pathToFind string) bool {
	if candidate == "" || pathToFind == "" {
		return false
	}
	// Add a leading "/" so strings.HasSuffix compares whole path segments
	// rather than substrings that happen to share a textual suffix.
	c := "/" + strings.TrimPrefix(candidate, "/")
	p := "/" + strings.TrimPrefix(pathToFind, "/")
	return c == p || strings.HasSuffix(c, p)
}

// isAbsoluteLike determines whether a path string represents an absolute
// path. Unlike filepath.IsAbs (which depends on the running OS), this is
// permissive and cross-platform: it accepts Unix absolute paths "/x" and
// Windows absolute paths "C:\x" / "C:/x" / "C:" on any platform. This
// makes tests with Windows-style fixtures deterministic on Linux CI while
// preserving the round-1 behavior of `strings.HasSuffix`-based matching.
func isAbsoluteLike(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return true
	}
	// Windows drive letter: "C:" followed by "/", "\", or end-of-string.
	if len(s) >= 2 && s[1] == ':' && isLetter(s[0]) {
		if len(s) == 2 {
			return true
		}
		if s[2] == '/' || s[2] == '\\' {
			return true
		}
	}
	// UNC paths "\\server\share\..." on any OS.
	if len(s) >= 2 && (s[0] == '\\' || s[0] == '/') && (s[1] == '\\' || s[1] == '/') {
		return true
	}
	return filepath.IsAbs(s)
}

func isLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// RelativeToProject renders a finding path the way a report should carry it:
// relative to the project, with forward slashes.
//
// Findings arrive in two shapes. Native Go detectors emit project-relative
// paths; detekt emits absolute file:// URIs. Leaving that mix in the report
// meant the SARIF uploaded to GitHub Code Scanning carried URIs like
// file:///C:/Users/<someone>/project/src/Foo.kt, which resolve to nothing in
// the repository, and every JSON report leaked the scanning machine layout.
//
// Paths outside projectRoot are returned cleaned but still absolute: they are
// genuinely outside the project, and silently rewriting them would be a lie.
func RelativeToProject(input, projectRoot string) string {
	if input == "" {
		return ""
	}

	path := stripFileURI(input)
	if projectRoot == "" {
		return filepath.ToSlash(filepath.Clean(path))
	}

	abs := NormalizePath(path, projectRoot)
	root := NormalizePath(projectRoot, "")

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	rel = filepath.ToSlash(rel)
	// A result that climbs out of the project is not "relative to" it.
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return abs
	}
	return rel
}

// stripFileURI turns a file:// URI into a plain path. detekt SARIF reports
// use them; percent-encoding (e.g. %20 for spaces) has to be undone or the
// path will not match anything on disk.
func stripFileURI(input string) string {
	const scheme = "file://"
	if !strings.HasPrefix(input, scheme) {
		return input
	}
	path := strings.TrimPrefix(input, scheme)
	// file:///C:/x -> /C:/x ; drop the leading slash before a drive letter.
	if len(path) > 2 && path[0] == '/' && isLetter(path[1]) && path[2] == ':' {
		path = path[1:]
	}
	if unescaped, err := url.PathUnescape(path); err == nil {
		path = unescaped
	}
	return path
}
