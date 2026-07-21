// Package qualityprompt constructs the prompt text that gets sent to the
// AI provider (e.g. Claude CLI) when kdoctor fix --ai is invoked. The
// round-2 task #6 redesign introduces slicing so we don't send the full
// file source — only ±N lines around the finding — which keeps the prompt
// small for projects with large files (1000+ lines) and concentrates the
// LLM's attention on the relevant region.
package qualityprompt

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/adkd/adkd/internal/core/types"
)

const systemPrompt = `You are kdoctor AI Fixer. Your goal is to fix the provided code according to the issue described.
You MUST output ONLY valid code patch/replacement.
Do NOT output any conversational text, explanations, or markdown blocks around the code unless explicitly requested in a diff format.
`

// Legacy template preserved verbatim from round-1. New callers should
// prefer BuildPromptWithContext for token-efficient ±N-line slicing.
const promptTemplateStr = `
File: {{ .Finding.File }}
Line: {{ .Finding.Line }}
Column: {{ .Finding.Column }}

Issue ID: {{ .Finding.ID }}
Rule: {{ .Finding.Rule }}
Message: {{ .Finding.Message }}
{{- if .Finding.FixHint }}
Fix Hint: {{ .Finding.FixHint }}
{{- end }}

Context/Source Code:
` + "```kotlin\n{{ .SourceCode }}\n```\n" + `
Please provide the fixed code for this file. Only provide the replacement code that fixes the issue.
`

var promptTmpl = template.Must(template.New("prompt").Parse(promptTemplateStr))

type PromptData struct {
	Finding    types.Finding
	SourceCode string
}

// DefaultContextLines is the default number of lines emitted before and
// after the finding's line when BuildPromptWithContext is called. Ten is
// enough context for most detekt SARIF rules (long enough to see the
// surrounding function/block, short enough to keep prompt size small).
const DefaultContextLines = 10

// genericChangeHint is the fallback used when Finding.FixHint is empty so
// the LLM still gets actionable guidance.
const genericChangeHint = "Apply the minimal change required to resolve the rule violation while preserving surrounding semantics."

// findingMarkerInline aligns with the line prefix width of the rendered
// block so the marker doesn't desync visual columns.
const findingMarkerInline = "  <-- FINDING"

