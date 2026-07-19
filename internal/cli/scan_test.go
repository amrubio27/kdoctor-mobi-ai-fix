package cli

import (
	"bytes"
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
	t.Setenv("ADKD_RULES_DIR", "")
	path, err := resolveRulesPath()
	if err == nil {
		t.Fatalf("expected error, got path %q", path)
	}
	if !strings.Contains(err.Error(), "metadata.json") {
		t.Fatal("error should reference metadata.json")
	}
}
