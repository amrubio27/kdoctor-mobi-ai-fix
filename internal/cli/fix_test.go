package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// leakyKotlin trips the native sec-log-pii detector, so these tests need no
// detekt and no JVM.
const leakyKotlin = "package demo\n\n" +
	"fun login(password: String) {\n" +
	"    Log.d(\"auth\", \"password=\" + password)\n" +
	"}\n"

func writeProject(t *testing.T) (dir, srcPath string) {
	t.Helper()
	dir = t.TempDir()
	srcPath = filepath.Join(dir, "Leaky.kt")
	if err := os.WriteFile(srcPath, []byte(leakyKotlin), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, srcPath
}

// fixArgs keeps detekt out of the way so these stay fast and hermetic.
func fixArgs(dir string, extra ...string) []string {
	return append([]string{
		"--project-dir=" + dir,
		"--detekt-bin=" + filepath.Join(dir, "no-such-detekt"),
	}, extra...)
}

// TestFixNeverModifiesSource is the invariant that replaced a data-loss bug.
//
// `fix` used to patch files in place, and when its LLM provider failed it
// synthesised a "// Provider failed" patch. That comment is brace-balanced, so
// patchguard accepted it and --mode=auto wrote it over real code. kdoctor no
// longer edits source at all, which makes the property absolute rather than
// conditional on a provider behaving.
func TestFixNeverModifiesSource(t *testing.T) {
	dir, srcPath := writeProject(t)

	// Include the deprecated flags that used to trigger in-place edits.
	cmd := NewFixCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(fixArgs(dir, "--ai", "--mode=auto"))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fix failed: %v", err)
	}

	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != leakyKotlin {
		t.Fatalf("fix modified source.\nwant: %q\ngot:  %q", leakyKotlin, string(got))
	}
}

func TestFixWritesPlanIntoProjectDir(t *testing.T) {
	dir, _ := writeProject(t)

	cmd := NewFixCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(fixArgs(dir))
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// The report belongs next to the code it describes, not in the CWD -- a
	// stray fixes.md ended up committed at the repo root that way.
	data, err := os.ReadFile(filepath.Join(dir, "fixes.md"))
	if err != nil {
		t.Fatalf("expected fixes.md inside --project-dir: %v", err)
	}
	for _, want := range []string{"remediation plan", "sec-log-pii", "Replace lines"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("plan missing %q:\n%s", want, data)
		}
	}
}

func TestFixJSONPlanIsMachineReadable(t *testing.T) {
	dir, _ := writeProject(t)

	var buf bytes.Buffer
	cmd := NewFixCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(fixArgs(dir, "--json"))
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var plan struct {
		SchemaVersion string `json:"schemaVersion"`
		Instructions  string `json:"instructions"`
		Items         []struct {
			ID          string `json:"id"`
			File        string `json:"file"`
			Line        int    `json:"line"`
			ReplaceFrom int    `json:"replaceFrom"`
			ReplaceTo   int    `json:"replaceTo"`
			Context     string `json:"context"`
		} `json:"items"`
	}
	if err := json.Unmarshal(buf.Bytes(), &plan); err != nil {
		t.Fatalf("plan is not valid JSON: %v\n%s", err, buf.String())
	}
	if plan.SchemaVersion == "" || plan.Instructions == "" {
		t.Fatal("plan must carry a schema version and instructions")
	}
	if len(plan.Items) == 0 {
		t.Fatal("expected at least one item for a file with a PII log leak")
	}
	for _, it := range plan.Items {
		// Without a concrete line range the agent has to guess what to replace,
		// which is what made the old in-place patching unsafe.
		if it.ReplaceFrom < 1 || it.ReplaceTo < it.ReplaceFrom {
			t.Errorf("item %s has an unusable range %d..%d", it.ID, it.ReplaceFrom, it.ReplaceTo)
		}
		if it.Context == "" {
			t.Errorf("item %s carries no source context", it.ID)
		}
	}
}

func TestFixValidateDetectsUnbalancedKotlin(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "Good.kt")
	bad := filepath.Join(dir, "Bad.kt")
	if err := os.WriteFile(good, []byte("fun a() {\n    println(\"ok\")\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A patch that swallowed a closing brace: the exact damage --validate exists
	// to catch after an agent edits a file.
	if err := os.WriteFile(bad, []byte("fun a() {\n    println(\"oops\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewFixCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--validate", "--project-dir=" + dir, good})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("balanced file should pass: %v", err)
	}

	cmd = NewFixCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--validate", "--project-dir=" + dir, bad})
	if err := cmd.Execute(); err == nil {
		t.Fatal("unbalanced file should fail validation")
	}
}
