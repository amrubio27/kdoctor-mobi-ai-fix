package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/adkd/adkd/internal/core/types"
)

// Regex patterns
var (
	piiRegex           = regexp.MustCompile(`(?i)(password|email|token|secret|phone|ssn|credential|pin|creditcard|@)`)
	webViewRegex       = regexp.MustCompile(`\bjavaScriptEnabled\s*=\s*true\b`)
	dispatchersRegex   = regexp.MustCompile(`\bDispatchers\.(IO|Default|Unconfined)\b`)
	itemsKeywordsRegex = regexp.MustCompile(`\b(items|itemsIndexed)\b`)
	logKeywordsRegex   = regexp.MustCompile(`\b(Log|Timber)\.[diwewtfv]\b|\bprint(ln)?\b`)
)

// Detector defines a native rule checker
type Detector interface {
	ID() string
	Check(filePath string, originalContent string, commentStripped string, commentAndStringStripped string) []types.Finding
}

// RunRegexDetectors walks the project, finds Kotlin files, runs all registered detectors, and returns findings.
func RunRegexDetectors(projectDir string, rules []types.Rule) ([]types.Finding, error) {
	// Filter live native rules
	var activeDetectors []Detector
	rulesByID := make(map[string]types.Rule)
	for _, r := range rules {
		if r.Status != "live" {
			continue
		}
		rulesByID[r.ID] = r

		// Wire detectors
		switch r.ID {
		case "compose-missing-key":
			activeDetectors = append(activeDetectors, &ComposeMissingKeyDetector{rule: r})
		case "sec-log-pii":
			activeDetectors = append(activeDetectors, &SecLogPiiDetector{rule: r})
		case "sec-webview-javascript-enabled":
			activeDetectors = append(activeDetectors, &SecWebViewJavascriptEnabledDetector{rule: r})
		case "coroutine-dispatchers-hardcoded":
			activeDetectors = append(activeDetectors, &CoroutineDispatchersHardcodedDetector{rule: r})
		}
	}

	if len(activeDetectors) == 0 {
		return nil, nil
	}

	files, err := findKotlinFiles(projectDir)
	if err != nil {
		return nil, fmt.Errorf("find Kotlin files: %w", err)
	}

	var findings []types.Finding
	for _, file := range files {
		absPath := filepath.Join(projectDir, file)
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Skip files we cannot read, but log/ignore for now
			continue
		}
		content := string(data)

		// Generate stripped versions
		commentStripped := stripComments(content)
		commentAndStringStripped := stripCommentsAndStrings(content)

		for _, det := range activeDetectors {
			fileFindings := det.Check(file, content, commentStripped, commentAndStringStripped)
			findings = append(findings, fileFindings...)
		}
	}

	return findings, nil
}

