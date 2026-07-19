package detektrunner

import (
	"os"
	"path/filepath"
)

// findProducedSARIF busca recursivamente el primer *.sarif bajo
// <projectDir>/build/. Devuelve "" si no hay ninguno.
// Acepta proyectos multi-módulo donde el SARIF aparece bajo
// <projectDir>/<module>/build/reports/detekt/*.sarif.
func findProducedSARIF(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return ""
	}
	root := filepath.Join(abs, "build", "reports", "detekt")
	if !fileExists(root) {
		// Walk profundo bajo build/ buscando el primer *.sarif
		var found string
		_ = filepath.WalkDir(filepath.Join(abs, "build"), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".sarif" {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(root, "*.sarif"))
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}
