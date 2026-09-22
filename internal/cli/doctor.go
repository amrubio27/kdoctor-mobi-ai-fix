// kdoctor doctor: reports what the next scan will actually be able to evaluate.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/detektrunner"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/rulemap"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/toolchain"
)

func NewDoctorCmd() *cobra.Command {
	var projectDir string
	var provision bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Report what the next scan can evaluate, and why",
		Long: "Diagnose the environment a scan depends on: the Java runtime, the detekt " +
			"binary, the Gradle wrapper, and how many catalog rules that combination can " +
			"actually evaluate.\n\nThe previous version listed whether a handful of binaries " +
			"were on PATH, which answered none of the questions people ask it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			wd := projectDir
			if wd == "" {
				if len(args) > 0 {
					wd = args[0]
				} else {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("getwd: %w", err)
					}
					wd = cwd
				}
			}

			fmt.Fprintln(out, "kdoctor doctor")
			fmt.Fprintln(out, "==============")
			fmt.Fprintf(out, "  Project : %s\n", wd)
			fmt.Fprintf(out, "  Runtime : Go %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
			fmt.Fprintln(out)

			reportJava(out)
			plan := reportDetekt(cmd, out, wd, provision)
			reportGradle(out, wd)
			reportCoverage(out, wd, plan)

			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project-dir", "", "project to diagnose (default: cwd)")
	// Downloading detekt lazily means the first scan pays for it, which looks
	// like a hang. This makes that cost explicit and schedulable.
	cmd.Flags().BoolVar(&provision, "provision", false, "download detekt now instead of on the first scan")

	return cmd
}

func reportJava(out io.Writer) {
	j, err := toolchain.ResolveJava()
	switch {
	case errors.Is(err, toolchain.ErrNoJava):
		fmt.Fprintf(out, "  %s Java    : not found\n", cross)
		fmt.Fprintln(out, "            Looked in KDOCTOR_JAVA, JAVA_HOME, PATH, Android Studio JBR, ~/.gradle/jdks")
		fmt.Fprintf(out, "            detekt needs a JDK %d-%d.\n", toolchain.MinDetektJavaMajor, toolchain.MaxDetektJavaMajor)
	case errors.Is(err, toolchain.ErrIncompatibleJava):
		// The case that actually bites: a JVM is installed, and detekt still
		// cannot load it.
		fmt.Fprintf(out, "  %s Java    : %d is too new for detekt (needs %d-%d)\n",
			cross, j.Major, toolchain.MinDetektJavaMajor, toolchain.MaxDetektJavaMajor)
		fmt.Fprintf(out, "            found %s via %s\n", j.Path, j.Via)
		fmt.Fprintln(out, "            Install a supported JDK, or set KDOCTOR_JAVA to one you have.")
	case err != nil:
		fmt.Fprintf(out, "  %s Java    : %v\n", cross, err)
	default:
		fmt.Fprintf(out, "  %s Java    : %d  (%s)\n", tick, j.Major, j.Via)
		fmt.Fprintf(out, "            %s\n", j.Path)
	}
}

func reportDetekt(cmd *cobra.Command, out io.Writer, wd string, provision bool) detektrunner.Plan {
	var progress io.Writer
	if provision {
		progress = out
	}

	plan, err := detektrunner.Resolve(context.Background(), detektrunner.ResolveOptions{
		ProjectDir: wd,
		// Without --provision, report what is available rather than fetching
		// 50 MB as a side effect of asking a question.
		NoDownload: !provision,
		Progress:   progress,
	})
	if err != nil {
		fmt.Fprintf(out, "  %s detekt  : %v\n", cross, err)
		return plan
	}
	if plan.Degraded {
		fmt.Fprintf(out, "  %s detekt  : unavailable\n", cross)
		for _, line := range splitLines(plan.Reason) {
			fmt.Fprintf(out, "            %s\n", line)
		}
		if !provision {
			fmt.Fprintln(out, "            Run `kdoctor doctor --provision` to download it now.")
		}
		return plan
	}
	fmt.Fprintf(out, "  %s detekt  : %s\n", tick, plan.Source)
	if plan.BinPath != "" {
		fmt.Fprintf(out, "            %s\n", plan.BinPath)
	}
	return plan
}

func reportGradle(out io.Writer, wd string) {
	// The old check did exec.LookPath("./gradlew"), which cannot resolve a
	// relative path and so always reported NOT FOUND.
	for _, name := range []string{"gradlew.bat", "gradlew"} {
		p := filepath.Join(wd, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			fmt.Fprintf(out, "  %s Gradle  : %s\n", tick, p)
			fmt.Fprintln(out, "            Use --detekt-mode=gradle to run detekt through the project build.")
			return
		}
	}
	fmt.Fprintf(out, "  - Gradle  : no wrapper in this project (not required)\n")
}

// reportCoverage answers the question the command exists for: how much of the
// catalog will the next scan actually evaluate?
func reportCoverage(out io.Writer, wd string, plan detektrunner.Plan) {
	loadResult, err := rulemap.LoadRulesCascade(wd, "")
	if err != nil {
		fmt.Fprintf(out, "\n  %s Rules   : could not load catalog: %v\n", cross, err)
		return
	}
	native, viaDetekt := ruleCoverage(loadResult.Rules)
	total := native + viaDetekt

	fmt.Fprintf(out, "\n  Rules     : %d catalogued, %d live (%d native + %d via detekt)\n",
		len(loadResult.Rules), total, native, viaDetekt)
	fmt.Fprintf(out, "            source: %s\n", loadResult.Source)

	if plan.Degraded {
		fmt.Fprintf(out, "\n  Next scan : %s %d/%d rules -- detekt is unavailable\n", cross, native, total)
		return
	}
	fmt.Fprintf(out, "\n  Next scan : %s %d/%d rules\n", tick, total, total)
}

const (
	tick  = "✓"
	cross = "×"
)

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
