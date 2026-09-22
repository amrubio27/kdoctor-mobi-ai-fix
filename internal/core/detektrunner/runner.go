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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/toolchain"
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
	// JavaPath pins the JVM used to run a detekt .jar. Empty means "resolve
	// one now". Never fall back to the bare name "java": the first JVM on
	// PATH is often a feature release detekt cannot load.
	JavaPath string
	// NoDownload and Progress are forwarded to plugin provisioning.
	NoDownload bool
	Progress   io.Writer
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
func probeDetektIsV2(ctx context.Context, bin, javaPath string) bool {
	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(bin), ".jar") {
		if javaPath == "" {
			return false
		}
		cmd = exec.CommandContext(ctx, javaPath, "-jar", bin, "--version")
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

	// Resolve the JVM once, up front, so the probe below and the run itself
	// agree on which java they are talking about.
	javaPath := opts.JavaPath
	// Resolve a JVM even for launcher scripts: we cannot pass -jar to them,
	// but we can hand them a JAVA_HOME they will honour. Failure is only
	// fatal for an actual .jar.
	if javaPath == "" {
		j, jerr := toolchain.ResolveJava()
		if jerr != nil && needsJava(bin) {
			return "", fmt.Errorf("detekt needs a Java runtime: %w", jerr)
		}
		if jerr == nil {
			javaPath = j.Path
		}
	}

	isV2 := probeDetektIsV2(ctx, bin, javaPath)

	baseArgs := []string{
		"--input", absProjectDir,
		"--report", "sarif:" + opts.SARIFOutput,
		"--excludes", excludesCSV,
		// Without this, --config REPLACES detekt default ruleset instead of
		// layering on top of it, so only the handful of rules named in the
		// config file ever fire. That is why a scan with a minimal detekt.yml
		// reported a single finding across a 16 KLOC project.
		"--build-upon-default-config",
	}
	if !isV2 {
		baseArgs = append(baseArgs, "--max-issues", fmt.Sprintf("%d", defaultMaxIssues))
	}

	// A project that already configures detekt should have its rules honoured,
	// so its detekt.yml wins. config/detekt/ is the layout the Gradle plugin
	// documents, so look there too.
	// The Compose rules are the ones that differentiate kdoctor from plain
	// detekt, so load the plugin that implements them. Losing it degrades
	// coverage; it never fails the scan.
	if plugins, perr := EnsureComposePlugins(ctx, ProvisionOptions{
		NoDownload: opts.NoDownload,
		Progress:   opts.Progress,
	}); perr == nil && len(plugins) > 0 {
		baseArgs = append(baseArgs, "--plugins", strings.Join(plugins, ","))
	}

	projectConfig := findProjectDetektConfig(absProjectDir)
	autoConfig := writeAutoConfig()

	runOnce := func(configPath string) error {
		args := append([]string{}, baseArgs...)
		if configPath != "" {
			args = append(args, "--config", configPath)
		}
		_ = os.Remove(opts.SARIFOutput)

		var cmd *exec.Cmd
		if needsJava(bin) {
			if javaPath == "" {
				return fmt.Errorf("%w: required to execute detekt jar %s", toolchain.ErrNoJava, bin)
			}
			cmd = exec.CommandContext(ctx, javaPath, append([]string{"-jar", bin}, args...)...)
		} else {
			cmd = exec.CommandContext(ctx, bin, args...)
		}
		cmd.Dir = absProjectDir
		// A detekt launcher script picks its own JVM from JAVA_HOME/PATH. If we
		// resolved a compatible one, force it on the child: otherwise the script
		// starts under, say, Java 25 and dies inside the bundled IntelliJ
		// platform with IllegalArgumentException: 25.0.1.
		cmd.Env = envWithJavaHome(os.Environ(), javaPath)
		out := opts.Stdout
		if out == nil {
			out = io.Discard
		}
		cmd.Stdout, cmd.Stderr = out, out
		return cmd.Run()
	}

	err = runOnce(firstNonEmpty(projectConfig, autoConfig))
	if err != nil {
		// Safety net: detekt exits non-zero when it finds issues. If it still
		// produced a SARIF report, those findings are exactly what we came for.
		if stat, statErr := os.Stat(opts.SARIFOutput); statErr == nil && stat.Size() > 0 {
			return opts.SARIFOutput, nil
		}

		// Exit code 3 means detekt rejected the config file, not the code. That
		// is usually a stale detekt.yml (older kdoctor releases generated one
		// naming style>UnusedImport, which detekt spells UnusedImports). Losing
		// 33 rules over a typo in a generated file is a bad trade, so retry with
		// our own config and let the caller report it.
		if exitCode(err) == detektInvalidConfig && projectConfig != "" && autoConfig != "" {
			if retryErr := runOnce(autoConfig); retryErr == nil {
				return opts.SARIFOutput, ErrProjectConfigRejected{Path: projectConfig}
			} else if stat, statErr := os.Stat(opts.SARIFOutput); statErr == nil && stat.Size() > 0 {
				return opts.SARIFOutput, ErrProjectConfigRejected{Path: projectConfig}
			}
		}
		return "", fmt.Errorf("detekt standalone: %w", err)
	}
	return opts.SARIFOutput, nil
}

