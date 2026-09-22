package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRootVersionReportsBuildVersion checks the contract -- whatever is in
// the version variable reaches --version -- rather than a literal.
//
// It used to assert the string "0.6.0", which was the hardcoded default.
// That made the test pass precisely because releases could NOT stamp a
// version: it would have failed the moment -X main.version started working.
func TestRootVersionReportsBuildVersion(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })
	version = "9.9.9-test"

	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	if !strings.Contains(buf.String(), "9.9.9-test") {
		t.Fatalf("build version did not reach --version output:\n%s", buf.String())
	}
}

func TestRootHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --help: %v", err)
	}
	if !strings.Contains(buf.String(), "kdoctor") {
		t.Fatalf("expected kdoctor in help output, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "scan") {
		t.Fatalf("expected scan subcommand advertised in help, got:\n%s", buf.String())
	}
}
