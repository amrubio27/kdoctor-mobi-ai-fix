package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAreSensible(t *testing.T) {
	c := Default()
	if c.ProjectType != "android" {
		t.Fatalf("projectType %q", c.ProjectType)
	}
	if c.Score.FailBelow <= 0 {
		t.Fatalf("failBelow %d", c.Score.FailBelow)
	}
	if c.AiFixer.Provider == "" {
		t.Fatal("provider empty")
	}
}

func TestLoadFromMissingFileReturnsDefault(t *testing.T) {
	c, err := Load(filepath.Join(os.TempDir(), "no-such-kdoctor-config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.ProjectType != "android" {
		t.Fatalf("default projectType lost: %q", c.ProjectType)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "kdoctor.config.yaml")
	if err := os.WriteFile(p, []byte(`
projectType: kmp
paths:
  kotlin:
    - "src/commonMain/**/*.kt"
score:
  failBelow: 90
aiFixer:
  provider: mobiai
  mode: auto
`), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.ProjectType != "kmp" {
		t.Fatalf("projectType %q", c.ProjectType)
	}
	if c.Score.FailBelow != 90 {
		t.Fatalf("failBelow %d", c.Score.FailBelow)
	}
	if c.AiFixer.Provider != "mobiai" {
		t.Fatalf("provider %q", c.AiFixer.Provider)
	}
	if c.AiFixer.Mode != "auto" {
		t.Fatalf("mode %q", c.AiFixer.Mode)
	}
	if len(c.Paths["kotlin"]) != 1 || c.Paths["kotlin"][0] == "" {
		t.Fatalf("paths.kotlin off")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	c := Default()
	c.ProjectType = "cmp"
	data, err := Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("marshal empty")
	}
	again, err := loadFromBytes(t, data)
	if err != nil {
		t.Fatal(err)
	}
	if again.ProjectType != "cmp" {
		t.Fatalf("projectType %q", again.ProjectType)
	}
}

// internal helper: parsea un []byte como Config roundtrip.
func loadFromBytes(t *testing.T, data []byte) (Config, error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(p, data, 0644); err != nil {
		return Config{}, err
	}
	return Load(p)
}
