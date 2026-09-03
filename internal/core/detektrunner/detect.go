package detektrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detect decide el modo de ejecución de Detekt:
//   - Si explicitBin != "" → standalone (máxima prioridad)
//   - Si preferStandalone y `detekt` está en PATH → standalone
//   - Si ./gradlew existe → gradlew
//   - Fallback final: standalone (que fallará con mensaje claro si no hay binario)
func Detect(projectDir string, preferStandalone bool, explicitBin string) ExecutionMode {
	if strings.TrimSpace(explicitBin) != "" {
		return ModeStandalone
	}
	if preferStandalone {
		if _, err := exec.LookPath("detekt"); err == nil {
			return ModeStandalone
		}
	}
	if _, err := os.Stat(filepath.Join(projectDir, "gradlew")); err == nil {
		return ModeGradleWrap
	}
	return ModeStandalone
}

// fileExists helper.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
