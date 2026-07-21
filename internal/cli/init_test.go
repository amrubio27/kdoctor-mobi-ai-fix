package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProjectType_CMP(t *testing.T) {
	tmp := t.TempDir()
	createDir(t, filepath.Join(tmp, "composeApp"))
	if got := detectProjectType(tmp); got != "cmp" {
		t.Fatalf("expected cmp, got %s", got)
	}
}

func TestDetectProjectType_KMP(t *testing.T) {
	tmp := t.TempDir()
	createDir(t, filepath.Join(tmp, "commonMain"))
	if got := detectProjectType(tmp); got != "kmp" {
		t.Fatalf("expected kmp, got %s", got)
	}
}

func TestDetectProjectType_Android(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "build.gradle.kts"), "plugins { id(\"com.android.application\") }")
	if got := detectProjectType(tmp); got != "android" {
		t.Fatalf("expected android, got %s", got)
	}
}

func TestDetectProjectType_JVM(t *testing.T) {
	tmp := t.TempDir()
	createDir(t, filepath.Join(tmp, "src", "main", "kotlin"))
	writeFile(t, filepath.Join(tmp, "src", "main", "kotlin", "App.kt"), "package app\n")
	if got := detectProjectType(tmp); got != "jvm" {
		t.Fatalf("expected jvm, got %s", got)
	}
}

func TestDetectProjectType_Gradle(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "build.gradle.kts"), "plugins { java }")
	if got := detectProjectType(tmp); got != "gradle" {
		t.Fatalf("expected gradle, got %s", got)
	}
}

func TestDetectProjectType_Plain(t *testing.T) {
	tmp := t.TempDir()
	if got := detectProjectType(tmp); got != "plain" {
		t.Fatalf("expected plain, got %s", got)
	}
}

func TestRunInitCreatesConfigAndDetekt(t *testing.T) {
	tmp := t.TempDir()
	setupInitTest(t, tmp)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--type", "android"})
	cmd.SetOut(os.Stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	assertFileExists(t, filepath.Join(tmp, "kdoctor.config.yaml"))
	assertFileExists(t, filepath.Join(tmp, "detekt.yml"))

	data, err := os.ReadFile(filepath.Join(tmp, "kdoctor.config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "projectType: android") {
		t.Fatalf("config does not contain projectType android")
	}
}

func TestRunInitRefusesOverwrite(t *testing.T) {
	tmp := t.TempDir()
	setupInitTest(t, tmp)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--type", "android"})
	cmd.SetOut(os.Stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	cmd2 := NewInitCmd()
	cmd2.SetArgs([]string{"--type", "android"})
	cmd2.SetOut(os.Stdout)
	if err := cmd2.Execute(); err == nil {
		t.Fatalf("second init without --force should fail")
	}
}

func TestRunInitForceOverwrites(t *testing.T) {
	tmp := t.TempDir()
	setupInitTest(t, tmp)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--type", "android"})
	cmd.SetOut(os.Stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	cmd2 := NewInitCmd()
	cmd2.SetArgs([]string{"--type", "android", "--force"})
	cmd2.SetOut(os.Stdout)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second init with --force failed: %v", err)
	}
}

func TestRunInitUpdatesGitignore(t *testing.T) {
	tmp := t.TempDir()
	setupInitTest(t, tmp)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--type", "plain"})
	cmd.SetOut(os.Stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	content := string(data)
	for _, want := range []string{"kdoctor.exe", ".kdoctor/", "report.json"} {
		if !strings.Contains(content, want) {
			t.Fatalf(".gitignore missing %q", want)
		}
	}
}

func TestUpdateGitignoreAppendsOnlyOnce(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".gitignore")
	updated, err := updateGitignore(path, false)
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	if !updated {
		t.Fatalf("expected first update to modify .gitignore")
	}

	updated, err = updateGitignore(path, false)
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if updated {
		t.Fatalf("second update should be a no-op")
	}
}

func createDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setupInitTest(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
