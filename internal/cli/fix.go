package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/aifixer/patchguard"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/aifixer/remediation"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/grader"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/rulemap"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/rules"
)

// NewFixCmd builds the `fix` command.
//
// It used to take an injectable provider so tests could stub the LLM call.
// There is no LLM call any more, so there is nothing to inject: the command
// reads findings and writes a plan, both of which are directly observable.
func NewFixCmd() *cobra.Command {
	var ai bool
	var mode string
	var preferStandalone bool
	var projectDir string
	var detektBin string
	var contextLines int
	var detektMode string
	var noDownload bool
	var asJSON bool
	var validate bool

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Emit a remediation plan for the findings (for an agent to apply)",
		Long: "Scan the project and emit a remediation plan: for every finding, the " +
			"source window around it, the rule, the fix hint and the exact line range to " +
			"replace.\n\nkdoctor does not call a language model and never edits your " +
			"source files. The agent that invoked kdoctor already has a model; this " +
			"command gives it what it needs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --ai and --mode described a model kdoctor no longer implements: it does
			// not call an LLM and never writes to source files. Both are accepted so
			// existing scripts keep running, with one warning instead of an error.
			if ai {
				fmt.Fprintln(cmd.ErrOrStderr(), "Note: --ai is deprecated and ignored. kdoctor emits a remediation plan for your agent instead of calling a model itself.")
			}
			if mode == "auto" || mode == "interactive" {
				fmt.Fprintf(cmd.ErrOrStderr(), "Note: --mode=%s is deprecated and ignored. kdoctor no longer edits source files; apply the plan yourself and validate it with `kdoctor fix --validate`.\n", mode)
			}

			// 1. Scan the project
			wd := projectDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			sarifPath := filepath.Join(os.TempDir(), "kdoctor-detekt-fix.sarif")
			out := cmd.OutOrStdout()

			// --validate only inspects files the agent already wrote; scanning
			// first would cost a detekt run for nothing.
			if validate {
				return runValidate(cmd, wd, args)
			}

			// Progress goes to stderr so --json leaves stdout parseable.
			fmt.Fprintln(cmd.ErrOrStderr(), "Scanning project for issues...")

			// Load the catalog first: the detekt phase reports coverage loss in
			// terms of it when it has to degrade.
			loadResult, err := rulemap.LoadRulesCascade(wd, "")
			if err != nil {
				return fmt.Errorf("load rules: %w", err)
			}
			ruleCatalog := loadResult.Rules

			raw := runDetektPhase(context.Background(), cmd, detektPhaseOptions{
				ProjectDir:  wd,
				SARIFPath:   sarifPath,
				ExplicitBin: detektBin,
				Mode:        detektModeFromFlags(detektMode),
				NoDownload:  noDownload,
				Catalog:     ruleCatalog,
			})

			// Run native rules
			nativeFindings, err := rules.RunRegexDetectors(wd, ruleCatalog)
			if err != nil {
				return fmt.Errorf("run native rules: %w", err)
			}
			raw = append(raw, nativeFindings...)

			idx := rulemap.BuildIndex(ruleCatalog)
			mapped := idx.Map(raw)

			if len(mapped) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No issues found to fix.")
				return nil
			}

			// 2. Generate fixes
			fmt.Fprintf(cmd.ErrOrStderr(), "Found %d issue(s).\n", len(mapped))

			// kdoctor no longer calls a language model. It emits the context an agent
			// needs -- source window, rule, hint, exact line range -- and lets the
			// model that invoked kdoctor produce the patch. See
			// internal/aifixer/remediation for why.
			read := func(rel string) (string, error) {
				path := rel
				if !filepath.IsAbs(path) {
					path = filepath.Join(wd, rel)
				}
				b, err := os.ReadFile(path)
				return string(b), err
			}

			totalLines := grader.CountKotlinLines(wd)
			score, _ := grader.ScoreWithKLOC(mapped, totalLines)
			plan, skipped := remediation.Build(wd, score, mapped, contextLines, read)
			for _, f := range skipped {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not read %s; its findings were left out of the plan.\n", f)
			}

			if asJSON {
				return remediation.WriteJSON(plan, out)
			}

			// Write the plan next to the code it describes, not wherever the process
			// happened to start.
			fixesFile := filepath.Join(wd, "fixes.md")
			f, err := os.Create(fixesFile)
			if err != nil {
				return fmt.Errorf("create %s: %w", fixesFile, err)
			}
			defer func() { _ = f.Close() }()
			if err := remediation.WriteMarkdown(plan, f); err != nil {
				return fmt.Errorf("write %s: %w", fixesFile, err)
			}

			fmt.Fprintf(out, "Remediation plan for %d finding(s) written to %s\n", len(plan.Items), fixesFile)
			fmt.Fprintf(out, "Health Score: %d/100\n", score)

			return nil
		},
	}

	cmd.Flags().BoolVar(&ai, "ai", false, "deprecated: ignored, kdoctor no longer calls a model")
	cmd.Flags().StringVar(&mode, "mode", "suggest", "deprecated: ignored, kdoctor never writes to source files")
	cmd.Flags().BoolVar(&preferStandalone, "prefer-standalone", false, "deprecated: standalone is now the default")
	cmd.Flags().StringVar(&projectDir, "project-dir", "", "project directory to fix (default: cwd)")
	cmd.Flags().StringVar(&detektBin, "detekt-bin", "", "explicit path to detekt binary")
	cmd.Flags().StringVar(&detektMode, "detekt-mode", "auto", "how to run detekt: auto|standalone|gradle")
	cmd.Flags().BoolVar(&noDownload, "no-download", false, "never fetch detekt over the network")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the remediation plan as JSON on stdout (for agents and the MCP server)")
	cmd.Flags().BoolVar(&validate, "validate", false, "check that the given Kotlin files still parse (run this after applying a plan)")
	cmd.Flags().IntVar(&contextLines, "context-lines", 10, "source lines of context to include around each finding (0 or negative falls back to 10)")

	return cmd
}

// runValidate is the other half of handing patching to the agent: kdoctor no
// longer writes files, but it still owns the cheap structural check that used
// to gate its own writes. Running it after an agent edits Kotlin catches the
// failure mode that matters -- a patch that dropped or duplicated a brace --
// without needing a compiler.
//
// This is a sanity check, not a parser: it counts delimiters while respecting
// strings, raw strings, char literals, templates and comments. Balanced code
// can still be wrong.
func runValidate(cmd *cobra.Command, wd string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("--validate needs at least one file, e.g. kdoctor fix --validate src/Foo.kt")
	}
	out := cmd.OutOrStdout()
	var failed int
	for _, rel := range args {
		path := rel
		if !filepath.IsAbs(path) {
			path = filepath.Join(wd, rel)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", rel, err)
			failed++
			continue
		}
		if err := patchguard.Validate(string(src)); err != nil {
			fmt.Fprintf(out, "FAIL %s: %v\n", rel, err)
			failed++
			continue
		}
		fmt.Fprintf(out, "ok   %s\n", rel)
	}
	if failed > 0 {
		return fmt.Errorf("%d file(s) failed validation", failed)
	}
	return nil
}
