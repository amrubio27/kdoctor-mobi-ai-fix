package detektrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/toolchain"
)

// DetektMode is the user-facing strategy selector (--detekt-mode).
type DetektMode string

const (
	// ModeAuto picks the first strategy that actually works.
	ModeAuto DetektMode = "auto"
	// ModeStandaloneOnly forbids the Gradle path.
	ModeStandaloneOnly DetektMode = "standalone"
	// ModeGradleOnly opts in to ./gradlew detekt, honouring whatever detekt
	// configuration the project already has in its build.
	ModeGradleOnly DetektMode = "gradle"
)

// Plan is the outcome of resolution: what will run, with what, and why.
//
// Reason is always populated, including on success, because "which detekt did
// you actually use?" was previously unanswerable: the old Detect() returned a
// bare enum, and the CLI reported a raw exec error when things went wrong.
type Plan struct {
	Mode     ExecutionMode
	BinPath  string // jar or executable; empty in gradlew mode
	JavaPath string // JVM used to launch a .jar; empty when not needed
	Source   string // where BinPath came from: PATH, cache, downloaded, ...
	Reason   string // human-readable explanation, shown by --verbose and doctor
	Degraded bool   // true when detekt cannot run and only native rules will
}

// ResolveOptions configures Resolve.
type ResolveOptions struct {
	ProjectDir  string
	ExplicitBin string     // --detekt-bin
	Mode        DetektMode // --detekt-mode
	NoDownload  bool       // --no-download
	Progress    io.Writer  // optional, for first-run download feedback
	// SkipPATH ignores any detekt found on PATH. Used to retry with a
	// provisioned jar after a PATH detekt turned out to be broken.
	SkipPATH bool
}

// Resolve decides how to run detekt.
//
// The ordering deliberately differs from the old Detect(): the mere existence
// of ./gradlew no longer wins. Every Android and KMP project has a gradlew, so
// that rule meant the Gradle path was taken essentially always, and it was
// broken, which is how 33 of the 53 live rules came to never fire in practice.
// Gradle is now opt-in via --detekt-mode=gradle.
//
// Resolve does not return an error for "detekt is unavailable": that is a
// degraded Plan, because kdoctor still has 20 native detectors that need no JVM
// at all. An error means the request itself was contradictory.
func Resolve(ctx context.Context, opts ResolveOptions) (Plan, error) {
	mode := opts.Mode
	if mode == "" {
		mode = ModeAuto
	}

	// 1. An explicit binary is an instruction, not a hint.
	if bin := strings.TrimSpace(opts.ExplicitBin); bin != "" {
		plan := Plan{
			Mode:    ModeStandalone,
			BinPath: bin,
			Source:  "--detekt-bin",
			Reason:  fmt.Sprintf("using detekt binary given on the command line: %s", bin),
		}
		if needsJava(bin) {
			j, err := toolchain.ResolveJava()
			if err != nil {
				return degraded(fmt.Sprintf(
					"--detekt-bin %s is a jar, but no usable Java runtime was found: %v", bin, err)), nil
			}
			plan.JavaPath = j.Path
			plan.Reason += fmt.Sprintf(" (Java %d via %s)", j.Major, j.Via)
		}
		return plan, nil
	}

	// 2. Gradle only when asked for it explicitly.
	if mode == ModeGradleOnly {
		if wrapper := gradleWrapper(opts.ProjectDir); wrapper != "" {
			return Plan{
				Mode:   ModeGradleWrap,
				Source: "gradlew",
				Reason: fmt.Sprintf("--detekt-mode=gradle, using %s", wrapper),
			}, nil
		}
		return Plan{}, fmt.Errorf(
			"--detekt-mode=gradle requires a Gradle wrapper, none found in %s", opts.ProjectDir)
	}

	// 3. A detekt the user installed themselves takes precedence over one we
	//    would fetch, but only if it actually runs. A launcher script that
	//    cannot find its own jar is worse than no detekt at all, because the
	//    failure surfaces as an opaque exit code in the middle of a scan.
	if p, err := exec.LookPath("detekt"); err == nil && !opts.SkipPATH && runsOK(ctx, p) {
		return Plan{
			Mode:    ModeStandalone,
			BinPath: p,
			Source:  "PATH",
			Reason:  fmt.Sprintf("using detekt found on PATH: %s", p),
		}, nil
	}

	// 4/5. Cached jar, or fetch one. Resolve Java first: downloading ~50 MB to
	//      run it with a JVM that does not exist helps nobody.
	j, javaErr := toolchain.ResolveJava()
	if javaErr != nil {
		return degraded(javaAdvice(javaErr, j)), nil
	}

	jar, source, err := EnsureDetektJar(ctx, ProvisionOptions{
		NoDownload: opts.NoDownload,
		Progress:   opts.Progress,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrProvisionDisabled):
			return degraded("detekt is not cached and downloads are disabled " +
				"(--no-download / KDOCTOR_NO_DOWNLOAD).\n" +
				"  Run `kdoctor doctor --provision` once with network access, or pass --detekt-bin."), nil
		case errors.Is(err, ErrChecksumMismatch):
			return degraded(fmt.Sprintf(
				"the downloaded detekt jar failed checksum verification and was discarded: %v", err)), nil
		default:
			return degraded(fmt.Sprintf("could not provision detekt: %v", err)), nil
		}
	}

	return Plan{
		Mode:     ModeStandalone,
		BinPath:  jar,
		JavaPath: j.Path,
		Source:   string(source),
		Reason: fmt.Sprintf("using detekt %s (%s) with Java %d via %s",
			detektVersion(), source, j.Major, j.Via),
	}, nil
}

