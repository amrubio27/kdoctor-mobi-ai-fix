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
)

type ExecutionMode string

const (
	ModeStandalone ExecutionMode = "standalone"
	ModeGradleWrap ExecutionMode = "gradlew"
)

type Options struct {
	ProjectDir    string
	SARIFOutput   string
	UseStandalone bool
	StandalonePath string
	Stdout        io.Writer // opcional, spinners del CLI
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

func runStandalone(ctx context.Context, opts Options) (string, error) {
	bin := opts.StandalonePath
	if bin == "" {
		bin = "detekt"
	}
	args := []string{"--input", opts.ProjectDir, "--report", "sarif:" + opts.SARIFOutput}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = opts.ProjectDir
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("detekt standalone: %w", err)
	}
	return opts.SARIFOutput, nil
}

func runGradlew(ctx context.Context, opts Options) (string, error) {
	gradlew := filepath.Join(opts.ProjectDir, "gradlew")
	if _, err := os.Stat(gradlew); err != nil {
		return "", fmt.Errorf("gradlew no encontrado en %s: %w", opts.ProjectDir, err)
	}
	initPath, err := WriteInitScript(opts.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("escribir init-script: %w", err)
	}
	// Note: defer cleanup. La init-script se regenera cada corrida.
	defer func() { _ = os.Remove(initPath) }()

	args := []string{"detekt", "--init-script", initPath}
	cmd := exec.CommandContext(ctx, gradlew, args...)
	cmd.Dir = opts.ProjectDir
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("./gradlew detekt: %w", err)
	}
	// SARIF Gradle plugin escribe típicamente a build/reports/detekt/adkd.sarif.
	// En multi-modulo (`:app`, `:core`, ...) puede aparecer en
	// <ProjectDir>/<module>/build/... Por eso hacemos find recursivo.
	if produced := findProducedSARIF(opts.ProjectDir); produced != "" {
		// Mover/copiar al target SARIFOutput para que el parser
		// tenga un único path de entrada. (En Fase 2 simplificamos.)
		if err := copyFile(produced, opts.SARIFOutput); err != nil {
			return opts.SARIFOutput, fmt.Errorf("copiar SARIF %s → %s: %w", produced, opts.SARIFOutput, err)
		}
		return opts.SARIFOutput, nil
	}
	return opts.SARIFOutput, fmt.Errorf("no se encontró SARIF en %s/build/reports/detekt/ (multi-mod OK, revisado)", opts.ProjectDir)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
