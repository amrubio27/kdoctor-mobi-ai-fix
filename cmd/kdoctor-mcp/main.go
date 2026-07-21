// kdoctor-mcp is a stdio-based Model Context Protocol (MCP) server that
// exposes kdoctor's CLI commands as tools to AI coding agents.
//
// It speaks JSON-RPC 2.0 over stdin/stdout. Supported methods:
//   - initialize
//   - tools/list
//   - tools/call
//
// Tools:
//   - kdoctor_scan        : run a kdoctor scan on a project
//   - kdoctor_rules       : list the kdoctor rule catalog
//   - kdoctor_init        : bootstrap kdoctor in a project directory
//   - kdoctor_doctor      : diagnose the kdoctor environment
//   - kdoctor_fix_suggest : generate AI fix suggestions without applying them
//
// The server expects a `kdoctor` binary available in PATH or pointed to by
// the KDOCTOR_BIN environment variable.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const protocolVersion = "2024-11-05"

// JSON-RPC structures.

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newRPCError(code int, msg string, args ...any) *rpcError {
	return &rpcError{Code: code, Message: fmt.Sprintf(msg, args...)}
}

// Tool schemas.

type tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

var tools = []tool{
	{
		Name:        "kdoctor_scan",
		Description: "Run a kdoctor quality scan on a project directory and return a JSON report.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"projectDir":   map[string]any{"type": "string", "description": "Project directory to scan. Defaults to the current working directory."},
				"projectType":  map[string]any{"type": "string", "description": "Project type: android, kmp, cmp, compose, jvm, gradle, plain."},
				"format":       map[string]any{"type": "string", "description": "Output format: json or sarif. Defaults to json."},
				"detektBin":    map[string]any{"type": "string", "description": "Explicit path to the detekt binary."},
				"failBelow":    map[string]any{"type": "integer", "description": "Fail if health score is below this value."},
				"diffRef":      map[string]any{"type": "string", "description": "Only show findings added/modified since this git ref."},
				"baselinePath": map[string]any{"type": "string", "description": "Path to a baseline file to suppress known findings."},
			},
		},
	},
	{
		Name:        "kdoctor_rules",
		Description: "List the curated kdoctor rule catalog with status and severity.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "kdoctor_init",
		Description: "Bootstrap kdoctor configuration files in a project directory.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"projectDir":  map[string]any{"type": "string", "description": "Project directory. Defaults to the current working directory."},
				"projectType": map[string]any{"type": "string", "description": "Project type. If omitted, auto-detection is used."},
				"force":       map[string]any{"type": "boolean", "description": "Overwrite existing config files."},
			},
		},
	},
	{
		Name:        "kdoctor_doctor",
		Description: "Diagnose the kdoctor environment (Go, Detekt, Gradle, LLM providers).",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "kdoctor_fix_suggest",
		Description: "Generate AI-driven fix suggestions for a project without applying them.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"projectDir":   map[string]any{"type": "string", "description": "Project directory to analyze. Defaults to the current working directory."},
				"detektBin":    map[string]any{"type": "string", "description": "Explicit path to the detekt binary."},
				"contextLines": map[string]any{"type": "integer", "description": "Number of context lines around each finding. Defaults to 10."},
			},
		},
	},
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "kdoctor-mcp error: %v\n", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if strings.TrimSpace(line) != "" {
				processLine(w, line)
			}
			return nil
		}
		if err != nil {
			return err
		}
		if strings.TrimSpace(line) != "" {
			if err := processLine(w, line); err != nil {
				return err
			}
		}
	}
}

func processLine(w *bufio.Writer, line string) error {
	var req rpcRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return writeResponse(w, rpcResponse{
			JSONRPC: "2.0",
			Error:   newRPCError(-32700, "invalid JSON: %v", err),
		})
	}
	resp := handle(req)
	if req.ID != nil {
		resp.ID = req.ID
	}
	return writeResponse(w, resp)
}

func handle(req rpcRequest) rpcResponse {
	if req.JSONRPC != "2.0" {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32600, "invalid JSON-RPC version")}
	}

	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			Result: map[string]any{
				"protocolVersion": protocolVersion,
				"serverInfo": map[string]any{
					"name":    "kdoctor-mcp",
					"version": "1.0.0",
				},
				"capabilities": map[string]any{},
			},
		}
	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", Result: map[string]any{"tools": tools}}
	case "tools/call":
		return handleToolCall(req.Params)
	default:
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32601, "method not found: %s", req.Method)}
	}
}

func handleToolCall(params json.RawMessage) rpcResponse {
	var payload struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &payload); err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "invalid params: %v", err)}
	}

	bin, err := findKdoctorBin()
	if err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32000, "kdoctor binary not found: %v", err)}
	}

	switch payload.Name {
	case "kdoctor_scan":
		return runScan(bin, payload.Arguments)
	case "kdoctor_rules":
		return runSimple(bin, "rules")
	case "kdoctor_init":
		return runInit(bin, payload.Arguments)
	case "kdoctor_doctor":
		return runSimple(bin, "doctor")
	case "kdoctor_fix_suggest":
		return runFixSuggest(bin, payload.Arguments)
	default:
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "unknown tool: %s", payload.Name)}
	}
}

