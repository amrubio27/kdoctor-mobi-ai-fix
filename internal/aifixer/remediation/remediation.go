// Package remediation turns findings into a remediation plan an agent can act
// on, instead of kdoctor calling a language model itself.
//
// Why kdoctor does not call an LLM:
//
//   - It is always invoked from something that already has one (Claude Code,
//     Cursor, MobiAI, a CI agent). Spawning a second model duplicates cost and
//     latency for no gain.
//   - The previous design ran one CLI invocation per finding, unbounded: a
//     300-finding project meant 300 subprocesses.
//   - It could destroy code. When the provider failed, the command synthesised
//     a "// Provider failed" patch, which is brace-balanced, so patchguard
//     accepted it and --mode=auto wrote it over real source.
//   - It never worked anyway: the provider shelled out to `claude --file`,
//     which is not a flag Claude Code has.
//
// So kdoctor emits the context (source window, rule, fix hint, exact line range
// to replace) and lets the caller's model produce the patch. patchguard remains
// available to validate whatever comes back.
package remediation

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/aifixer/qualityprompt"
	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
)

// SchemaVersion identifies the remediation plan format. Consumers should check
// it before trusting field names.
const SchemaVersion = "1"

// Item is one finding plus everything needed to fix it without re-reading the
// whole file.
type Item struct {
	ID       string `json:"id"`
	Rule     string `json:"rule,omitempty"`
	Cluster  string `json:"cluster"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	FixHint  string `json:"fixHint,omitempty"`

	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column,omitempty"`

	// ReplaceFrom and ReplaceTo are 1-based, inclusive: the exact line range
	// Context covers, so a patch can be applied without guessing boundaries.
	ReplaceFrom int `json:"replaceFrom"`
	ReplaceTo   int `json:"replaceTo"`

	// Context is the numbered source window around the finding.
	Context string `json:"context"`
}

// Plan is the whole remediation output.
type Plan struct {
	SchemaVersion string `json:"schemaVersion"`
	ProjectDir    string `json:"projectDir"`
	HealthScore   int    `json:"healthScore"`
	Instructions  string `json:"instructions"`
	Items         []Item `json:"items"`
}

// Instructions is the guidance handed to whatever model consumes the plan.
// It used to be a system prompt sent to a subprocess; now it travels with the
// data, so the caller's model sees the same rules.
const Instructions = `Fix each item below in the project's own style.
For every item, replace lines replaceFrom..replaceTo of the given file.
Follow modern Kotlin K2 and Jetpack Compose practice:
- Prefer collectAsStateWithLifecycle() over collectAsState() to avoid state leaks.
- Use atomic _uiState.update { ... } in ViewModels rather than assigning .value.
- Annotate UI state models with @Immutable or @Stable for K2 stability.
- Give LazyColumn/LazyRow items an explicit key.
Change only what the finding is about; do not reformat surrounding code.`

// SourceReader supplies file contents. Injected so callers control path
// resolution and tests need no filesystem.
type SourceReader func(path string) (string, error)

// Build assembles a plan. Findings whose source cannot be read are skipped,
// and their paths returned, so the caller can report them rather than failing
// the whole run over one unreadable file.
func Build(projectDir string, score int, findings []types.Finding, contextLines int, read SourceReader) (Plan, []string) {
	plan := Plan{
		SchemaVersion: SchemaVersion,
		ProjectDir:    projectDir,
		HealthScore:   score,
		Instructions:  Instructions,
		Items:         make([]Item, 0, len(findings)),
	}
	var skipped []string

	for _, f := range findings {
		if f.File == "" {
			continue
		}
		src, err := read(f.File)
		if err != nil {
			skipped = append(skipped, f.File)
			continue
		}
		context, err := qualityprompt.BuildPromptWithContext(f, src, contextLines)
		if err != nil {
			skipped = append(skipped, f.File)
			continue
		}
		lines := qualityprompt.SplitLines(src)
		start, end := qualityprompt.SliceRange(f.Line, contextLines, len(lines))

		plan.Items = append(plan.Items, Item{
			ID:       f.ID,
			Rule:     f.Rule,
			Cluster:  f.Cluster,
			Severity: string(f.Severity),
			Message:  f.Message,
			FixHint:  f.FixHint,
			File:     f.File,
			Line:     f.Line,
			Column:   f.Column,
			// SliceRange is 0-based half-open; the plan speaks 1-based inclusive
			// because that is how editors and humans count lines.
			ReplaceFrom: start + 1,
			ReplaceTo:   end,
			Context:     context,
		})
	}
	return plan, skipped
}

// WriteJSON emits the plan for agents and the MCP server.
func WriteJSON(p Plan, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

// WriteMarkdown emits the plan for humans reviewing it in a file.
func WriteMarkdown(p Plan, w io.Writer) error {
	var b strings.Builder
	b.WriteString("# kdoctor remediation plan\n\n")
	fmt.Fprintf(&b, "Health Score: **%d/100** · %d item(s)\n\n", p.HealthScore, len(p.Items))
	b.WriteString("> Generated by `kdoctor fix`. kdoctor does not call a language model:\n")
	b.WriteString("> this file is the context an agent needs to produce the patches.\n\n")
	b.WriteString("## Instructions\n\n")
	b.WriteString(p.Instructions)
	b.WriteString("\n\n")

	for i, it := range p.Items {
		fmt.Fprintf(&b, "## %d. `%s` — %s:%d\n\n", i+1, it.ID, filepath.ToSlash(it.File), it.Line)
		fmt.Fprintf(&b, "- **Severity:** %s · **Cluster:** %s\n", it.Severity, it.Cluster)
		if it.Message != "" {
			fmt.Fprintf(&b, "- **Issue:** %s\n", it.Message)
		}
		if it.FixHint != "" {
			fmt.Fprintf(&b, "- **Hint:** %s\n", it.FixHint)
		}
		fmt.Fprintf(&b, "- **Replace lines %d-%d**\n\n", it.ReplaceFrom, it.ReplaceTo)
		b.WriteString(it.Context)
		b.WriteString("\n\n---\n\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}