func degraded(reason string) Plan {
	return Plan{Degraded: true, Reason: reason}
}

// javaAdvice turns a resolution failure into something the user can act on.
// "exec: java: executable file not found" is technically true and practically
// useless, especially in the Java-25-on-PATH case, where a JVM is installed and
// the real problem is that detekt cannot load it.
func javaAdvice(err error, j toolchain.Java) string {
	if errors.Is(err, toolchain.ErrIncompatibleJava) {
		return fmt.Sprintf(
			"detekt needs Java %d-%d, but the only runtime found is Java %d (%s, via %s).\n"+
				"  Install a supported JDK, or point kdoctor at one you already have:\n    %s",
			toolchain.MinDetektJavaMajor, toolchain.MaxDetektJavaMajor,
			j.Major, j.Path, j.Via, setEnvHint("KDOCTOR_JAVA", exampleJavaPath()))
	}
	return fmt.Sprintf(
		"no Java runtime found, and detekt needs one.\n"+
			"  Looked in: KDOCTOR_JAVA, JAVA_HOME, PATH, Android Studio's bundled JBR, ~/.gradle/jdks\n"+
			"  Install a JDK %d-%d, or point kdoctor at one:\n    %s",
		toolchain.MinDetektJavaMajor, toolchain.MaxDetektJavaMajor,
		setEnvHint("KDOCTOR_JAVA", exampleJavaPath()))
}

func setEnvHint(name, value string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("set %s=%s", name, value)
	}
	return fmt.Sprintf("export %s=%s", name, value)
}

func exampleJavaPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\Program Files\\Eclipse Adoptium\\jdk-17\\bin\\java.exe"
	}
	return "/usr/lib/jvm/temurin-17-jdk/bin/java"
}

// gradleWrapper returns the platform-appropriate wrapper path, or "".
func gradleWrapper(projectDir string) string {
	names := []string{"gradlew"}
	if runtime.GOOS == "windows" {
		// Prefer the batch wrapper on Windows, but keep the shell one as a
		// fallback: it does run when a POSIX sh is available.
		names = []string{"gradlew.bat", "gradlew"}
	}
	for _, n := range names {
		p := filepath.Join(projectDir, n)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// runsOK checks that a detekt launcher is more than a file that exists.
func runsOK(ctx context.Context, bin string) bool {
	return exec.CommandContext(ctx, bin, "--version").Run() == nil
}

// Detect is the pre-Resolve entry point.
//
// Deprecated: use Resolve, which also reports why a strategy was chosen and can
// provision detekt. Kept so existing callers keep compiling; note that it
// preserves the OLD ordering, including the gradlew-wins-by-existing rule.
func Detect(projectDir string, preferStandalone bool, explicitBin string) ExecutionMode {
	if strings.TrimSpace(explicitBin) != "" {
		return ModeStandalone
	}
	if preferStandalone {
		if _, err := exec.LookPath("detekt"); err == nil {
			return ModeStandalone
		}
	}
	if gradleWrapper(projectDir) != "" {
		return ModeGradleWrap
	}
	return ModeStandalone
}
