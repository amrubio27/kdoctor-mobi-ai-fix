package diff

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/adkd/adkd/internal/core/pathutil"
	"github.com/adkd/adkd/internal/core/types"
)

type LineRange struct {
	Start int
	Count int
}

// GetMergeBase returns the merge base between the given ref and HEAD.
// Note: git is invoked with cmd.Dir=projectDir and walks up to find the
// git root. Paths that GetLineDiff returns are relative to that git
// root, NOT to projectDir. If projectDir is a subdirectory of the git
// root, callers should use GetGitRoot to establish the correct root for
// path normalization (see FilterFindingsByDiffWithRoot's godoc).
func GetMergeBase(ref string, projectDir string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref, "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get merge base for %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetGitRoot walks up from projectDir to find the git top-level using
// `git rev-parse --show-toplevel`. Returns the absolute path of the git
// root, or an error if projectDir is not inside a git repo. Round-2
// task #5 introduced this helper so callers can pass the true git root
// as `projectRoot` to FilterFindingsByDiffWithRoot when --project-dir
// is a submodule/subdirectory of the git repo (path equality required
// for the git-root vs projectRoot case).
func GetGitRoot(projectDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed (not in a git repo?): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetLineDiff returns a map of changed files (relative paths) to their
// modified/added line ranges. Paths in the map come from `git diff` and are
// therefore relative to the git root of projectDir.
func GetLineDiff(baseRef string, projectDir string) (map[string][]LineRange, error) {
	// -U0 gets diff without context lines so each hunk maps to a tight range.
	cmd := exec.Command("git", "diff", baseRef, "-U0")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	return parseDiff(out), nil
}

// @@ -oldStart,oldCount +newStart,newCount @@
var hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
var diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)`)

func parseDiff(diffOutput []byte) map[string][]LineRange {
	result := make(map[string][]LineRange)
	lines := bytes.Split(diffOutput, []byte("\n"))

	var currentFile string

	for _, lineBytes := range lines {
		line := string(lineBytes)
		if strings.HasPrefix(line, "diff --git") {
			matches := diffHeaderRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				currentFile = matches[2]
			}
			continue
		}

		if strings.HasPrefix(line, "@@ ") && currentFile != "" {
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				start, _ := strconv.Atoi(matches[1])
				count := 1 // Default count is 1 if omitted
				if len(matches) >= 3 && matches[2] != "" {
					count, _ = strconv.Atoi(matches[2])
				}

				// Only tracking additions/modifications (count > 0)
				if count > 0 {
					result[currentFile] = append(result[currentFile], LineRange{
						Start: start,
						Count: count,
					})
				}
			}
		}
	}

	return result
}

// FilterFindingsByDiff is preserved as a backward-compatible wrapper for
// callers and tests that do not yet pass a project root. New callers should
// use FilterFindingsByDiffWithRoot directly.
func FilterFindingsByDiff(findings []types.Finding, diffMap map[string][]LineRange) []types.Finding {
	return FilterFindingsByDiffWithRoot(findings, diffMap, "")
}

// FilterFindingsByDiffWithRoot filters findings to those whose File location
// is in a modified/added line range according to the git diff map. The
// projectRoot is used to absolutize relative paths so matching is robust
// against detekt SARIF emitting mixed absolute/relative paths and across
// Windows vs Unix runners.
//
// IMPORTANT — git-root vs projectRoot: git is invoked with cmd.Dir set to
// the scanned working directory. If that directory is a subdirectory of
// the git root (e.g. a Gradle module), git walks up and emits diff paths
// RELATIVE TO THE GIT ROOT, not to projectRoot. Callers should detect the
// real git root with GetGitRoot and pass that as `projectRoot` here. The
// current scan.go integration passes `wd` (the --project-dir value) which
// is correct ONLY when --project-dir equals the git root.
//
// Matching strategy (in order):
//  1. Exact match after pathutil.NormalizePath on both finding.File and
//     each diffMap key, joined against projectRoot when needed.
//  2. Boundary-aware suffix match via pathutil.SuffixMatch, which prevents
//     "OtherFoo.kt" from matching against "Foo.kt" (a known fragility of
//     plain strings.HasSuffix used in the round-1 implementation).
//
// A finding only survives if its line falls within at least one of the
// matched file's LineRanges.
//
// Limitations:
//   - If two diffMap keys normalize to the same string (e.g. "Foo.kt" and
//     "./Foo.kt"), only the last wins. This is rare in real-world diffs.
func FilterFindingsByDiffWithRoot(findings []types.Finding, diffMap map[string][]LineRange, projectRoot string) []types.Finding {
	var filtered []types.Finding

	// Pre-normalize diffMap keys so the inner loop is cheaper and the
	// comparison logic is symmetric with the finding.File normalization.
	normalizedDiffMap := make(map[string][]LineRange, len(diffMap))
	for k, v := range diffMap {
		normKey := pathutil.NormalizePath(k, projectRoot)
		normalizedDiffMap[normKey] = v
	}

	for _, finding := range findings {
		normFinding := pathutil.NormalizePath(finding.File, projectRoot)

		var matchedRanges []LineRange
		if ranges, ok := normalizedDiffMap[normFinding]; ok {
			matchedRanges = ranges
		} else {
			for diffKey, ranges := range normalizedDiffMap {
				if pathutil.SuffixMatch(normFinding, diffKey) {
					matchedRanges = ranges
					break
				}
			}
		}

		if matchedRanges == nil {
			continue
		}

		inRange := false
		for _, r := range matchedRanges {
			if finding.Line >= r.Start && finding.Line < r.Start+r.Count {
				inRange = true
				break
			}
		}
		if inRange {
			filtered = append(filtered, finding)
		}
	}

	return filtered
}
