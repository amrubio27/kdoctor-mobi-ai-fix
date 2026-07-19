// kdoctor doctor: Diagnóstico del propio kdoctor y del entorno.
package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose kdoctor environment: Go, Detekt, Gradle, MobiAI, LLM providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "kdoctor doctor")
			fmt.Fprintln(out, "===========")
			fmt.Fprintf(out, "  Go version runtime: %s\n", runtime.Version())
			checks := []struct {
				name string
				bin  string
			}{
				{"detekt", "detekt"},
				{"java (for gradle)", "java"},
				{"gradle wrapper (project)", "./gradlew"}, // checked per-project
				{"git", "git"},
				{"claude code CLI", "claude"},
				{"gemini CLI", "gemini"},
				{"codex CLI", "codex"},
				{"mobiai CLI", "mobiai"},
			}
			for _, c := range checks {
				p, err := exec.LookPath(c.bin)
				if err != nil {
					fmt.Fprintf(out, "  \u00d7 %s: NOT FOUND\n", c.name)
					continue
				}
				fmt.Fprintf(out, "  \u2713 %s: %s\n", c.name, p)
			}
			return nil
		},
	}
	return cmd
}
