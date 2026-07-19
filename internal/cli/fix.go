package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/aifixer/provider"
	"github.com/adkd/adkd/internal/aifixer/qualityprompt"
	"github.com/adkd/adkd/internal/core/detektrunner"
	"github.com/adkd/adkd/internal/core/rulemap"
	"github.com/adkd/adkd/internal/core/rules"
	"github.com/adkd/adkd/internal/core/sarif"
)

func NewFixCmd() *cobra.Command {
	var ai bool
	var mode string
	var preferStandalone bool
	var projectDir string
	var detektBin string
	var contextLines int

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Auto-fix issues using AI or basic refactorings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !ai {
				return fmt.Errorf("currently only --ai is supported for the fix command")
			}

			if mode != "suggest" && mode != "interactive" && mode != "auto" {
				return fmt.Errorf("invalid mode %q. Use suggest, interactive, or auto", mode)
			}

			// 1. Scan the project
			wd := projectDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			runnerMode := detektrunner.Detect(wd, preferStandalone)
			sarifPath := filepath.Join(os.TempDir(), "kdoctor-detekt-fix.sarif")
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Scanning project for issues...")
			if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
				ProjectDir:     wd,
				SARIFOutput:    sarifPath,
				UseStandalone:  runnerMode == detektrunner.ModeStandalone,
				StandalonePath: detektBin,
				Stdout:         os.Stderr, // Redirect detekt output to stderr
			}); err != nil {
				return fmt.Errorf("detekt: %w", err)
			}

			f, err := os.Open(sarifPath)
			if err != nil {
				return err
			}
			defer f.Close()

			raw, err := sarif.Parse(f)
			if err != nil {
				return err
			}

			rulesPath, err := resolveRulesPath()
			if err != nil {
				return fmt.Errorf("rules metadata: %w", err)
			}
			ruleCatalog, err := rulemap.LoadRules(rulesPath)
			if err != nil {
				return err
			}

			// Run native rules
			nativeFindings, err := rules.RunRegexDetectors(wd, ruleCatalog)
			if err != nil {
				return fmt.Errorf("run native rules: %w", err)
			}
			raw = append(raw, nativeFindings...)

			idx := rulemap.BuildIndex(ruleCatalog)
			mapped := idx.Map(raw)

			if len(mapped) == 0 {
				fmt.Fprintln(out, "No issues found to fix.")
				return nil
			}

			// 2. Generate fixes
			fmt.Fprintf(out, "Found %d issues. Generating fixes in %q mode...\n", len(mapped), mode)

			claude := &provider.ClaudeProvider{}
			var fixesBuilder strings.Builder
			fixesBuilder.WriteString("# kdoctor AI Fixes\n\n")

			for _, finding := range mapped {
				if finding.File == "" {
					continue
				}

				sourceCodeBytes, err := os.ReadFile(finding.File)
				var sourceCode string
				if err != nil {
					sourceCode = fmt.Sprintf("// Failed to read file: %v", err)
				} else {
					sourceCode = string(sourceCodeBytes)
				}

				prompt, err := qualityprompt.BuildPromptWithContext(finding, sourceCode, contextLines)
				if err != nil {
					fmt.Fprintf(out, "Error building prompt for %s: %v\n", finding.ID, err)
					continue
				}

				var patch string
				// We execute provider if Claude is available, else we put a placeholder.
				// Since we are mocking / building the structure, we can try invoking it.
				// In a real environment, if claude CLI isn't installed, it errors.
				patch, err = claude.Fix(prompt)
				if err != nil {
					// Fallback to stub behavior if provider fails (e.g. no claude installed)
					patch = fmt.Sprintf("```kotlin\n// Provider failed or not configured: %v\n// Execute claude manually with the prompt.\n```", err)
				}

				fixesBuilder.WriteString(fmt.Sprintf("## Fix for %s in %s:%d\n\n", finding.ID, finding.File, finding.Line))
				fixesBuilder.WriteString(patch)
				fixesBuilder.WriteString("\n\n---\n\n")

				if mode != "suggest" {
					fmt.Fprintf(out, "Mode %q not fully implemented yet for applying patches.\n", mode)
				}
			}

			fixesFile := "fixes.md"
			if err := os.WriteFile(fixesFile, []byte(fixesBuilder.String()), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", fixesFile, err)
			}

			fmt.Fprintf(out, "Generated fixes and saved to %s\n", fixesFile)
			return nil
		},
	}

	cmd.Flags().BoolVar(&ai, "ai", false, "Use AI to fix issues")
	cmd.Flags().StringVar(&mode, "mode", "suggest", "Mode of operation: suggest|interactive|auto")
	cmd.Flags().BoolVar(&preferStandalone, "prefer-standalone", false, "prefer standalone detekt binary over ./gradlew")
	cmd.Flags().StringVar(&projectDir, "project-dir", "", "project directory to fix (default: cwd)")
	cmd.Flags().StringVar(&detektBin, "detekt-bin", "", "explicit path to detekt binary")
	cmd.Flags().IntVar(&contextLines, "context-lines", 10, "number of source lines to include before/after finding.Line in the prompt (0 or negative falls back to 10)")

	return cmd
}
