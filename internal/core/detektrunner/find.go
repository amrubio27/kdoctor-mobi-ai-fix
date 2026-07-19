package detektrunner

import (
	"os"
	"path/filepath"
	"strings"
)

// findProducedSARIF busca recursivamente el primer *.sarif bajo
// <projectDir>/. Acepta proyectos multi-módulo donde el SARIF vive
// bajo <projectDir>/<module>/build/reports/detekt/*.sarif.
//
// Prioriza paths bajo build/reports/detekt/ (los que Detekt produce).
// Si no los encuentra, devuelve el primer *.sarif que aparezca en
// cualquier lugar del árbol del proyecto (fallback conservador).
//
// Skip dirs: .git, node_modules, .gradle, .idea — carpetas ruidosas
// que pueden contener miles de archivos irrelevantes para SARIF.
func findProducedSARIF(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return ""
	}

	var preferred, fallback string
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != abs {
				name := filepath.Base(path)
				if name == ".git" || name == "node_modules" || name == ".gradle" || name == ".idea" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) == ".sarif" {
			normalized := filepath.ToSlash(path)
			if strings.Contains(normalized, "/build/reports/detekt/") {
				preferred = path
				return filepath.SkipAll
			}
			if fallback == "" {
				fallback = path
			}
		}
		return nil
	})

	if preferred != "" {
		return preferred
	}
	return fallback
}
