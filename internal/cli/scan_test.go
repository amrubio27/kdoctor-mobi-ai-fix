package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanCmdAdvertisesFlags(t *testing.T) {
	cmd := NewScanCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"--json", "--prefer-standalone", "--fail-below", "--type"} {
		if !strings.Contains(out, want) {
			t.Errorf("flag %s missing from help\n%s", want, out)
		}
	}
}

func TestScanCmdResolvesRulesPathErrorIfMissing(t *testing.T) {
	// Con un CWD de tempdir vacío y sin reglas debe fallar con mensaje claro.
	dir := t.TempDir()
	original, _ := saveWD()
	defer restoreWD(original)
	if err := chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KDOCTOR_RULES_DIR", "")
	path, err := resolveRulesPath()
	if err == nil {
		t.Fatalf("expected error, got path %q", path)
	}
	if !strings.Contains(err.Error(), "metadata.json") {
		t.Fatal("error should reference metadata.json")
	}
}

// TestScanQualityGateAppliesToEveryOutputFormat pins the contract behind a bug
// where the output-format switch `return`ed from each branch, so the quality
// gate (and the MobiAI export) were only ever reached in console mode. The
// practical effect was that `kdoctor scan --json --fail-below N` — the exact
// invocation used by CI, the MCP server and the Gradle plugin — always exited 0.
func TestScanQualityGateAppliesToEveryOutputFormat(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Leaky.kt")
	// Two native-rule violations; enough to push the score below 100.
	code := "package demo\n\n" +
		"fun login(password: String) {\n" +
		"    Log.d(\"auth\", \"password=\" + password)\n" +
		"}\n\n" +
		"class Repo {\n" +
		"    val scope = Dispatchers.IO\n" +
		"}\n"
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	formats := []struct {
		name string
		flag string
	}{
		{"console", ""},
		{"json", "--json"},
		{"sarif", "--sarif"},
		{"md", "--md"},
		{"html", "--html"},
	}

	for _, tc := range formats {
		t.Run(tc.name+"/below_threshold_fails", func(t *testing.T) {
			args := []string{
				"--project-dir=" + dir,
				"--fail-below=100",
				// Point at a binary that cannot exist so detekt fails fast and
				// the scan degrades to the native detectors.
				"--detekt-bin=" + filepath.Join(dir, "no-such-detekt"),
				"--out=" + filepath.Join(t.TempDir(), "report.out"),
			}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			cmd := NewScanCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(args)
			err := cmd.Execute()
			if !errors.Is(err, ErrFailBelow) {
				t.Fatalf("format %s: expected ErrFailBelow, got %v", tc.name, err)
			}
		})

		t.Run(tc.name+"/above_threshold_passes", func(t *testing.T) {
			args := []string{
				"--project-dir=" + dir,
				"--fail-below=1",
				"--detekt-bin=" + filepath.Join(dir, "no-such-detekt"),
				"--out=" + filepath.Join(t.TempDir(), "report.out"),
			}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			cmd := NewScanCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("format %s: expected success, got %v", tc.name, err)
			}
		})
	}
}

// TestScanMobiaiExportIsReachableWithJSON covers the other half of the same
// bug: the MobiAI JSONL dump sits after the format switch, so `--mobiai --json`
// — the flow documented in docs/integrations/mobiai.md — wrote nothing at all.
func TestScanMobiaiExportIsReachableWithJSON(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Leaky.kt")
	code := "package demo\n\nfun login(password: String) {\n" +
		"    Log.d(\"auth\", \"password=\" + password)\n}\n"
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewScanCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"--project-dir=" + dir,
		"--json",
		"--mobiai",
		"--detekt-bin=" + filepath.Join(dir, "no-such-detekt"),
		"--out=" + filepath.Join(t.TempDir(), "report.json"),
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mobiai", "graph", "findings.jsonl"))
	if err != nil {
		t.Fatalf("mobiai export not written with --json: %v", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		t.Fatal("mobiai findings.jsonl is empty")
	}
}

// TestQualityGateHonoursConfigFailBelow covers score.failBelow, which was
// parsed from kdoctor.config.yaml and then never read: a team could commit a
// threshold and CI would pass regardless.
//
// The subtlety is that config.Load() returns a default carrying failBelow: 80
// even when there is no file, so naively wiring it up would have started
// failing every project that never opted in.
func TestQualityGateHonoursConfigFailBelow(t *testing.T) {
	newProject := func(t *testing.T, configYAML string) string {
		t.Helper()
		dir := t.TempDir()
		code := "package demo\n\nfun login(password: String) {\n" +
			"    Log.d(\"auth\", \"password=\" + password)\n}\n"
		if err := os.WriteFile(filepath.Join(dir, "Leaky.kt"), []byte(code), 0o644); err != nil {
			t.Fatal(err)
		}
		if configYAML != "" {
			if err := os.WriteFile(filepath.Join(dir, "kdoctor.config.yaml"), []byte(configYAML), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	run := func(t *testing.T, dir string, extra ...string) error {
		t.Helper()
		args := append([]string{
			"--project-dir=" + dir,
			"--detekt-bin=" + filepath.Join(dir, "no-such-detekt"),
			"--out=" + filepath.Join(t.TempDir(), "r.json"),
			"--json",
		}, extra...)
		cmd := NewScanCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	t.Run("config threshold is enforced", func(t *testing.T) {
		dir := newProject(t, "projectType: android\nscore:\n  failBelow: 100\n")
		if err := run(t, dir); !errors.Is(err, ErrFailBelow) {
			t.Fatalf("config failBelow ignored: %v", err)
		}
	})

	t.Run("explicit flag overrides the config", func(t *testing.T) {
		dir := newProject(t, "projectType: android\nscore:\n  failBelow: 100\n")
		if err := run(t, dir, "--fail-below=1"); err != nil {
			t.Fatalf("--fail-below should win over the config file: %v", err)
		}
	})

	t.Run("config without a score section means no threshold", func(t *testing.T) {
		// The case the first version of this feature got wrong: config.Load()
		// fills failBelow with 80 even when the YAML never mentions it, so a
		// project using kdoctor.config.yaml purely for excludes would have
		// silently acquired a quality gate it never asked for.
		dir := newProject(t, "projectType: android\nexcludes:\n  - "+"`**/build/**`"+"\n")
		if err := run(t, dir); err != nil {
			t.Fatalf("a config without score.failBelow must not gate: %v", err)
		}
	})

	t.Run("no config means no threshold", func(t *testing.T) {
		// The regression guard: config.Load()'s default of 80 must not leak in.
		dir := newProject(t, "")
		if err := run(t, dir); err != nil {
			t.Fatalf("a project without kdoctor.config.yaml must not gain a threshold: %v", err)
		}
	})
}
