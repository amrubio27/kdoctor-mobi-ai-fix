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

func runStandalone(ctx context.Context, opts Options) (string, error) {
	bin := opts.StandalonePath
	if bin == "" {
		bin = "detekt"
	}
	// Resuelve opts.ProjectDir a path absoluto. Sin esto, cuando el usuario
	// pasa --project-dir <ruta_relativa> (e.g. "examples/bad-project"),
	// cmd.Dir también es relativo y detekt interpreta --input relativo a
	// cmd.Dir — termina buscando <cwd>/<relative>/<relative> (no existe) y
	// falla con exit 1 sin escribir SARIF. Trap diagnosticado en Fase 2.
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("abs path for projectDir: %w", err)
	}
	excludesCSV := strings.Join(defaultExcludes, ",")
	// Crítico: limpia cualquier SARIF stale de runs previos. Sin esto,
	// el safety net acepta el archivo anterior aunque detekt haya fallado.
	_ = os.Remove(opts.SARIFOutput)
	args := []string{
		"--input", absProjectDir,
		"--report", "sarif:" + opts.SARIFOutput,
		"--max-issues", fmt.Sprintf("%d", defaultMaxIssues),
		"--excludes", excludesCSV,
	}
	// Si el proyecto provee su propio detekt.yml (o detekt.yaml) en la raíz,
	// lo pasamos con --config. Permite al usuario (o a los test fixtures)
	// habilitar reglas que detekt-cli desactiva por defecto (HardcodedPassword,
	// GlobalCoroutineUsage, UnusedImport, etc.).
	for _, name := range []string{"detekt.yml", "detekt.yaml"} {
		candidate := filepath.Join(absProjectDir, name)
		if cfg, err := os.Stat(candidate); err == nil && !cfg.IsDir() {
			args = append(args, "--config", candidate)
			break
		}
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = absProjectDir
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		// Safety net: si detekt escribió un SARIF fresco pese a exit != 0,
		// algunos runtimes retornan non-zero incluso con findings legítimos.
		// El SARIF es fresh porque acabamos de hacer Remove arriba.
		if _, statErr := os.Stat(opts.SARIFOutput); statErr == nil {
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
