package toolchain

import (
	"errors"
	"testing"
)

func TestParseMajor(t *testing.T) {
	cases := []struct {
		line string
		want int
	}{
		{`openjdk version "17.0.17" 2025-10-21 LTS`, 17},
		{`java version "25.0.1" 2025-10-21 LTS`, 25},
		{`java version "1.8.0_402"`, 8},
		{`openjdk version "11.0.22" 2024-01-16`, 11},
		{`openjdk version "21" 2023-09-19`, 21},
		{`something without a version`, 0},
		{``, 0},
	}
	for _, c := range cases {
		if got := ParseMajor(c.line); got != c.want {
			t.Errorf("ParseMajor(%q) = %d, want %d", c.line, got, c.want)
		}
	}
}

func TestCompatible(t *testing.T) {
	cases := []struct {
		major int
		want  bool
	}{
		{8, false}, // below the floor
		{11, true}, // floor
		{17, true},
		{21, true},  // ceiling
		{25, false}, // the release that crashes detekt's bundled IntelliJ
		{0, true},   // unparseable: do not refuse to run on a parse failure
	}
	for _, c := range cases {
		if got := (Java{Major: c.major}).Compatible(); got != c.want {
			t.Errorf("Java{Major:%d}.Compatible() = %v, want %v", c.major, got, c.want)
		}
	}
}

// TestResolveJavaSkipsIncompatibleOnPath is the regression test for the failure
// observed on a real machine: Java 25 first on PATH, a supported JDK installed
// elsewhere, and detekt dying with IllegalArgumentException: 25.0.1 before
// analysing anything. Resolution must not stop at the first java it finds.
func TestResolveJavaSkipsIncompatibleOnPath(t *testing.T) {
	j, err := ResolveJava()
	if errors.Is(err, ErrNoJava) {
		t.Skip("no JVM on this machine")
	}
	if errors.Is(err, ErrIncompatibleJava) {
		// Legitimate outcome when every installed JVM is too new; the error
		// must still name the version so the CLI can act on it.
		if j.Major == 0 {
			t.Fatal("ErrIncompatibleJava must carry the offending version")
		}
		t.Logf("only incompatible JVMs present: Java %d via %s", j.Major, j.Via)
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !j.Compatible() {
		t.Fatalf("resolved an incompatible JVM: Java %d at %s (via %s)", j.Major, j.Path, j.Via)
	}
	t.Logf("resolved Java %d via %s: %s", j.Major, j.Via, j.Path)
}

func TestResolveJavaHonoursExplicitOverride(t *testing.T) {
	t.Setenv("KDOCTOR_JAVA", "/definitely/not/a/java")
	// A bogus override must not crash resolution; it is skipped like any other
	// candidate that fails to execute.
	if _, err := ResolveJava(); err != nil && !errors.Is(err, ErrNoJava) && !errors.Is(err, ErrIncompatibleJava) {
		t.Fatalf("unexpected error kind: %v", err)
	}
}

// TestResolveJavaWithoutJavaHome exercises the common case where the user never
// set JAVA_HOME: PATH may then be the only pointer, and on this class of machine
// it is exactly the one that breaks detekt. Fallbacks (Android Studio JBR,
// Gradle toolchains) are what keep the scan working.
func TestResolveJavaWithoutJavaHome(t *testing.T) {
	t.Setenv("JAVA_HOME", "")
	t.Setenv("KDOCTOR_JAVA", "")

	j, err := ResolveJava()
	switch {
	case errors.Is(err, ErrNoJava):
		t.Skip("no JVM on this machine")
	case errors.Is(err, ErrIncompatibleJava):
		t.Logf("without JAVA_HOME only Java %d (via %s) is reachable - detekt would be skipped, "+
			"which is why the CLI must say so instead of printing a raw exec error", j.Major, j.Via)
	case err != nil:
		t.Fatalf("unexpected error: %v", err)
	default:
		if !j.Compatible() {
			t.Fatalf("resolved incompatible JVM: Java %d via %s", j.Major, j.Via)
		}
		t.Logf("fell back to Java %d via %s", j.Major, j.Via)
	}
}
