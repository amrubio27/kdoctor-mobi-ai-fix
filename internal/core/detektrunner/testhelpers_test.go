package detektrunner

import (
	"os"
	"path/filepath"
)

// readFile / mkdirAll / writeFileString helpers usados en runner_test.go.
// Definidos aquí para evitar colisión con os.ReadFile y arrancar más rápido.

func readFile(p string) ([]byte, error) {
	return os.ReadFile(p)
}

func mkdirAll(p string) error {
	return os.MkdirAll(p, 0755)
}

func writeFileString(p, content string) error {
	if err := mkdirAll(filepath.Dir(p)); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0644)
}
