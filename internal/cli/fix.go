package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/aifixer/applier"
	"github.com/adkd/adkd/internal/aifixer/patchguard"
	"github.com/adkd/adkd/internal/aifixer/provider"
	"github.com/adkd/adkd/internal/aifixer/qualityprompt"
	"github.com/adkd/adkd/internal/core/detektrunner"
	"github.com/adkd/adkd/internal/core/rulemap"
	"github.com/adkd/adkd/internal/core/rules"
	"github.com/adkd/adkd/internal/core/sarif"
	"github.com/adkd/adkd/internal/core/types"
)

// fixProvider abstracts the AI provider so the command can be tested with
// a fake implementation that returns deterministic patches.
type fixProvider interface {
	Fix(prompt string) (string, error)
}

func NewFixCmd() *cobra.Command {
	return newFixCmdWithProvider(nil)
}

func newFixCmdWithProvider(p fixProvider) *cobra.Command {
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
			runnerMode := detektrunner.Detect(wd, preferStandalone, detektBin)
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

			if p == nil {
				p = &provider.ClaudeProvider{}
			}
			var fixesBuilder strings.Builder
			fixesBuilder.WriteString("# kdoctor AI Fixes\n\n")

			type appliedFix struct {
				file   string
				id     string
				status string
			}
			var applied []appliedFix

			for _, finding := range mapped {
				if finding.File == "" {
					continue
				}

				sourceCodeBytes, err := os.ReadFile(finding.File)
				var sourceCode string
				if err != nil {
					fmt.Fprintf(out, "Error reading %s: %v\n", finding.File, err)
					continue
				}
				sourceCode = string(sourceCodeBytes)

				prompt, err := qualityprompt.BuildPromptWithContext(finding, sourceCode, contextLines)
				if err != nil {
					fmt.Fprintf(out, "Error building prompt for %s: %v\n", finding.ID, err)
					continue
				}

				patch, err := p.Fix(prompt)
				if err != nil {
					fmt.Fprintf(out, "Provider failed for %s: %v\n", finding.ID, err)
					patch = fmt.Sprintf("```kotlin\n// Provider failed: %v\n```", err)
				}

				fixesBuilder.WriteString(fmt.Sprintf("## Fix for %s in %s:%d\n\n", finding.ID, finding.File, finding.Line))
				fixesBuilder.WriteString(patch)
				fixesBuilder.WriteString("\n\n---\n\n")

				if mode == "auto" {
					status, err := applyFix(finding, sourceCode, patch, contextLines)
					applied = append(applied, appliedFix{file: finding.File, id: finding.ID, status: status})
					if err != nil {
						fmt.Fprintf(out, "[%s] %s: %v\n", status, finding.ID, err)
					} else {
						fmt.Fprintf(out, "[%s] %s in %s\n", status, finding.ID, finding.File)
					}
				}
			}

			fixesFile := "fixes.md"
			if err := os.WriteFile(fixesFile, []byte(fixesBuilder.String()), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", fixesFile, err)
			}

			fmt.Fprintf(out, "Generated fixes and saved to %s\n", fixesFile)

			if mode == "auto" && len(applied) > 0 {
				fmt.Fprintln(out, "\nAuto-fix summary:")
				for _, a := range applied {
					fmt.Fprintf(out, "  %s: %s (%s)\n", a.status, a.id, a.file)
				}
			}

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

// applyFix applies an LLM patch to a single finding in memory, validates the
// patched source with patchguard, and writes the result to disk only if it
// passes validation. Because the patched source is computed in memory and the
// file is only written after validation succeeds, a validation failure leaves
// the original file untouched. Returns a short status string ("applied",
// "failed", "skipped") and an error if the operation could not be completed
// cleanly.
func applyFix(finding types.Finding, sourceCode string, patch string, contextLines int) (string, error) {
	// 1. Extract the replacement snippet from the LLM response.
	snippet, err := applier.ExtractCodeBlock(patch)
	if err != nil {
		return "failed", fmt.Errorf("extract code block: %w", err)
	}

	// 2. Determine the line window that was shown to the LLM.
	lines := qualityprompt.SplitLines(sourceCode)
	start, end := qualityprompt.SliceRange(finding.Line, contextLines, len(lines))
	if start == 0 && end == 0 {
		return "failed", fmt.Errorf("empty source file")
	}

	// 3. Apply the patch in memory.
	patchedCode, err := applier.ApplyPatch(sourceCode, snippet, start, end)
	if err != nil {
		return "failed", fmt.Errorf("apply patch: %w", err)
	}

	// 4. Validate the patched source. If validation fails, do NOT write the
	// file; the caller still holds the original content in memory.
	if err := patchguard.Validate(patchedCode); err != nil {
		return "failed", fmt.Errorf("patchguard validation: %w", err)
	}

	// 5. Write the patched source. If this fails, restore the original file.
	if err := os.WriteFile(finding.File, []byte(patchedCode), 0644); err != nil {
		return "failed", fmt.Errorf("write file: %w", err)
	}

	return "applied", nil
}
