package detektrunner

import (
	"os"
	"os/exec"
)

// Detect decide el modo de ejecución de Detekt:
//   - Si preferStandalone y `detekt` está en PATH → standalone
//   - Si ./gradlew existe → gradlew
//   - Fallback final: standalone (que fallará con mensaje claro si no hay binario)
func Detect(projectDir string, preferStandalone bool) ExecutionMode {
	if preferStandalone {
		if _, err := exec.LookPath("detekt"); err == nil {
			return ModeStandalone
		}
	}
	if _, err := os.Stat(projectDir + "/gradlew"); err == nil {
		return ModeGradleWrap
	}
	return ModeStandalone
}

// fileExists helper.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
