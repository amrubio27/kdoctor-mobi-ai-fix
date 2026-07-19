package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveRulesPath encuentra rules/metadata.json siguiendo este orden:
//  1. Env var KDOCTOR_RULES_DIR (override explícito)
//  2. Junto al binario: <exe>/rules/metadata.json, <exe>/../rules/, <exe>/../../rules/
//  3. CWD: ./rules/metadata.json (modo desarrollo con `go run`)
//
// Devuelve error claro si no encuentra nada — sin fallback silencioso.
func resolveRulesPath() (string, error) {
	if p := os.Getenv("KDOCTOR_RULES_DIR"); p != "" {
		candidate := filepath.Join(p, "metadata.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "rules", "metadata.json"),
			filepath.Join(dir, "..", "rules", "metadata.json"),
			filepath.Join(dir, "..", "..", "rules", "metadata.json"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
	}
	if _, err := os.Stat("rules/metadata.json"); err == nil {
		return "rules/metadata.json", nil
	}
	return "", fmt.Errorf("rules/metadata.json no encontrado; usa `go run ./scripts/genschema` para regenerarlo o compila desde la raíz del repo")
}
