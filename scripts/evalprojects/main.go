// evalprojects itera examples/scoring-fixtures/*.json, corre `adkd scan --json`
// en cada projectPath asociado y valida que el HealthScore cae dentro del
// rango esperado.
//
// Uso: go run ./scripts/evalprojects <path-a-adkd-binary>
// Exit code 0 si todas las fixtures pasan; 1 en caso contrario.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adkd/adkd/internal/core/types"
)

type Fixture struct {
	ProjectPath        string   `json:"projectPath"`
	MinScoreExpected   int      `json:"minScoreExpected"`
	MaxScoreExpected   int      `json:"maxScoreExpected"`
	MustIncludeFinding []string `json:"mustIncludeFinding"`
}

type finding struct {
	ID string `json:"id"`
}

type report struct {
	SchemaVersion string    `json:"schemaVersion"`
	HealthScore   int       `json:"healthScore"`
	Findings      []finding `json:"findings"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: evalprojects <path-a-adkd-binary>")
		os.Exit(2)
	}
	adkdBin := os.Args[1]

	fixturesGlob := filepath.Join("examples", "scoring-fixtures", "*.json")
	files, err := filepath.Glob(fixturesGlob)
	if err != nil {
		fmt.Fprintln(os.Stderr, "glob:", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no se encontraron fixtures en", fixturesGlob)
		os.Exit(1)
	}

	fails := 0
	for _, f := range files {
		data, _ := os.ReadFile(f)
		var fix Fixture
		if err := json.Unmarshal(data, &fix); err != nil {
			fmt.Fprintln(os.Stderr, f, err)
			fails++
			continue
		}
		if fix.ProjectPath == "" || !filepathExists(fix.ProjectPath) {
			fmt.Fprintf(os.Stderr,
				"\u26a0  skip %s: projectPath %q no existe (probablemente pendiente de Fase 3)\n",
				f, fix.ProjectPath)
			continue
		}
		if err := evaluate(fix, adkdBin); err != nil {
			fmt.Fprintln(os.Stderr, "\u274c", err)
			fails++
			continue
		}
		fmt.Println("\u2705", fix.ProjectPath)
	}
	if fails > 0 {
		fmt.Fprintf(os.Stderr, "%d fixtures fallaron\n", fails)
		os.Exit(1)
	}
}

func evaluate(fixture Fixture, adkdBinary string) error {
	cmd := exec.Command(adkdBinary, "scan", "--json", "--type=android")
	cmd.Dir = fixture.ProjectPath
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("adkd scan en %s: %w", fixture.ProjectPath, err)
	}
	var r report
	if err := json.Unmarshal(out, &r); err != nil {
		return fmt.Errorf("parse %s: %w", fixture.ProjectPath, err)
	}
	if r.SchemaVersion != types.SchemaVersion {
		return fmt.Errorf("%s: schemaVersion %q no soportado (esperado %q)",
			fixture.ProjectPath, r.SchemaVersion, types.SchemaVersion)
	}
	if r.HealthScore < fixture.MinScoreExpected || r.HealthScore > fixture.MaxScoreExpected {
		return fmt.Errorf("%s: score %d fuera de [%d, %d]",
			fixture.ProjectPath, r.HealthScore,
			fixture.MinScoreExpected, fixture.MaxScoreExpected)
	}
	ids := map[string]bool{}
	for _, f := range r.Findings {
		ids[f.ID] = true
	}
	for _, want := range fixture.MustIncludeFinding {
		if !ids[want] {
			return fmt.Errorf("%s: falta finding esperado %q", fixture.ProjectPath, want)
		}
	}
	return nil
}

func filepathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
