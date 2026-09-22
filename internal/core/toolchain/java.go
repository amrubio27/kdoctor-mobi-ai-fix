// Package toolchain locates the external runtimes kdoctor shells out to.
//
// It exists because "java is on PATH" is neither necessary nor sufficient:
//
//   - Not necessary: Android Studio ships its own JBR, and Gradle keeps
//     downloaded JDKs under ~/.gradle/jdks. A developer can have several JVMs
//     and none of them exported on PATH.
//   - Not sufficient: detekt 1.x embeds an old IntelliJ platform whose
//     JavaVersion parser throws on feature releases it predates. On a machine
//     with Java 25 on PATH, detekt dies with
//     `ExceptionInInitializerError ... IllegalArgumentException: 25.0.1`
//     before analysing a single file. Picking the first java found is
//     therefore a bug, not a default.
package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// MaxDetektJavaMajor is the highest Java feature release known to work with the
// detekt 1.23.x line. Newer JVMs load, then crash inside the bundled IntelliJ
// platform. Raise this once kdoctor pins a detekt release that tolerates them.
const MaxDetektJavaMajor = 21

// MinDetektJavaMajor is the lowest Java feature release detekt 1.23.x supports.
const MinDetektJavaMajor = 11

// Java describes a resolved JVM.
type Java struct {
	Path  string // absolute path to the java executable
	Major int    // feature release, e.g. 17 (0 when it could not be parsed)
	Raw   string // first line of `java -version`, for diagnostics
	Via   string // how it was found, e.g. "KDOCTOR_JAVA", "PATH", "Android Studio JBR"
}

// Compatible reports whether this JVM is in the range detekt 1.23.x supports.
// A JVM whose version could not be parsed is optimistically treated as usable:
// refusing to run on a parse failure would be worse than trying.
func (j Java) Compatible() bool {
	if j.Major == 0 {
		return true
	}
	return j.Major >= MinDetektJavaMajor && j.Major <= MaxDetektJavaMajor
}

type candidate struct {
	path string
	via  string
}

// ResolveJava returns the JVM kdoctor should use to run detekt.
//
// Candidates are probed in order of how strongly they signal intent, and the
// first *compatible* one wins. If every candidate is incompatible the best
// candidate is still returned along with ErrIncompatibleJava, so the caller can
// tell "no JVM at all" apart from "a JVM detekt cannot use" and say something
// actionable either way.
func ResolveJava() (Java, error) {
	var first *Java

	for _, c := range javaCandidates() {
		j, err := probe(c)
		if err != nil {
			continue
		}
		if j.Compatible() {
			return j, nil
		}
		if first == nil {
			cp := j
			first = &cp
		}
	}

	if first != nil {
		return *first, fmt.Errorf("%w: found Java %d at %s (via %s); detekt supports %d-%d",
			ErrIncompatibleJava, first.Major, first.Path, first.Via,
			MinDetektJavaMajor, MaxDetektJavaMajor)
	}
	return Java{}, ErrNoJava
}

// javaCandidates lists every plausible java executable, most explicit first.
func javaCandidates() []candidate {
	var out []candidate
	add := func(p, via string) {
		if p == "" {
			return
		}
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			return
		}
		out = append(out, candidate{path: p, via: via})
	}

	// 1. Explicit override always wins the ordering.
	add(os.Getenv("KDOCTOR_JAVA"), "KDOCTOR_JAVA")

	// 2. JAVA_HOME before PATH: it is a deliberate choice, PATH is ambient.
	if home := os.Getenv("JAVA_HOME"); home != "" {
		add(filepath.Join(home, "bin", javaExe()), "JAVA_HOME")
	}

	// 3. PATH.
	if p, err := exec.LookPath("java"); err == nil {
		add(p, "PATH")
	}

	// 4. Android Studio's bundled JetBrains Runtime — present on most machines
	//    that have an Android project at all, and always a supported version.
	for _, p := range androidStudioJBRs() {
		add(p, "Android Studio JBR")
	}

	// 5. JDKs Gradle downloaded via toolchain auto-provisioning.
	if home, err := os.UserHomeDir(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(home, ".gradle", "jdks", "*", "bin", javaExe()))
		for _, m := range matches {
			add(m, "Gradle toolchain")
		}
		// Some distributions nest one level deeper (jdks/<id>/<dist>/bin/java).
		deeper, _ := filepath.Glob(filepath.Join(home, ".gradle", "jdks", "*", "*", "bin", javaExe()))
		for _, m := range deeper {
			add(m, "Gradle toolchain")
		}
	}

	return out
}

func androidStudioJBRs() []string {
	var roots []string
	switch runtime.GOOS {
	case "windows":
		for _, env := range []string{"LOCALAPPDATA", "ProgramFiles"} {
			if base := os.Getenv(env); base != "" {
				roots = append(roots,
					filepath.Join(base, "Programs", "Android Studio", "jbr", "bin", "java.exe"),
					filepath.Join(base, "Android", "Android Studio", "jbr", "bin", "java.exe"),
				)
			}
		}
	case "darwin":
		roots = append(roots,
			"/Applications/Android Studio.app/Contents/jbr/Contents/Home/bin/java")
		if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots,
				filepath.Join(home, "Applications", "Android Studio.app", "Contents", "jbr", "Contents", "Home", "bin", "java"))
		}
	default:
		if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots,
				filepath.Join(home, "android-studio", "jbr", "bin", "java"))
		}
		roots = append(roots,
			"/opt/android-studio/jbr/bin/java",
			"/usr/local/android-studio/jbr/bin/java")
	}
	return roots
}

func javaExe() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func probe(c candidate) (Java, error) {
	// `java -version` writes to stderr on every JVM worth supporting.
	out, err := exec.Command(c.path, "-version").CombinedOutput()
	if err != nil {
		return Java{}, err
	}
	raw := firstLine(string(out))
	return Java{
		Path:  c.path,
		Major: ParseMajor(raw),
		Raw:   raw,
		Via:   c.via,
	}, nil
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

var versionRe = regexp.MustCompile(`"([0-9][0-9._]*)"`)

// ParseMajor extracts the Java feature release from a `java -version` line,
// handling both the legacy 1.8.0_x scheme and the modern 17.0.11 one.
// Returns 0 when nothing recognisable is present.
func ParseMajor(versionLine string) int {
	m := versionRe.FindStringSubmatch(versionLine)
	if m == nil {
		return 0
	}
	parts := strings.FieldsFunc(m[1], func(r rune) bool { return r == '.' || r == '_' })
	if len(parts) == 0 {
		return 0
	}
	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	// "1.8.0_402" means Java 8.
	if first == 1 && len(parts) > 1 {
		if second, err := strconv.Atoi(parts[1]); err == nil {
			return second
		}
	}
	return first
}
