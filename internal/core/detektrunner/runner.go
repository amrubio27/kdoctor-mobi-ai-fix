// Package detektrunner lanza Detekt como subproceso y captura su salida SARIF.
//
// Modo dual:
//   - Standalone: binario `detekt` instalado en PATH
//   - GradleWrapper: ./gradlew detekt con init-script generado por WriteInitScript
//
// Decide el modo con Detect(projectDir, preferStandalone) y respeta la
// elección del usuario vía flag --prefer-standalone en CLI.
package detektrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ExecutionMode string

const (
	ModeStandalone ExecutionMode = "standalone"
	ModeGradleWrap ExecutionMode = "gradlew"
)

type Options struct {
	ProjectDir     string
	SARIFOutput    string
	UseStandalone  bool
	StandalonePath string
	Stdout         io.Writer // opcional, spinners del CLI
}

func RunDetekt(ctx context.Context, opts Options) (string, error) {
	if opts.ProjectDir == "" {
		return "", fmt.Errorf("ProjectDir required")
	}
	if opts.SARIFOutput == "" {
		return "", fmt.Errorf("SARIFOutput required")
	}
	if opts.UseStandalone {
		return runStandalone(ctx, opts)
	}
	return runGradlew(ctx, opts)
}

// defaultMaxIssues fija un umbral alto para que detekt-cli salga con
// exit code 0 tras reportar findings. Detekt retorna exit 2 cuando hay
// issues; queremos esos findings en el SARIF, no fallar. Configurable
// en Fase 2 desde kdoctor.config.yaml (e.g. failAbove=N real).
const defaultMaxIssues = 99999

// defaultExcludes son patrones glob que detekt salta. Generados Android/KMP
// (build/, kspCaches/, .gradle/, Room migrations, etc.) crean ruido sin
// valor para kdoctor. Robusto a Fase 2 cuando el usuario pueda override.
var defaultExcludes = []string{
	"**/build/**",
	"**/.gradle/**",
	"**/kspCaches/**",
}

// probeDetektIsV2 checks whether the resolved detekt binary reports version 2.x.
func probeDetektIsV2(ctx context.Context, bin string) bool {
	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(bin), ".jar") {
		cmd = exec.CommandContext(ctx, "java", "-jar", bin, "--version")
	} else {
		cmd = exec.CommandContext(ctx, bin, "--version")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	verStr := strings.TrimSpace(string(out))
	return strings.HasPrefix(verStr, "2.") || strings.Contains(verStr, " 2.")
}

func runStandalone(ctx context.Context, opts Options) (string, error) {
	bin := opts.StandalonePath
	if bin == "" {
		bin = "detekt"
	}
	// Resuelve opts.ProjectDir a path absoluto.
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("abs path for projectDir: %w", err)
	}
	excludesCSV := strings.Join(defaultExcludes, ",")
	// Crítico: limpia cualquier SARIF stale de runs previos.
	_ = os.Remove(opts.SARIFOutput)

	isV2 := probeDetektIsV2(ctx, bin)

	args := []string{
		"--input", absProjectDir,
		"--report", "sarif:" + opts.SARIFOutput,
		"--excludes", excludesCSV,
	}
	if !isV2 {
		args = append(args, "--max-issues", fmt.Sprintf("%d", defaultMaxIssues))
	}

	// Si el proyecto provee su propio detekt.yml (o detekt.yaml) en la raíz,
	// lo pasamos con --config.
	var configFound bool
	for _, name := range []string{"detekt.yml", "detekt.yaml"} {
		candidate := filepath.Join(absProjectDir, name)
		if cfg, err := os.Stat(candidate); err == nil && !cfg.IsDir() {
			args = append(args, "--config", candidate)
			configFound = true
			break
		}
	}
	if !configFound {
		autoConfig := filepath.Join(os.TempDir(), "kdoctor-auto-detekt.yml")
		content := "naming:\n  active: true\n  FunctionNaming:\n    active: true\n    ignoreAnnotated:\n      - \"Composable\"\n"
		if err := os.WriteFile(autoConfig, []byte(content), 0644); err == nil {
			args = append(args, "--config", autoConfig)
		}
	}

	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(bin), ".jar") {
		if _, err := exec.LookPath("java"); err != nil {
			return "", fmt.Errorf("java runtime not found in PATH required to execute detekt jar: %s", bin)
		}
		javaArgs := append([]string{"-jar", bin}, args...)
		cmd = exec.CommandContext(ctx, "java", javaArgs...)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
	}
	cmd.Dir = absProjectDir
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		// Safety net: si detekt escribió un SARIF fresco pese a exit != 0
		// (e.g. detekt retorna exit 2 cuando hay findings), aceptamos el SARIF.
		if stat, statErr := os.Stat(opts.SARIFOutput); statErr == nil && stat.Size() > 0 {
			return opts.SARIFOutput, nil
		}
		return "", fmt.Errorf("detekt standalone: %w", err)
	}
	return opts.SARIFOutput, nil
}

func runGradlew(ctx context.Context, opts Options) (string, error) {
	// Mismo fix de path absoluto que runStandalone.
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("abs path for projectDir: %w", err)
	}
	gradlew := filepath.Join(absProjectDir, "gradlew")
	if _, err := os.Stat(gradlew); err != nil {
		return "", fmt.Errorf("gradlew no encontrado en %s: %w", absProjectDir, err)
	}
	initPath, err := WriteInitScript(absProjectDir)
	if err != nil {
		return "", fmt.Errorf("escribir init-script: %w", err)
	}
	// Note: defer cleanup. La init-script se regenera cada corrida.
	defer func() { _ = os.Remove(initPath) }()

	args := []string{"detekt", "--init-script", initPath}
	cmd := exec.CommandContext(ctx, gradlew, args...)
	cmd.Dir = absProjectDir
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("./gradlew detekt: %w", err)
	}
	// SARIF Gradle plugin escribe típicamente a build/reports/detekt/kdoctor.sarif.
	// En multi-modulo (`:app`, `:core`, ...) puede aparecer en
	// <ProjectDir>/<module>/build/... Por eso hacemos find recursivo.
	if produced := findProducedSARIF(absProjectDir); produced != "" {
		// Mover/copiar al target SARIFOutput para que el parser
		// tenga un único path de entrada. (En Fase 2 simplificamos.)
		if err := copyFile(produced, opts.SARIFOutput); err != nil {
			return opts.SARIFOutput, fmt.Errorf("copiar SARIF %s → %s: %w", produced, opts.SARIFOutput, err)
		}
		return opts.SARIFOutput, nil
	}
	return opts.SARIFOutput, fmt.Errorf("no se encontró SARIF en %s/build/reports/detekt/ (multi-mod OK, revisado)", absProjectDir)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
