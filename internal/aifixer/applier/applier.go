// Package applier turns an LLM response into an actual source-code patch,
// applies it in memory, and lets callers validate the result before writing
// it to disk. It is intentionally conservative: it never touches the file
// system directly; the caller is responsible for reading the original source,
// writing the patched result, and rolling back if validation fails.
package applier

import (
	"fmt"
	"regexp"
	"strings"
)

// lineNumberPrefix matches leading whitespace + one or more digits + optional
// colon/period/space that LLMs sometimes emit when returning a code block
// (e.g. "12: val x = 1" or "  12  val x = 1"). It is intentionally narrow so
// it does not strip valid Kotlin labels that happen to start with digits.
var lineNumberPrefix = regexp.MustCompile(`^\s*\d+[:.)\s]\s*`)

// stripLineNumbers removes leading "<num>: " / "<num>. " markers from every
// line of a snippet. This is a best-effort cleanup for LLMs that ignore the
// "Do NOT include line numbers" instruction.
func stripLineNumbers(snippet string) string {
	lines := strings.Split(snippet, "\n")
	for i, l := range lines {
		if lineNumberPrefix.MatchString(l) {
			lines[i] = lineNumberPrefix.ReplaceAllString(l, "")
		}
	}
	return strings.Join(lines, "\n")
}

// ExtractCodeBlock finds the first fenced code block in an LLM response and
// returns its content. It supports ```kotlin ... ```, ``` ... ``` and any
// other language tag. If no fence is found but the response is non-empty, the
// trimmed response is returned as a fallback (useful when the LLM returns
// only the code without fences). Empty or whitespace-only responses produce
// an error.
func ExtractCodeBlock(response string) (string, error) {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return "", fmt.Errorf("empty LLM response, cannot extract code block")
	}

	// Locate the first opening fence ``` or ```<lang>.
	startMarker := "```"
	startIdx := strings.Index(trimmed, startMarker)
	if startIdx == -1 {
		// No fences at all: assume the entire response is the code.
		return trimmed, nil
	}

	// Move past the opening fence and its optional language tag.
	rest := trimmed[startIdx+len(startMarker):]
	rest = strings.TrimLeft(rest, " \t")
	// Skip the language tag (if present) up to newline.
	if nl := strings.Index(rest, "\n"); nl != -1 && rest[0] != '`' {
		rest = rest[nl+1:]
	}

	// Find the closing fence.
	endIdx := strings.Index(rest, startMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("malformed code block: opening fence found but no closing fence")
	}

	return strings.TrimRight(rest[:endIdx], " \t\n"), nil
}

// ApplyPatch replaces the half-open line range [start, end) of originalCode
// with snippet. All line indices are 0-based. The snippet may end with a
// newline or not; the function preserves the original file's trailing
// newline property whenever the replacement does not touch the last line.
func ApplyPatch(originalCode string, snippet string, start, end int) (string, error) {
	lines := splitLines(originalCode)
	if start < 0 || start > len(lines) {
		return "", fmt.Errorf("start index %d out of range [0,%d]", start, len(lines))
	}
	if end < start || end > len(lines) {
		return "", fmt.Errorf("end index %d out of range [%d,%d]", end, start, len(lines))
	}

	snippet = stripLineNumbers(snippet)

	snippetLines := splitLines(snippet)
	if len(snippetLines) == 1 && snippetLines[0] == "" {
		snippetLines = nil
	}

	originalHadTrailingNewline := strings.HasSuffix(originalCode, "\n")

	// Build the patched content, normalising every line with a single '\n'.
	var b strings.Builder
	for i := 0; i < start; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	for _, l := range snippetLines {
		b.WriteString(l)
		b.WriteString("\n")
	}
	for i := end; i < len(lines); i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	patched := b.String()

	// Preserve the original file's trailing-newline property unless the
	// replacement touched the last line of the file, in which case the
	// snippet's own trailing-newline property wins.
	if end < len(lines) && !originalHadTrailingNewline {
		patched = strings.TrimSuffix(patched, "\n")
	}

	return patched, nil
}

// splitLines is the same normalization used by qualityprompt: split on
// newlines, discarding a single trailing empty entry if the input ends in a
// newline. This keeps the line count consistent with the window returned by
// qualityprompt.SliceRange.
func splitLines(code string) []string {
	if code == "" {
		return nil
	}
	trimmed := strings.TrimRight(code, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
