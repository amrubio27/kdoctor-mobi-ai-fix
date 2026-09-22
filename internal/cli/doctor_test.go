package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runDoctor(t *testing.T, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	cmd := NewDoctorCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, buf.String())
	}
	return buf.String()
}

// TestDoctorReportsRuleCoverage covers the reason the command exists: telling
// the user how much of the catalog the next scan can evaluate. The previous
// implementation only listed whether some binaries were on PATH, which answered
// none of that.
func TestDoctorReportsRuleCoverage(t *testing.T) {
	dir := t.TempDir()
	out := runDoctor(t, "--project-dir="+dir)

	for _, want := range []string{"Rules", "live", "Next scan", "Java"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q:\n%s", want, out)
		}
	}
	// The counts must come from the catalog, not be invented.
	if !strings.Contains(out, "catalogued") {
		t.Errorf("doctor should report the catalog size:\n%s", out)
	}
}

// TestDoctorDetectsGradleWrapper regression-guards a check that could never
// pass: it used exec.LookPath("./gradlew"), which does not resolve relative
// paths, so the wrapper was reported missing even when it sat right there.
func TestDoctorDetectsGradleWrapper(t *testing.T) {
	dir := t.TempDir()
	// Write both names: the resolver prefers gradlew.bat on Windows and
	// gradlew elsewhere, and the test should not care which host it runs on.
	for _, name := range []string{"gradlew", "gradlew.bat"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out := runDoctor(t, "--project-dir="+dir)
	if !strings.Contains(out, "gradlew") {
		t.Errorf("doctor did not find the Gradle wrapper in %s:\n%s", dir, out)
	}
}

func TestDoctorWithoutGradleWrapperIsNotAnError(t *testing.T) {
	out := runDoctor(t, "--project-dir="+t.TempDir())
	// A KMP or Android project without a wrapper is unusual but legal, and a
	// missing wrapper is not a failure now that standalone is the default.
	if !strings.Contains(out, "not required") {
		t.Errorf("a missing wrapper should be reported as optional:\n%s", out)
	}
}
