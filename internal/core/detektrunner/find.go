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
// Si no los encuentra, devuelve el primer *.sarif válido que aparezca
// en cualquier lugar del árbol del proyecto (fallback conservador).
//
// Cada candidato se valida con isValidSARIF(): lee las primeras 512B
// y exige que contenga "version":"2.1.0" en JSON. Eso descarta falsos
// positivos cuando el proyecto tiene .sarif legacy o de otras tools.
//
// Skip dirs: .git, node_modules, .gradle, .idea — carpetas ruidosas
// que pueden contener miles de archivos irrelevantes para SARIF.
//
// TODO (Fase 2 optimization): la walk recursiva es lineal y puede
// ser lenta en monorepos de 50K+ archivos. filepath.Glob no soporta
// `**` así que se necesita WalkDir. Optimizar con depth-limit (e.g.
// max 5 niveles por módulo) si hace falta.
//
// TODO (Fase 2 logging): en modo --verbose, loggear cuántos entries
// se inspeccionaron y los perm errors encontrados (hoy silenciados).
func findProducedSARIF(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return ""
	}

	var preferred, fallback string
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // TODO Fase 2: --verbose log
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
		if filepath.Ext(path) != ".sarif" {
			return nil
		}
		if !isValidSARIF(path) {
			return nil // no es SARIF 2.1.0 — descarta
		}
		normalized := filepath.ToSlash(path)
		if strings.Contains(normalized, "/build/reports/detekt/") {
			preferred = path
			return filepath.SkipAll
		}
		if fallback == "" {
			fallback = path
		}
		return nil
	})

	if preferred != "" {
		return preferred
	}
	return fallback
}

// isValidSARIF abre las primeras 512 bytes del archivo y valida que
// contenga el marcador JSON de SARIF 2.1.0 ("version":"2.1.0").
// Es un check defensivo contra ficheros renombrados a .sarif que no
// son SARIF válido.
func isValidSARIF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return false
	}
	head := string(buf[:n])
	return strings.Contains(head, `"version"`) && strings.Contains(head, `"2.1.0"`)
}

// isDetektSARIFPath — diagnostic helper (kept for tests). Indica si
// el path parece provenir de Detekt (i.e. está bajo build/reports/detekt/).
func isDetektSARIFPath(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/build/reports/detekt/")
}
