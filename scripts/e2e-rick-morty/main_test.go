package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSmokeRickMortyApp(t *testing.T) {
	project := os.Getenv("RICK_MORTY_APP")
	if project == "" {
		project = "D:/Programacion/RickMortyApp"
	}
	if _, err := os.Stat(project); err != nil {
		t.Skipf("RickMortyApp not found at %s (set RICK_MORTY_APP)", project)
	}

	root, ok := repoRoot()
	if !ok {
		t.Skip("could not locate repo root")
	}

	out := filepath.Join(os.TempDir(), "e2e-rick-morty-test")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", out, "./scripts/e2e-rick-morty")
	buildCmd.Dir = root
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build e2e runner: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(out) })

	run := exec.Command(out, "--smoke")
	run.Dir = root
	run.Env = append(os.Environ(), "KDOCTOR_DETEKT_BIN=D:/tools/detekt.cmd")
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("e2e --smoke failed: %v\n%s", err, string(output))
	}
	if len(output) == 0 {
		t.Fatal("expected non-empty output from e2e runner")
	}
	got := string(output)
	if !contains(got, "Smoke OK") {
		t.Fatalf("expected 'Smoke OK' in runner output, got:\n%s", got)
	}
}

func repoRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	wd, _ = filepath.Abs(wd)
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, true
		}
		wd = filepath.Dir(wd)
	}
	return "", false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSub(s, sub))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// avoid "fmt imported and not used" if we ever need it.
var _ = fmt.Sprintf
