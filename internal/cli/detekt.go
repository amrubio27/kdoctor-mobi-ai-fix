package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/detektrunner"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/sarif"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

// detektPhaseOptions describes one detekt run.
type detektPhaseOptions struct {
	ProjectDir  string
	SARIFPath   string
	ExplicitBin string
	Mode        detektrunner.DetektMode
	NoDownload  bool
	Verbose     bool
	// Catalog is used to tell the user how much coverage they are losing when
	// detekt cannot run. A count is far more actionable than "detekt failed".
	Catalog []types.Rule
}

// runDetektPhase resolves a detekt strategy, runs it, and returns whatever
// findings it produced.
//
// It never returns an error: kdoctor has 20 native detectors that need no JVM,
// so a missing or broken detekt degrades coverage instead of failing the scan.
// Both `scan` and `fix` go through here so they cannot drift apart again --
// `fix` used to abort where `scan` degraded.
func runDetektPhase(ctx context.Context, cmd *cobra.Command, opts detektPhaseOptions) []types.Finding {
	var progress io.Writer
	if opts.Verbose {
		progress = cmd.ErrOrStderr()
	}

	plan, err := detektrunner.Resolve(ctx, detektrunner.ResolveOptions{
		ProjectDir:  opts.ProjectDir,
		ExplicitBin: opts.ExplicitBin,
		Mode:        opts.Mode,
		NoDownload:  opts.NoDownload,
		// Always surface first-run download progress: a silent 50 MB fetch
		// looks like a hang.
		Progress: cmd.ErrOrStderr(),
	})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
		return nil
	}
	_ = progress

	if opts.Verbose && !plan.Degraded {
		fmt.Fprintf(cmd.OutOrStdout(), "[kdoctor] detekt strategy: %s -- %s\n", plan.Mode, plan.Reason)
	}
	if plan.Degraded {
		printDegraded(cmd, plan.Reason, opts.Catalog)
		return nil
	}

	var detektOut io.Writer = io.Discard
	if opts.Verbose {
		detektOut = cmd.OutOrStdout()
	}

	_, err = detektrunner.RunDetekt(ctx, detektrunner.Options{
		ProjectDir:     opts.ProjectDir,
		SARIFOutput:    opts.SARIFPath,
		UseStandalone:  plan.Mode == detektrunner.ModeStandalone,
		StandalonePath: plan.BinPath,
		JavaPath:       plan.JavaPath,
		Stdout:         detektOut,
		NoDownload:     opts.NoDownload,
		Progress:       cmd.ErrOrStderr(),
	})
	// Not every non-nil error is a failure: the scan can succeed after
	// falling back from a project detekt config that detekt refused to
	// load. The user still needs to hear about it.
	if warn := configRejection(err); warn != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warn)
		err = nil
	}
	if err != nil {
		// A detekt on PATH can answer --version and still be unusable: a
		// launcher whose jar moved, or one that insists on a JVM detekt
		// cannot load. Falling back to a provisioned jar turns that from a
		// 20/53-rule scan into a full one, so it is worth one retry.
		if plan.Source == "PATH" {
			if findings, ok := retryWithProvisionedJar(ctx, cmd, opts, err); ok {
				return findings
			}
		}
		printDegraded(cmd, fmt.Sprintf("detekt was found (%s) but failed to run: %v", plan.Source, err), opts.Catalog)
		return nil
	}

	file, err := os.Open(opts.SARIFPath)
	if err != nil {
		printDegraded(cmd, fmt.Sprintf("detekt ran but produced no SARIF report: %v", err), opts.Catalog)
		return nil
	}
	defer func() { _ = file.Close() }()

	parsed, err := sarif.Parse(file)
	if err != nil {
		printDegraded(cmd, fmt.Sprintf("detekt produced a SARIF report that could not be parsed: %v", err), opts.Catalog)
		return nil
	}
	return parsed
}

// printDegraded explains a partial scan once, on stderr, in terms of what the
// user loses. The previous message was a raw error dump ("detekt execution
// skipped or failed (exit status 1)") that named no consequence and no remedy.
func printDegraded(cmd *cobra.Command, reason string, catalog []types.Rule) {
	native, viaDetekt := ruleCoverage(catalog)
	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "\nPartial scan: %d of %d active rules evaluated (%d need detekt).\n",
		native, native+viaDetekt, viaDetekt)
	fmt.Fprintf(w, "Reason: %s\n\n", reason)
}

// ruleCoverage splits the live catalog into rules kdoctor evaluates natively
// and rules delegated to detekt.
func ruleCoverage(catalog []types.Rule) (native, viaDetekt int) {
	for _, r := range catalog {
		if r.Status != "live" {
			continue
		}
		if r.DetektRule != "" {
			viaDetekt++
		} else {
			native++
		}
	}
	return native, viaDetekt
}

// detektModeFromFlags maps a --detekt-mode string onto the resolver enum.
func detektModeFromFlags(mode string) detektrunner.DetektMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case string(detektrunner.ModeGradleOnly):
		return detektrunner.ModeGradleOnly
	case string(detektrunner.ModeStandaloneOnly):
		return detektrunner.ModeStandaloneOnly
	default:
		return detektrunner.ModeAuto
	}
}

// retryWithProvisionedJar re-resolves with PATH excluded and runs again.
// Reports ok=false when the retry is impossible or also fails, leaving the
// caller to explain the ORIGINAL failure rather than the retry - the user
// cares about the detekt they installed, not about our fallback.
func retryWithProvisionedJar(ctx context.Context, cmd *cobra.Command, opts detektPhaseOptions, origErr error) ([]types.Finding, bool) {
	plan, err := detektrunner.Resolve(ctx, detektrunner.ResolveOptions{
		ProjectDir: opts.ProjectDir,
		Mode:       opts.Mode,
		NoDownload: opts.NoDownload,
		SkipPATH:   true,
		Progress:   cmd.ErrOrStderr(),
	})
	if err != nil || plan.Degraded {
		return nil, false
	}

	fmt.Fprintf(cmd.ErrOrStderr(),
		"Note: the detekt on PATH failed (%v); retrying with %s.\n", origErr, plan.Source)

	var detektOut io.Writer = io.Discard
	if opts.Verbose {
		detektOut = cmd.OutOrStdout()
	}
	if _, err := detektrunner.RunDetekt(ctx, detektrunner.Options{
		ProjectDir:     opts.ProjectDir,
		SARIFOutput:    opts.SARIFPath,
		UseStandalone:  plan.Mode == detektrunner.ModeStandalone,
		StandalonePath: plan.BinPath,
		JavaPath:       plan.JavaPath,
		Stdout:         detektOut,
		NoDownload:     opts.NoDownload,
		Progress:       cmd.ErrOrStderr(),
	}); err != nil {
		return nil, false
	}

	file, err := os.Open(opts.SARIFPath)
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	parsed, err := sarif.Parse(file)
	if err != nil {
		return nil, false
	}
	return parsed, true
}

// configRejection returns a warning message when err is the recoverable
// "your detekt config was rejected, we scanned with ours" signal, else "".
func configRejection(err error) string {
	var rejected detektrunner.ErrProjectConfigRejected
	if errors.As(err, &rejected) {
		return rejected.Error()
	}
	return ""
}