// Tool implementations.

func runScan(bin string, args json.RawMessage) rpcResponse {
	var p struct {
		ProjectDir   string `json:"projectDir"`
		ProjectType  string `json:"projectType"`
		Format       string `json:"format"`
		DetektBin    string `json:"detektBin"`
		FailBelow    int    `json:"failBelow"`
		DiffRef      string `json:"diffRef"`
		BaselinePath string `json:"baselinePath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "invalid arguments: %v", err)}
	}

	argv := []string{"scan"}
	if p.Format == "sarif" {
		argv = append(argv, "--sarif")
	} else {
		argv = append(argv, "--json")
	}
	if p.ProjectDir != "" {
		argv = append(argv, "--project-dir", p.ProjectDir)
	}
	if p.ProjectType != "" {
		argv = append(argv, "--type", p.ProjectType)
	}
	if p.DetektBin != "" {
		argv = append(argv, "--detekt-bin", p.DetektBin)
	}
	if p.FailBelow > 0 {
		argv = append(argv, "--fail-below", fmt.Sprintf("%d", p.FailBelow))
	}
	if p.DiffRef != "" {
		argv = append(argv, "--diff", p.DiffRef)
	}
	if p.BaselinePath != "" {
		argv = append(argv, "--baseline", p.BaselinePath)
	}

	return execTool(bin, argv, "")
}

func runInit(bin string, args json.RawMessage) rpcResponse {
	var p struct {
		ProjectDir  string `json:"projectDir"`
		ProjectType string `json:"projectType"`
		Force       bool   `json:"force"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "invalid arguments: %v", err)}
	}

	if p.ProjectDir != "" {
		info, err := os.Stat(p.ProjectDir)
		if err != nil {
			if err := os.MkdirAll(p.ProjectDir, 0755); err != nil {
				return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "cannot create projectDir: %v", err)}
			}
		} else if !info.IsDir() {
			return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "projectDir is not a directory: %s", p.ProjectDir)}
		}
	}

	argv := []string{"init"}
	if p.ProjectType != "" {
		argv = append(argv, "--type", p.ProjectType)
	}
	if p.Force {
		argv = append(argv, "--force")
	}
	return execTool(bin, argv, p.ProjectDir)
}

func runFixSuggest(bin string, args json.RawMessage) rpcResponse {
	var p struct {
		ProjectDir   string `json:"projectDir"`
		DetektBin    string `json:"detektBin"`
		ContextLines int    `json:"contextLines"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: newRPCError(-32602, "invalid arguments: %v", err)}
	}

	argv := []string{"fix", "--ai", "--mode", "suggest"}
	if p.ProjectDir != "" {
		argv = append(argv, "--project-dir", p.ProjectDir)
	}
	if p.DetektBin != "" {
		argv = append(argv, "--detekt-bin", p.DetektBin)
	}
	if p.ContextLines > 0 {
		argv = append(argv, "--context-lines", fmt.Sprintf("%d", p.ContextLines))
	}
	return execTool(bin, argv, "")
}

func runSimple(bin string, subcmd string) rpcResponse {
	return execTool(bin, []string{subcmd}, "")
}

// execTool runs the kdoctor binary with the given arguments and returns an
// MCP content result. stdout and stderr are captured and returned in the
// result so the calling agent can see both normal output and warnings.
func execTool(bin string, argv []string, workDir string) rpcResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Env = os.Environ()
	if workDir != "" {
		cmd.Dir = workDir
	}

	outBytes, err := cmd.CombinedOutput()
	out := string(outBytes)
	if err != nil {
		return rpcResponse{
			JSONRPC: "2.0",
			Result: map[string]any{
				"content": []map[string]string{
					{"type": "text", "text": out},
				},
				"isError": true,
				"error":   fmt.Sprintf("%v", err),
			},
		}
	}
	return rpcResponse{
		JSONRPC: "2.0",
		Result: map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": out},
			},
			"isError": false,
		},
	}
}

// findKdoctorBin resolves the kdoctor binary path. Priority:
//  1. KDOCTOR_BIN environment variable
//  2. "kdoctor" executable in PATH
func findKdoctorBin() (string, error) {
	if env := os.Getenv("KDOCTOR_BIN"); env != "" {
		info, err := os.Stat(env)
		if err != nil {
			return "", fmt.Errorf("KDOCTOR_BIN=%s not found", env)
		}
		if info.IsDir() {
			return "", fmt.Errorf("KDOCTOR_BIN=%s is a directory, not an executable", env)
		}
		return env, nil
	}
	p, err := exec.LookPath("kdoctor")
	if err != nil {
		return "", fmt.Errorf("kdoctor not found in PATH and KDOCTOR_BIN unset")
	}
	return p, nil
}

func writeResponse(w *bufio.Writer, resp rpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if _, err := w.WriteString("\n"); err != nil {
		return err
	}
	return w.Flush()
}
