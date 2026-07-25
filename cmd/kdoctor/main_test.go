package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	if !strings.Contains(buf.String(), "0.2.0") {
		t.Fatalf("expected 0.2.0 in version output, got:\n%s", buf.String())
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