// BuildPrompt is preserved verbatim from round-1 for backward compatibility
// with the existing test suite and CLI integration. It emits the entire
// sourceCode as context — callers wanting token efficiency should switch to
// BuildPromptWithContext.
func BuildPrompt(finding types.Finding, sourceCode string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(systemPrompt)

	err := promptTmpl.Execute(&buf, PromptData{
		Finding:    finding,
		SourceCode: sourceCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	return buf.String(), nil
}

// BuildPromptWithContext emits a prompt that includes only contextLines
// lines of source code before and after finding.Line (clamped to the file
// boundaries). The output format is:
//
//	File: <basename>
//	Issue at line N
//	Rule ID:    <id>
//	Cluster:    <cluster>
//	Severity:   <severity>
//	Message:    <message>     (optional)
//	Fix Hint:   <hint>        (from Finding.FixHint if non-empty)
//	Change Hint: <hint>       (Finding.FixHint or generic fallback)
//
//	Lines X-Y of <basename> (with N marked):
//	```kotlin
//	   X: <source>
//	   ...
//	 N: <source>  <-- FINDING
//	   ...
//	   Y: <source>
//	```
//
// Defaults: contextLines <= 0 falls back to DefaultContextLines (10).
// finding.Line <= 0 or > numLines is clamped so the marker still fits in
// the slice range.
//
// Returns the rendered prompt string. No error path is currently exercised
// (no template, no IO); the error signature is kept for forward compat
// (e.g. callers may want to validate N bounds in the future).
func BuildPromptWithContext(finding types.Finding, sourceCode string, contextLines int) (string, error) {
	if contextLines <= 0 {
		contextLines = DefaultContextLines
	}

	lines := SplitLines(sourceCode)
	start, end := SliceRange(finding.Line, contextLines, len(lines))

	var b strings.Builder
	b.WriteString(systemPrompt)

	// Header.
	b.WriteString(fmt.Sprintf("\nFile: %s\n", fileBase(finding.File)))
	b.WriteString(fmt.Sprintf("Issue at line %d\n", finding.Line))
	b.WriteString(fmt.Sprintf("Rule ID:    %s\n", finding.ID))
	b.WriteString(fmt.Sprintf("Cluster:    %s\n", finding.Cluster))
	b.WriteString(fmt.Sprintf("Severity:   %s\n", finding.Severity))
	if finding.Message != "" {
		b.WriteString(fmt.Sprintf("Message:    %s\n", finding.Message))
	}
	b.WriteString("Change Hint: ")
	if finding.FixHint != "" {
		b.WriteString(finding.FixHint)
	} else {
		b.WriteString(genericChangeHint)
	}
	b.WriteString("\n\n")

	// Body — labeled slice with absolute line numbers and FINDING marker.
	b.WriteString(fmt.Sprintf("Lines %d-%d of %s (line %d marked):\n",
		start+1, end, fileBase(finding.File), finding.Line))
	b.WriteString("```kotlin\n")
	if len(lines) == 0 {
		b.WriteString("// (empty source)\n")
	} else {
		// Pad line number width so columns align for arbitrary file sizes.
		width := numWidth(end)
		for i := start; i < end; i++ {
			line := lines[i]
			b.WriteString(fmt.Sprintf("%*d: %s", width, i+1, line))
			if i+1 == finding.Line {
				b.WriteString(findingMarkerInline)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("```\n")
	b.WriteString("\nPlease provide the fixed code for the marked line, preserving surrounding semantics.\n")

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("IMPORTANT: Replace exactly lines %d to %d shown above. "+
		"Return ONLY the replacement code for those lines. Do NOT include line numbers. "+
		"Do NOT explain. Wrap the code in a ```kotlin ... ``` fence.\n",
		start+1, end))

	return b.String(), nil
}

// SplitLines splits sourceCode on newlines without emitting a trailing
// empty entry — "a\nb\n" → ["a", "b"], which is what 1-based line
// indexing expects. Exported so applier can reuse the same normalization
// and line-counting semantics.
func SplitLines(sourceCode string) []string {
	if sourceCode == "" {
		return nil
	}
	// Strip trailing newline so a final "a\n" doesn't produce an empty
	// trailing entry in the slice.
	trimmed := strings.TrimRight(sourceCode, "\n")
	if trimmed == "" {
		// The source was only newlines.
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// SliceRange computes the half-open line range [start, end) clamped to
// [0, numLines]. findingLine is treated as 1-based and clamped if out of
// bounds, so a malformed finding.Line never produces a panic or empty slice.
// Exported so the fix applier can operate on the same window that was
// shown to the LLM.
func SliceRange(findingLine, contextLines, numLines int) (start, end int) {
	if numLines <= 0 {
		return 0, 0
	}
	// Target line in 0-based slice index. Clamp to [1, numLines].
	target := findingLine - 1
	if target < 0 {
		target = 0
	}
	if target >= numLines {
		target = numLines - 1
	}

	start = target - contextLines
	if start < 0 {
		start = 0
	}
	end = target + contextLines + 1
	if end > numLines {
		end = numLines
	}
	return start, end
}

// fileBase returns the basename of finding.File using OS-agnostic forward
// slash semantics. Because Go's `path` package only treats `/` as the
// separator (not `\`), Windows-style absolute paths like
// "C:\Users\X\proj\App.kt" must be backslash-normalized FIRST so that
// path.Base can extract the trailing segment. Without this normalization,
// a Windows user with `--project-dir=C:\...` would see the full absolute
// path echoed into every LLM prompt header.
func fileBase(p string) string {
	if p == "" {
		return p
	}
	return path.Base(strings.ReplaceAll(p, "\\", "/"))
}

// numWidth is the character width needed to render line numbers up to n
// without truncation. For n=10..99 → 2; n=100..999 → 3; etc.
func numWidth(n int) int {
	switch {
	case n < 10:
		return 1
	case n < 100:
		return 2
	case n < 1000:
		return 3
	case n < 10000:
		return 4
	default:
		return 6
	}
}
