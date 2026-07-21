// e2e-rick-morty runs kdoctor against a real Android/KMP project
// (default: D:/Programacion/RickMortyApp) and validates the output.
//
// Usage:
//
//	go run ./scripts/e2e-rick-morty --smoke
//
// Environment variables:
//
//	RICK_MORTY_APP      path to the real project (default D:/Programacion/RickMortyApp)
//	KDOCTOR_DETEKT_BIN  explicit detekt binary path (default D:/tools/detekt.cmd)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adkd/adkd/internal/core/types"
)

type report struct {
	SchemaVersion string `json:"schemaVersion"`
	HealthScore   int    `json:"healthScore"`
	Summary       struct {
		Total int `json:"total"`
	} `json:"summary"`
	Findings []struct {
		ID string `json:"id"`
	} `json:"findings"`
}

func main() {
	var smoke bool
	flag.BoolVar(&smoke, "smoke", false, "only verify that the scan runs and returns a valid report")
	flag.Parse()

	project := os.Getenv("RICK_MORTY_APP")
	if project == "" {
		project = "D:/Programacion/RickMortyApp"
	}
	detektBin := os.Getenv("KDOCTOR_DETEKT_BIN")
	if detektBin == "" {
		detektBin = "D:/tools/detekt.cmd"
	}

	if _, err := os.Stat(project); err != nil {
		fmt.Fprintf(os.Stderr, "Project not found at %s (set RICK_MORTY_APP)\n", project)
		os.Exit(2)
	}

	kdoctorBin := findKdoctor()
	if kdoctorBin == "" {
		fmt.Fprintln(os.Stderr, "Could not find kdoctor binary; run `go build -o kdoctor ./cmd/kdoctor` first")
		os.Exit(2)
	}

	fmt.Printf("Running e2e scan against %s (smoke=%v)\n", project, smoke)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, kdoctorBin,
		"scan", "--json",
		"--type=kmp",
		"--prefer-standalone",
		"--detekt-bin="+detektBin,
		"--project-dir="+project,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kdoctor scan failed: %v\n--- output ---\n%s\n", err, truncate(string(out), 4000))
		os.Exit(1)
	}

	var r report
	if err := json.Unmarshal(out, &r); err != nil {
		fmt.Fprintf(os.Stderr, "parse report: %v\n--- output (first 4000B) ---\n%s\n", err, truncate(string(out), 4000))
		os.Exit(1)
	}

	if r.SchemaVersion != types.SchemaVersion {
		fmt.Fprintf(os.Stderr, "schema version mismatch: got %q want %q\n", r.SchemaVersion, types.SchemaVersion)
		os.Exit(1)
	}

	if smoke {
		fmt.Printf("Smoke OK: score=%d findings=%d\n", r.HealthScore, r.Summary.Total)
		os.Exit(0)
	}

	if r.HealthScore < 0 || r.HealthScore > 100 {
		fmt.Fprintf(os.Stderr, "health score out of range: %d\n", r.HealthScore)
		os.Exit(1)
	}
	if r.Summary.Total == 0 {
		fmt.Fprintln(os.Stderr, "expected at least one finding in a real project")
		os.Exit(1)
	}

	foundArchGod := false
	for _, f := range r.Findings {
		if f.ID == "arch-god-class" {
			foundArchGod = true
			break
		}
	}
	if !foundArchGod {
		fmt.Fprintln(os.Stderr, "expected arch-god-class to be present in RickMortyApp")
		os.Exit(1)
	}

	fmt.Printf("E2E OK: score=%d findings=%d\n", r.HealthScore, r.Summary.Total)
}

func findKdoctor() string {
	if v := os.Getenv("KDOCTOR_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	wd, err := os.Getwd()
	if err == nil {
		for _, name := range []string{"kdoctor.exe", "kdoctor"} {
			c := filepath.Join(wd, name)
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	if runtime.GOOS == "windows" {
		return ""
	}
	return "kdoctor"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}
