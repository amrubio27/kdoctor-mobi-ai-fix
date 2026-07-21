package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

// fakeProvider lets us inject deterministic LLM responses.
type fakeProvider struct {
	response string
	err      error
}

func (f *fakeProvider) Fix(prompt string) (string, error) {
	return f.response, f.err
}

func TestApplyFix_AppliesValidPatch(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "App.kt")
	original := "package com.example\n\nfun main() {\n    val x = 1\n    println(\"hello\")\n    val y = 2\n}\n"
	if err := os.WriteFile(srcPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	// LLM returns a replacement for the whole shown window.
	patch := "```kotlin\n    val x = 1\n    println(\"hello world\")\n    val y = 2\n```"
	finding := types.Finding{File: srcPath, Line: 5, ID: "dummy"}

	status, err := applyFix(finding, original, patch, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != "applied" {
		t.Fatalf("expected status applied, got %q", status)
	}

	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "package com.example\n\nfun main() {\n    val x = 1\n    println(\"hello world\")\n    val y = 2\n}\n"
	if string(got) != want {
		t.Fatalf("want %q, got %q", want, string(got))
	}
}

func TestApplyFix_RollsBackWhenPatchGuardFails(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "App.kt")
	original := "package com.example\n\nfun main() {\n    val x = 1\n    println(\"hello\")\n    val y = 2\n}\n"
	if err := os.WriteFile(srcPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	// LLM returns a replacement for the whole shown window with unbalanced braces.
	patch := "```kotlin\n    val x = 1\n    println(\"hello\" {\n    val y = 2\n```"
	finding := types.Finding{File: srcPath, Line: 5, ID: "dummy"}

	status, err := applyFix(finding, original, patch, 1)
	if err == nil {
		t.Fatal("expected error for invalid patch")
	}
	if status != "failed" {
		t.Fatalf("expected status failed, got %q", status)
	}

	// File must be unchanged.
	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("file was modified despite validation failure; want %q, got %q", original, string(got))
	}
}

func TestApplyFix_ExtractErrorWhenNoCodeBlock(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "App.kt")
	original := "fun main() {}\n"
	if err := os.WriteFile(srcPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	// Response is empty, so extraction should fail.
	finding := types.Finding{File: srcPath, Line: 1, ID: "dummy"}
	status, err := applyFix(finding, original, "", 1)
	if err == nil {
		t.Fatal("expected error for empty patch")
	}
	if status != "failed" {
		t.Fatalf("expected status failed, got %q", status)
	}
}
