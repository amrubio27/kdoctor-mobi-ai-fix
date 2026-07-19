package detektrunner

import (
	"path/filepath"
	"testing"
)

func TestIsValidSARIFAccepts21(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.sarif")
	if err := writeFileString(path, `{"version":"2.1.0","runs":[]}`); err != nil {
		t.Fatal(err)
	}
	if !isValidSARIF(path) {
		t.Fatal("valid SARIF 2.1.0 debe aceptar")
	}
}

func TestIsValidSARIFRejectsOldVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.sarif")
	if err := writeFileString(path, `{"version":"2.0.0","runs":[]}`); err != nil {
		t.Fatal(err)
	}
	if isValidSARIF(path) {
		t.Fatal("SARIF 2.0.0 debe rechazar (no es 2.1.0)")
	}
}

func TestIsValidSARIFRejectsRandomFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.sarif")
	if err := writeFileString(path, `{"some":"other tool's data"}`); err != nil {
		t.Fatal(err)
	}
	if isValidSARIF(path) {
		t.Fatal("fichero renombrado a .sarif sin version 2.1.0 debe rechazar")
	}
}

func TestIsValidSARIFRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.sarif")
	if err := writeFileString(path, ""); err != nil {
		t.Fatal(err)
	}
	if isValidSARIF(path) {
		t.Fatal("fichero vacío debe rechazar")
	}
}

func TestIsValidSARIFRejectsMissingFile(t *testing.T) {
	if isValidSARIF(filepath.Join(t.TempDir(), "nope.sarif")) {
		t.Fatal("path inexistente debe rechazar (sin panic)")
	}
}

func TestIsDetektSARIFPath(t *testing.T) {
	cases := map[string]bool{
		"/Users/foo/proj/build/reports/detekt/kdoctor.sarif":     true,
		"/Users/foo/proj/app/build/reports/detekt/kdoctor.sarif": true,
		"/Users/foo/proj/somewhere/random/file.sarif":         false,
		"/Users/foo/proj/build/something-else/detekt/out.sarif": false,
	}
	for input, want := range cases {
		if got := isDetektSARIFPath(input); got != want {
			t.Errorf("isDetektSARIFPath(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestFindProducedSARIFSkipsNonSARIFDotSarif(t *testing.T) {
	dir := t.TempDir()
	multi := filepath.Join(dir, "app", "build", "reports", "detekt")
	if err := mkdirAll(multi); err != nil {
		t.Fatal(err)
	}
	// ficheros renombrados a .sarif pero no son SARIF 2.1.0
	if err := writeFileString(filepath.Join(multi, "fake.sarif"), `{"some":"other"}`); err != nil {
		t.Fatal(err)
	}
	if err := writeFileString(filepath.Join(multi, "real.sarif"), `{"version":"2.1.0","runs":[]}`); err != nil {
		t.Fatal(err)
	}

	got := findProducedSARIF(dir)
	want := filepath.Join(multi, "real.sarif")
	if got != want {
		t.Fatalf("expected detector SARIF real %q, got %q (debe ignorar fake.sarif)", want, got)
	}
}
