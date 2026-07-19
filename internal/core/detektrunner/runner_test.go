package detektrunner

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectReturnsKnownMode(t *testing.T) {
	tmp := t.TempDir()
	mode := Detect(tmp, false)
	if mode != ModeStandalone && mode != ModeGradleWrap {
		t.Fatalf("unexpected mode %q", mode)
	}
}

func TestWriteInitScriptWritesFile(t *testing.T) {
	tmp := t.TempDir()
	path, err := WriteInitScript(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, initScriptName) {
		t.Fatalf("unexpected path %q", path)
	}
	data, _ := readFile(path)
	if !strings.Contains(string(data), "allprojects") {
		t.Fatal("template should contain allprojects")
	}
	if !strings.Contains(string(data), "sarif") {
		t.Fatal("template should configure sarif")
	}
}

func TestFindProducedSARIFRecursive(t *testing.T) {
	tmp := t.TempDir()
	// Construimos un layout multi-mod tipo app/build/reports/detekt/X.sarif
	multi := filepath.Join(tmp, "app", "build", "reports", "detekt")
	if err := mkdirAll(multi); err != nil {
		t.Fatal(err)
	}
	if err := writeFileString(filepath.Join(multi, "kdoctor.sarif"), `{"version":"2.1.0","runs":[]}`); err != nil {
		t.Fatal(err)
	}

	got := findProducedSARIF(tmp)
	want := filepath.Join(multi, "kdoctor.sarif")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFindProducedSARIFReturnsEmptyOnMissing(t *testing.T) {
	tmp := t.TempDir()
	if got := findProducedSARIF(tmp); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
