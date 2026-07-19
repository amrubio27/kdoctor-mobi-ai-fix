// main_test.go exposes the evalprojects CLI as a go-test smoke test.
//
// On `go test ./...`:
//  1. TestMain builds a kdoctor binary into os.TempDir() (or reuses an
//     existing repo-root `kdoctor.exe`) and tracks it for cleanup.
//  2. For each examples/scoring-fixtures/*.json: reads Fixture, runs
//     `kdoctor scan --json` against the project path with detekt-cli,
//     parses the report, asserts schemaVersion, score band, and the
//     mustIncludeFinding list.
//
// Tests SKIP (not FAIL) when prerequisites are missing (no detekt-cli,
// no kdoctor binary buildable). This keeps `go test ./...` clean for
// CI environments that haven't installed detekt yet.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

var kdoctorTestBin string // Built/reused by TestMain; cleaned up on exit.

func TestMain(m *testing.M) {
	kdoctorTestBin = buildOrLocateKdoctor()
	if kdoctorTestBin == "" {
		fmt.Fprintln(os.Stderr, "=== evalprojects: could not build kdoctor — tests will SKIP ===")
	} else {
		fmt.Fprintf(os.Stderr, "=== evalprojects: kdoctor binary = %s\n", kdoctorTestBin)
	}
	code := m.Run()
	if kdoctorTestBin != "" && strings.HasPrefix(kdoctorTestBin, os.TempDir()) {
		_ = os.Remove(kdoctorTestBin)
	}
	os.Exit(code)
}

// buildOrLocateKdoctor prefers an existing repo-root binary (faster path
// for local dev). Falls back to `go build ./cmd/kdoctor` into os.TempDir()
// for CI environments.
func buildOrLocateKdoctor() string {
	root, ok := repoRoot()
	if !ok {
		return ""
	}
	for _, name := range []string{"kdoctor.exe", "kdoctor"} {
		c := filepath.Join(root, name)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	tmpName := "kdoctor-smoketest"
	if runtime.GOOS == "windows" {
		tmpName += ".exe"
	}
	out := filepath.Join(os.TempDir(), tmpName)
	cmd := exec.Command("go", "build", "-o", out, "./cmd/kdoctor")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build kdoctor: %v\n", err)
		return ""
	}
	return out
}

// repoRoot ascends from cwd until it finds go.mod. Test files in nested
// packages need this because Detekt's working directory matters.
func repoRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	wd, _ = filepath.Abs(wd)
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, true
		}
		wd = filepath.Dir(wd)
	}
	return "", false
}

// detectDetektBinary returns the absolute path to a detekt-cli launcher.
// Order: KDOCTOR_DETEKT_BIN env var → Windows-specific D:\tools\detekt.cmd
// → POSIX /d/tools/detekt.cmd → /opt/detekt/bin/detekt → /usr/local/bin/detekt
// → PATH lookup. Returns "" if not found (caller SKIPs the test).
func detectDetektBinary() string {
	if v := os.Getenv("KDOCTOR_DETEKT_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	candidates := []string{}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, `D:\tools\detekt.cmd`)
	} else {
		candidates = append(candidates, "/d/tools/detekt.cmd")
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("detekt"); err == nil {
		return p
	}
	return ""
}

func TestEvalBadProjectFixture(t *testing.T) {
	root, ok := repoRoot()
	if !ok {
		t.Skip("could not locate repo root")
	}
	runFixture(t, filepath.Join(root, "examples", "scoring-fixtures", "bad.json"), false)
}

// TestEvalBadProjectFixtureRelative regression-guards the runner.go
// filepath.Abs fix (Fase 1 close-out). Same fixture, but --project-dir
// is passed RELATIVE to repo root which is the working directory. If
// runner.go regresses (passes a relative path straight to detekt's
// `--input` which then resolves relative to cmd.Dir), this test fails.
func TestEvalBadProjectFixtureRelative(t *testing.T) {
	root, ok := repoRoot()
	if !ok {
		t.Skip("could not locate repo root")
	}
	runFixture(t, filepath.Join(root, "examples", "scoring-fixtures", "bad.json"), true)
}

func TestEvalGoodProjectFixture(t *testing.T) {
	root, ok := repoRoot()
	if !ok {
		t.Skip("could not locate repo root")
	}
	runFixture(t, filepath.Join(root, "examples", "scoring-fixtures", "good.json"), false)
}

// runFixture executes kdoctor scan --json on each fixture's projectPath and
// asserts schema version, score band, and required findings. When
// useRelative is true, --project-dir is passed as a relative path to
// regression-guard the runner.go filepath.Abs fix.
func runFixture(t *testing.T, fixturePath string, useRelative bool) {
	t.Helper()
	if kdoctorTestBin == "" {
		t.Skip("kdoctor binary not available")
	}
	detektBin := detectDetektBinary()
	if detektBin == "" {
		t.Skip("detekt-cli not found; install or set KDOCTOR_DETEKT_BIN")
	}
	root, ok := repoRoot()
	if !ok {
		t.Skip("could not locate repo root")
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read %s: %v", fixturePath, err)
	}
	var fix Fixture
	if err := json.Unmarshal(data, &fix); err != nil {
		t.Fatalf("parse %s: %v", fixturePath, err)
	}
	projectPath := fix.ProjectPath
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join(root, projectPath)
	}
	if _, err := os.Stat(projectPath); err != nil {
		t.Skipf("project %s missing: %v", projectPath, err)
	}
	if useRelative {
		projectPath = filepath.ToSlash(fix.ProjectPath) // relative to repo root
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	args := []string{
		"scan", "--json",
		"--type=kmp",
		"--prefer-standalone",
		"--detekt-bin=" + detektBin,
		"--project-dir=" + projectPath,
	}
	cmd := exec.CommandContext(ctx, kdoctorTestBin, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kdoctor scan exit=%v\n--- output ---\n%s\n--- end ---",
			err, truncateForLog(string(out), 4000))
	}
	var r report
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("parse kdoctor JSON: %v\n--- output (first 4000B) ---\n%s",
			err, truncateForLog(string(out), 4000))
	}
	if r.SchemaVersion != "3" {
		t.Errorf("%s: schemaVersion=%q != \"3\"", fix.ProjectPath, r.SchemaVersion)
	}
	if r.HealthScore < fix.MinScoreExpected || r.HealthScore > fix.MaxScoreExpected {
		t.Errorf("%s: score=%d not in [%d, %d]",
			fix.ProjectPath, r.HealthScore, fix.MinScoreExpected, fix.MaxScoreExpected)
	}
	found := map[string]bool{}
	for _, ff := range r.Findings {
		found[ff.ID] = true
	}
	for _, want := range fix.MustIncludeFinding {
		if !found[want] {
			keys := sortedKeys(found)
			t.Errorf("%s: required finding %q NOT detected\n      detected: %s",
				fix.ProjectPath, want, keys)
		}
	}
	t.Logf("%s: score=%d findings=%d required=%s",
		filepath.Base(fix.ProjectPath), r.HealthScore, len(r.Findings),
		requiredCheckmarks(found, fix.MustIncludeFinding))
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}

func sortedKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func requiredCheckmarks(m map[string]bool, want []string) string {
	if len(want) == 0 {
		return "[]"
	}
	out := make([]string, 0, len(want))
	for _, k := range want {
		if m[k] {
			out = append(out, k+"✓")
		} else {
			out = append(out, k+"✗")
		}
	}
	return strings.Join(out, ",")
}