// detektInvalidConfig is detekt CLI exit code for "invalid config property".
const detektInvalidConfig = 3

// ErrProjectConfigRejected reports that the scan succeeded, but only after
// falling back from the project detekt config that detekt refused to load.
// It is deliberately not fatal: the caller gets findings AND something
// actionable to tell the user.
type ErrProjectConfigRejected struct{ Path string }

func (e ErrProjectConfigRejected) Error() string {
	return fmt.Sprintf("%s was rejected by detekt (invalid properties); scanned with kdoctor defaults instead. "+
		"Regenerate it with `kdoctor init --force`.", e.Path)
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// findProjectDetektConfig returns the project detekt config, or "".
func findProjectDetektConfig(projectDir string) string {
	for _, rel := range []string{
		"detekt.yml", "detekt.yaml",
		filepath.Join("config", "detekt", "detekt.yml"),
		filepath.Join("config", "detekt", "detekt.yaml"),
	} {
		p := filepath.Join(projectDir, rel)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// writeAutoConfig drops kdoctor own minimal detekt config in the temp dir, so
// a freshly cloned project needs no manual detekt setup. Returns "" on error.
func writeAutoConfig() string {
	path := filepath.Join(os.TempDir(), "kdoctor-auto-detekt.yml")
	const content = `
naming:
  active: true
  FunctionNaming:
    active: true
    # Compose functions are PascalCase by convention.
    ignoreAnnotated:
      - 'Composable'
    # Kotlin test functions are conventionally written as backticked
    # sentences. Flagging them produced hundreds of false positives
    # that drowned out every real finding.
    excludes:
      - '**/test/**'
      - '**/androidTest/**'
      - '**/androidHostTest/**'
      - '**/commonTest/**'
      - '**/iosTest/**'
      - '**/jvmTest/**'
      - '**/*Test.kt'
      - '**/*Tests.kt'
      - '**/*Spec.kt'
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ""
	}
	return path
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

// needsJava reports whether the resolved detekt binary is a .jar that must be
// launched through a JVM. A native launcher script brings its own.
func needsJava(bin string) bool {
	return strings.HasSuffix(strings.ToLower(bin), ".jar")
}

// envWithJavaHome returns env with JAVA_HOME pointing at the JDK that owns
// javaPath (.../bin/java -> ...), and that bin dir prepended to PATH. Returns
// env untouched when javaPath is empty.
func envWithJavaHome(env []string, javaPath string) []string {
	if javaPath == "" {
		return env
	}
	binDir := filepath.Dir(javaPath)
	javaHome := filepath.Dir(binDir)
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		upper := strings.ToUpper(kv)
		switch {
		case strings.HasPrefix(upper, "JAVA_HOME="):
			continue
		case strings.HasPrefix(upper, "PATH="):
			out = append(out, kv[:len("PATH=")]+binDir+string(os.PathListSeparator)+kv[len("PATH="):])
		default:
			out = append(out, kv)
		}
	}
	return append(out, "JAVA_HOME="+javaHome)
}