func findKotlinFiles(projectDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".kt" || ext == ".kts" {
			rel, err := filepath.Rel(projectDir, path)
			if err != nil {
				files = append(files, path)
			} else {
				files = append(files, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	return files, err
}

func isIgnoredDir(name string) bool {
	switch name {
	case "build", ".gradle", ".git", "bin", "out", "kspCaches", ".gemini", ".freebuff", "node_modules", "dist":
		return true
	}
	return false
}

// ----------------------------------------------------
// Strip Helpers
// ----------------------------------------------------

func stripComments(content string) string {
	bytes := []byte(content)
	n := len(bytes)
	out := make([]byte, n)
	copy(out, bytes)

	i := 0
	for i < n {
		// Block comment
		if i+1 < n && bytes[i] == '/' && bytes[i+1] == '*' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < n {
				if i+1 < n && bytes[i] == '*' && bytes[i+1] == '/' {
					out[i] = ' '
					out[i+1] = ' '
					i += 2
					break
				}
				if bytes[i] != '\n' && bytes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}

		// Line comment
		if i+1 < n && bytes[i] == '/' && bytes[i+1] == '/' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < n {
				if bytes[i] == '\n' || bytes[i] == '\r' {
					break
				}
				out[i] = ' '
				i++
			}
			continue
		}

		// Skip string literal to avoid confusing strings with comment starters
		if bytes[i] == '"' {
			isTriple := i+2 < n && bytes[i+1] == '"' && bytes[i+2] == '"'
			if isTriple {
				i += 3
				for i < n {
					if i+2 < n && bytes[i] == '"' && bytes[i+1] == '"' && bytes[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
			} else {
				i++
				for i < n {
					if bytes[i] == '\\' && i+1 < n {
						i += 2
						continue
					}
					if bytes[i] == '"' {
						i++
						break
					}
					i++
				}
			}
			continue
		}

		i++
	}
	return string(out)
}

func stripCommentsAndStrings(content string) string {
	bytes := []byte(content)
	n := len(bytes)
	out := make([]byte, n)
	copy(out, bytes)

	i := 0
	for i < n {
		// Block comment
		if i+1 < n && bytes[i] == '/' && bytes[i+1] == '*' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < n {
				if i+1 < n && bytes[i] == '*' && bytes[i+1] == '/' {
					out[i] = ' '
					out[i+1] = ' '
					i += 2
					break
				}
				if bytes[i] != '\n' && bytes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}

		// Line comment
		if i+1 < n && bytes[i] == '/' && bytes[i+1] == '/' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < n {
				if bytes[i] == '\n' || bytes[i] == '\r' {
					break
				}
				out[i] = ' '
				i++
			}
			continue
		}

		// Triple quoted string
		if i+2 < n && bytes[i] == '"' && bytes[i+1] == '"' && bytes[i+2] == '"' {
			out[i] = ' '
			out[i+1] = ' '
			out[i+2] = ' '
			i += 3
			for i < n {
				if i+2 < n && bytes[i] == '"' && bytes[i+1] == '"' && bytes[i+2] == '"' {
					out[i] = ' '
					out[i+1] = ' '
					out[i+2] = ' '
					i += 3
					break
				}
				if bytes[i] != '\n' && bytes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}

		// Regular string
		if bytes[i] == '"' {
			out[i] = ' '
			i++
			for i < n {
				if bytes[i] == '\\' && i+1 < n {
					if bytes[i+1] != '\n' && bytes[i+1] != '\r' {
						out[i] = ' '
						out[i+1] = ' '
					}
					i += 2
					continue
				}
				if bytes[i] == '"' {
					out[i] = ' '
					i++
					break
				}
				if bytes[i] != '\n' && bytes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}

		// Char literal
		if bytes[i] == '\'' {
			out[i] = ' '
			i++
			for i < n {
				if bytes[i] == '\\' && i+1 < n {
					if bytes[i+1] != '\n' && bytes[i+1] != '\r' {
						out[i] = ' '
						out[i+1] = ' '
					}
					i += 2
					continue
				}
				if bytes[i] == '\'' {
					out[i] = ' '
					i++
					break
				}
				if bytes[i] != '\n' && bytes[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			continue
		}

		i++
	}
	return string(out)
}

// ----------------------------------------------------
// Argument Extractor
// ----------------------------------------------------

func extractArguments(content string, startIdx int) (string, int) {
	n := len(content)
	if startIdx >= n {
		return "", -1
	}

	openParenIdx := -1
	for i := startIdx; i < n; i++ {
		if content[i] == '(' {
			openParenIdx = i
			break
		}
		if content[i] == '\n' || content[i] == '\r' {
			return "", -1
		}
		if content[i] != ' ' && content[i] != '\t' {
			return "", -1
		}
	}
	if openParenIdx == -1 {
		return "", -1
	}

	parenCount := 0
	braceCount := 0
	bracketCount := 0
	inString := false
	inTripleString := false

	for i := openParenIdx; i < n; i++ {
		r := content[i]

		if inTripleString {
			if r == '"' && i+2 < n && content[i+1] == '"' && content[i+2] == '"' {
				inTripleString = false
				i += 2
			}
			continue
		}
		if inString {
			if r == '\\' && i+1 < n {
				i++
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		if r == '"' {
			if i+2 < n && content[i+1] == '"' && content[i+2] == '"' {
				inTripleString = true
				i += 2
			} else {
				inString = true
			}
			continue
		}

		switch r {
		case '(':
			parenCount++
		case ')':
			parenCount--
			if parenCount == 0 && braceCount == 0 && bracketCount == 0 {
				return content[openParenIdx+1 : i], i
			}
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
	}

	return "", -1
}

// getLineAndCol returns 1-based line and column of the character at index
func getLineAndCol(content string, byteIndex int) (int, int) {
	line := 1
	col := 1
	for i := 0; i < byteIndex && i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		if r == '\n' {
			line++
			col = 1
		} else if r != '\r' {
			col++
		}
		i += size
	}
	return line, col
}

// containsWord checks if word is present in content as a whole word
func containsWord(content, word string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(content)
}

// ----------------------------------------------------
// Detector Implementations
// ----------------------------------------------------

// 1. ComposeMissingKeyDetector
type ComposeMissingKeyDetector struct {
	rule types.Rule
}

func (d *ComposeMissingKeyDetector) ID() string { return d.rule.ID }

func (d *ComposeMissingKeyDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding

	// We search in commentAndStringStripped to avoid matching commented items or items in string literals
	matches := itemsKeywordsRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		startIdx := m[0]
		keyword := commentAndStringStripped[m[0]:m[1]]

		// Verify if it is preceded by "fun" or "import" (avoid false positives)
		precededBy := ""
		if startIdx > 7 {
			precededBy = commentAndStringStripped[startIdx-7 : startIdx]
		}
		if strings.Contains(precededBy, "fun ") || strings.Contains(precededBy, "import ") {
			continue
		}

		args, endIdx := extractArguments(commentAndStringStripped, startIdx+len(keyword))
		if endIdx == -1 {
			// Not a function call
			continue
		}

		// Check if it specifies "key"
		if !containsWord(args, "key") {
			line, col := getLineAndCol(original, startIdx)
			findings = append(findings, types.Finding{
				ID:       d.rule.ID,
				Cluster:  d.rule.Cluster,
				Rule:     d.rule.ID,
				Severity: d.rule.Severity,
				File:     filePath,
				Line:     line,
				Column:   col,
				Message:  "compose-missing-key: items() or itemsIndexed() inside lazy lists should specify a key to avoid unnecessary recompositions.",
				FixHint:  d.rule.FixHint,
				DocURL:   d.rule.DocURL,
			})
		}
	}

	return findings
}

// 2. SecLogPiiDetector
type SecLogPiiDetector struct {
	rule types.Rule
}

func (d *SecLogPiiDetector) ID() string { return d.rule.ID }

func (d *SecLogPiiDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding

	// We check on commentStripped because string literals need to be preserved for PII scanning
	matches := logKeywordsRegex.FindAllStringIndex(commentStripped, -1)
	for _, m := range matches {
		startIdx := m[0]
		keyword := commentStripped[m[0]:m[1]]

		args, endIdx := extractArguments(commentStripped, startIdx+len(keyword))
		if endIdx == -1 {
			continue
		}

		if piiRegex.MatchString(args) {
			line, col := getLineAndCol(original, startIdx)
			findings = append(findings, types.Finding{
				ID:       d.rule.ID,
				Cluster:  d.rule.Cluster,
				Rule:     d.rule.ID,
				Severity: d.rule.Severity,
				File:     filePath,
				Line:     line,
				Column:   col,
				Message:  "sec-log-pii: Do not log PII (Personal Identifiable Information) such as emails, passwords, tokens, or phone numbers.",
				FixHint:  d.rule.FixHint,
				DocURL:   d.rule.DocURL,
			})
		}
	}

	return findings
}

// 3. SecWebViewJavascriptEnabledDetector
type SecWebViewJavascriptEnabledDetector struct {
	rule types.Rule
}

func (d *SecWebViewJavascriptEnabledDetector) ID() string { return d.rule.ID }

func (d *SecWebViewJavascriptEnabledDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding

	// Check on commentAndStringStripped
	matches := webViewRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		startIdx := m[0]
		line, col := getLineAndCol(original, startIdx)
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  "sec-webview-javascript-enabled: WebView JavaScript execution should not be enabled unless necessary and properly sanitised.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}

	return findings
}

// 4. CoroutineDispatchersHardcodedDetector
type CoroutineDispatchersHardcodedDetector struct {
	rule types.Rule
}

func (d *CoroutineDispatchersHardcodedDetector) ID() string { return d.rule.ID }

func (d *CoroutineDispatchersHardcodedDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding

	// Check on commentAndStringStripped
	matches := dispatchersRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		startIdx := m[0]
		line, col := getLineAndCol(original, startIdx)
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("coroutine-dispatchers-hardcoded: Avoid hardcoding %s. Use dependency injection to inject Dispatchers.", commentAndStringStripped[m[0]:m[1]]),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}

	return findings
}
