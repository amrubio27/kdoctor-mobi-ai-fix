package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestHandleInitialize verifies the initialize method returns server info.
func TestHandleInitialize(t *testing.T) {
	req := rpcRequest{JSONRPC: "2.0", ID: []byte(`1`), Method: "initialize"}
	resp := handle(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	if m["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion: want %q, got %q", protocolVersion, m["protocolVersion"])
	}
	info, ok := m["serverInfo"].(map[string]any)
	if !ok || info["name"] != "kdoctor-mcp" {
		t.Errorf("unexpected serverInfo: %v", m["serverInfo"])
	}
}

// TestHandleToolsList verifies the tools/list method returns the expected tools.
func TestHandleToolsList(t *testing.T) {
	req := rpcRequest{JSONRPC: "2.0", ID: []byte(`1`), Method: "tools/list"}
	resp := handle(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	list, ok := m["tools"].([]tool)
	if !ok || len(list) == 0 {
		t.Fatalf("expected non-empty tools list, got %v", m["tools"])
	}
	want := []string{"kdoctor_scan", "kdoctor_rules", "kdoctor_init", "kdoctor_doctor", "kdoctor_fix_suggest"}
	if len(list) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(list))
	}
	for i, w := range want {
		if list[i].Name != w {
			t.Errorf("tool[%d]: want %q, got %q", i, w, list[i].Name)
		}
	}
}

// TestHandleMethodNotFound verifies unknown methods return an error.
func TestHandleMethodNotFound(t *testing.T) {
	req := rpcRequest{JSONRPC: "2.0", ID: []byte(`1`), Method: "unknown/method"}
	resp := handle(req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code: want -32601, got %d", resp.Error.Code)
	}
}

// TestRunTransportsJSONRPC verifies the stdio transport reads/wields valid JSON-RPC messages.
func TestRunTransportsJSONRPC(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	if err := run(in, &out); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if string(resp.ID) != "1" {
		t.Errorf("id: want 1, got %s", resp.ID)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

// TestExecToolWithFakeKdoctor verifies execTool returns the fake binary output.
func TestExecToolWithFakeKdoctor(t *testing.T) {
	bin := fakeKdoctorBinary(t, "scan")
	resp := execTool(bin, []string{"scan", "--json"}, "")
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	content, ok := result["content"].([]map[string]string)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content, got %v", result["content"])
	}
	if !strings.Contains(content[0]["text"], "FAKE_KDOCTOR_OK") {
		t.Errorf("unexpected output: %s", content[0]["text"])
	}
}

// TestFindKdoctorBinEnv verifies KDOCTOR_BIN is respected.
func TestFindKdoctorBinEnv(t *testing.T) {
	bin := fakeKdoctorBinary(t, "scan")
	t.Setenv("KDOCTOR_BIN", bin)
	found, err := findKdoctorBin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != bin {
		t.Errorf("want %q, got %q", bin, found)
	}
}

// TestFindKdoctorBinMissing verifies a clear error when the binary is unavailable.
func TestFindKdoctorBinMissing(t *testing.T) {
	t.Setenv("KDOCTOR_BIN", "")
	pathEnv := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() {
		os.Setenv("PATH", pathEnv)
	}()
	_, err := findKdoctorBin()
	if err == nil {
		t.Fatal("expected error when binary is missing")
	}
	if !strings.Contains(err.Error(), "kdoctor not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestFindKdoctorBinRejectsDirectory verifies KDOCTOR_BIN pointing to a directory fails.
func TestFindKdoctorBinRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDOCTOR_BIN", dir)
	_, err := findKdoctorBin()
	if err == nil {
		t.Fatal("expected error when KDOCTOR_BIN is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestRunInitCreatesMissingDir verifies the init tool creates a missing projectDir.
func TestRunInitCreatesMissingDir(t *testing.T) {
	bin := fakeKdoctorBinary(t, "init")
	dir := filepath.Join(t.TempDir(), "missing", "nested")
	payload, err := json.Marshal(map[string]string{"projectDir": dir})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resp := runInit(bin, payload)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("projectDir was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("projectDir is not a directory: %s", dir)
	}
}

// fakeKdoctorBinary writes a small executable that echoes a known marker.
func fakeKdoctorBinary(t *testing.T, _ string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "kdoctor")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	script := `#!/bin/sh
printf '%s' 'FAKE_KDOCTOR_OK'
`
	if runtime.GOOS == "windows" {
		script = `@echo off
echo FAKE_KDOCTOR_OK`
		bin = strings.TrimSuffix(bin, ".exe") + ".bat"
	}

	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return bin
}
