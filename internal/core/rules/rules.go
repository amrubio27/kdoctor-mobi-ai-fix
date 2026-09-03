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
	piiRegex              = regexp.MustCompile(`(?i)(password|email|token|secret|phone|ssn|credential|pin|creditcard|@)`)
	webViewRegex          = regexp.MustCompile(`\bjavaScriptEnabled\s*=\s*true\b`)
	dispatchersRegex      = regexp.MustCompile(`\bDispatchers\.(IO|Default|Unconfined)\b`)
	itemsKeywordsRegex    = regexp.MustCompile(`\b(items|itemsIndexed)\b`)
	logKeywordsRegex      = regexp.MustCompile(`\b(Log|Timber)\.[diwewtfv]\b|\bprint(ln)?\b`)
	presentationPathRegex = regexp.MustCompile(`(?i)[/\\](presentation|ui|view|composable)[/\\]|ViewModel\.kt$|Screen\.kt$`)
	dataImportRegex       = regexp.MustCompile(`(?m)^\s*import\s+[\w\.]+\.data\..*`)
	dataImplRefRegex      = regexp.MustCompile(`\b([A-Z]\w*(?:RepositoryImpl|DataSource|Dao|Room|Retrofit|(?:[A-Za-z0-9_]+Api)))\b`)
	useCasePathRegex      = regexp.MustCompile(`(?i)[/\\]usecase[/\\]|UseCase\.kt$`)
	viewModelClassRegex   = regexp.MustCompile(`class\s+([A-Za-z0-9_]+ViewModel)\b`)
	useCaseClassRegex     = regexp.MustCompile(`class\s+([A-Za-z0-9_]+UseCase)\b`)
	textLiteralRegex      = regexp.MustCompile(`\bText\s*\(\s*"([^"]+)"`)
	directInstRegex       = regexp.MustCompile(`\bval\s+\w+\s*=\s*([A-Z]\w*(?:RepositoryImpl|DataSource|Dao|Retrofit|HttpClient|Database))\s*\(`)
	mutableStateFlowRegex = regexp.MustCompile(`\bMutableStateFlow\b|\bmutableStateOf\b`)
	dtoRefRegex           = regexp.MustCompile(`\b([A-Z]\w*(?:Response|Entity|DTO))\b`)
	rawCatchRegex         = regexp.MustCompile(`catch\s*\(\s*\w+\s*:\s*(?:Exception|Throwable)\s*\)\s*\{`)
	sqlOrHttpRegex        = regexp.MustCompile(`\b(SELECT|INSERT|UPDATE|DELETE|Retrofit|HttpClient|HttpResponse)\b`)
	composableAnnoRegex   = regexp.MustCompile(`@Composable`)
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
		case "arch-presentation-depends-on-data":
			activeDetectors = append(activeDetectors, &ArchPresentationDependsOnDataDetector{rule: r})
		case "arch-viewmodel-contract":
			activeDetectors = append(activeDetectors, &ArchViewModelContractDetector{rule: r})
		case "arch-usecase-contract":
			activeDetectors = append(activeDetectors, &ArchUseCaseContractDetector{rule: r})
		case "arch-usecase-multiple-public-methods":
			activeDetectors = append(activeDetectors, &ArchUseCaseMultiplePublicMethodsDetector{rule: r})
		case "arch-misplaced-domain-logic":
			activeDetectors = append(activeDetectors, &ArchMisplacedDomainLogicDetector{rule: r})
		case "arch-misplaced-data-logic":
			activeDetectors = append(activeDetectors, &ArchMisplacedDataLogicDetector{rule: r})
		case "arch-model-mapping-leak":
			activeDetectors = append(activeDetectors, &ArchModelMappingLeakDetector{rule: r})
		case "error-handling-layer-mapping":
			activeDetectors = append(activeDetectors, &ErrorHandlingLayerMappingDetector{rule: r})
		case "arch-viewmodel-mvi-suggestion":
			activeDetectors = append(activeDetectors, &ArchViewModelMviSuggestionDetector{rule: r})
		case "compose-heavy-composable":
			activeDetectors = append(activeDetectors, &ComposeHeavyComposableDetector{rule: r})
		case "compose-graphics-layer":
			activeDetectors = append(activeDetectors, &ComposeGraphicsLayerDetector{rule: r})
		case "compose-recomposition-optimizer":
			activeDetectors = append(activeDetectors, &ComposeRecompositionOptimizerDetector{rule: r})
		case "ui-hardcoded-strings":
			activeDetectors = append(activeDetectors, &UIHardcodedStringsDetector{rule: r})
		case "testability-direct-instantiation":
			activeDetectors = append(activeDetectors, &TestabilityDirectInstantiationDetector{rule: r})
		case "arch-udf-sealed-events":
			activeDetectors = append(activeDetectors, &ArchUdfSealedEventsDetector{rule: r})
		case "arch-repository-impl-interface":
			activeDetectors = append(activeDetectors, &ArchRepositoryImplContractDetector{rule: r})
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

// 5. ArchPresentationDependsOnDataDetector
type ArchPresentationDependsOnDataDetector struct {
	rule types.Rule
}

func (d *ArchPresentationDependsOnDataDetector) ID() string { return d.rule.ID }

func (d *ArchPresentationDependsOnDataDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	isPresentation := presentationPathRegex.MatchString(filePath) || composableAnnoRegex.MatchString(commentAndStringStripped) || viewModelClassRegex.MatchString(commentAndStringStripped)
	if !isPresentation {
		return findings
	}

	if matches := dataImportRegex.FindAllStringIndex(commentAndStringStripped, -1); len(matches) > 0 {
		for _, m := range matches {
			line, col := getLineAndCol(original, m[0])
			findings = append(findings, types.Finding{
				ID:       d.rule.ID,
				Cluster:  d.rule.Cluster,
				Rule:     d.rule.ID,
				Severity: d.rule.Severity,
				File:     filePath,
				Line:     line,
				Column:   col,
				Message:  "arch-presentation-depends-on-data: Presentation layer must not import data layer packages. Use domain UseCases or Repository interfaces.",
				FixHint:  d.rule.FixHint,
				DocURL:   d.rule.DocURL,
			})
		}
	}

	matches := dataImplRefRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		// Excluir anotaciones y APIs experimentales estándar de Kotlin / Jetpack Compose / CMP
		if strings.HasPrefix(matchStr, "Experimental") || matchStr == "Api" {
			continue
		}
		prefix := ""
		if m[0] >= 10 {
			prefix = commentAndStringStripped[m[0]-10 : m[0]]
		} else {
			prefix = commentAndStringStripped[:m[0]]
		}
		if strings.Contains(prefix, "@") || strings.Contains(prefix, "OptIn") {
			continue
		}
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-presentation-depends-on-data: Presentation layer directly references data implementation '%s'. Pass UseCases or Repository interfaces instead.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 6. ArchViewModelContractDetector
type ArchViewModelContractDetector struct {
	rule types.Rule
}

func (d *ArchViewModelContractDetector) ID() string { return d.rule.ID }

func (d *ArchViewModelContractDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !strings.HasSuffix(filePath, "ViewModel.kt") && !viewModelClassRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	matches := dataImplRefRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		if strings.HasPrefix(matchStr, "Experimental") || matchStr == "Api" {
			continue
		}
		prefix := ""
		if m[0] >= 10 {
			prefix = commentAndStringStripped[m[0]-10 : m[0]]
		} else {
			prefix = commentAndStringStripped[:m[0]]
		}
		if strings.Contains(prefix, "@") || strings.Contains(prefix, "OptIn") {
			continue
		}
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-viewmodel-contract: ViewModel should receive UseCases rather than concrete '%s'. Repositories (interfaces) are allowed ONLY for passthrough UseCases without extra business logic.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 7. ArchUseCaseContractDetector
type ArchUseCaseContractDetector struct {
	rule types.Rule
}

func (d *ArchUseCaseContractDetector) ID() string { return d.rule.ID }

func (d *ArchUseCaseContractDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !useCasePathRegex.MatchString(filePath) && !useCaseClassRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	matches := dataImplRefRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		if strings.HasPrefix(matchStr, "Experimental") || matchStr == "Api" {
			continue
		}
		prefix := ""
		if m[0] >= 10 {
			prefix = commentAndStringStripped[m[0]-10 : m[0]]
		} else {
			prefix = commentAndStringStripped[:m[0]]
		}
		if strings.Contains(prefix, "@") || strings.Contains(prefix, "OptIn") {
			continue
		}
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-usecase-contract: UseCase should depend only on domain Repository interfaces, not data implementation '%s'.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 8. ArchUseCaseMultiplePublicMethodsDetector
type ArchUseCaseMultiplePublicMethodsDetector struct {
	rule types.Rule
}

func (d *ArchUseCaseMultiplePublicMethodsDetector) ID() string { return d.rule.ID }

func (d *ArchUseCaseMultiplePublicMethodsDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !useCasePathRegex.MatchString(filePath) && !useCaseClassRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	funRegex := regexp.MustCompile(`\bfun\s+([A-Za-z0-9_]+)\s*\(`)
	matches := funRegex.FindAllStringSubmatchIndex(commentAndStringStripped, -1)
	publicFuns := 0
	var firstExtraLine, firstExtraCol int
	for _, m := range matches {
		funName := commentAndStringStripped[m[2]:m[3]]
		preText := ""
		if m[0] >= 10 {
			preText = commentAndStringStripped[m[0]-10 : m[0]]
		}
		if strings.Contains(preText, "private") || strings.Contains(preText, "protected") {
			continue
		}
		if funName == "invoke" || funName == "execute" {
			continue
		}
		publicFuns++
		if publicFuns == 1 {
			firstExtraLine, firstExtraCol = getLineAndCol(original, m[0])
		}
	}

	if publicFuns > 0 {
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     firstExtraLine,
			Column:   firstExtraCol,
			Message:  "arch-usecase-multiple-public-methods: UseCases should expose a single public method (operator fun invoke or execute) for Single Responsibility Principle.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 9. ArchMisplacedDomainLogicDetector
type ArchMisplacedDomainLogicDetector struct {
	rule types.Rule
}

func (d *ArchMisplacedDomainLogicDetector) ID() string { return d.rule.ID }

func (d *ArchMisplacedDomainLogicDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	isViewModelOrUI := strings.HasSuffix(filePath, "ViewModel.kt") || viewModelClassRegex.MatchString(commentAndStringStripped) || composableAnnoRegex.MatchString(commentAndStringStripped)
	if !isViewModelOrUI {
		return findings
	}

	bizRegex := regexp.MustCompile(`\bfun\s+(calculate[A-Z]\w*|validate[A-Z]\w*|compute[A-Z]\w*|applyTax|processOrder)\b`)
	matches := bizRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		line, col := getLineAndCol(original, m[0])
		matchStr := commentAndStringStripped[m[0]:m[1]]
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-misplaced-domain-logic: Business logic function '%s' should be placed in a UseCase instead of ViewModel/UI.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 10. ArchMisplacedDataLogicDetector
type ArchMisplacedDataLogicDetector struct {
	rule types.Rule
}

func (d *ArchMisplacedDataLogicDetector) ID() string { return d.rule.ID }

func (d *ArchMisplacedDataLogicDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	isUseCaseOrViewModel := useCasePathRegex.MatchString(filePath) || useCaseClassRegex.MatchString(commentAndStringStripped) || strings.HasSuffix(filePath, "ViewModel.kt")
	if !isUseCaseOrViewModel {
		return findings
	}

	matches := sqlOrHttpRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-misplaced-data-logic: Low-level data access / HTTP logic '%s' should be in RepositoryImpl or DataSource, not in UseCase/ViewModel.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 11. ArchModelMappingLeakDetector
type ArchModelMappingLeakDetector struct {
	rule types.Rule
}

func (d *ArchModelMappingLeakDetector) ID() string { return d.rule.ID }

func (d *ArchModelMappingLeakDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	isPresentation := presentationPathRegex.MatchString(filePath) || composableAnnoRegex.MatchString(commentAndStringStripped) || viewModelClassRegex.MatchString(commentAndStringStripped)
	if !isPresentation {
		return findings
	}

	matches := dtoRefRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-model-mapping-leak: Data DTO '%s' leaked to Presentation layer. Map Data -> Domain -> UiModel.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 12. ErrorHandlingLayerMappingDetector
type ErrorHandlingLayerMappingDetector struct {
	rule types.Rule
}

func (d *ErrorHandlingLayerMappingDetector) ID() string { return d.rule.ID }

func (d *ErrorHandlingLayerMappingDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	matches := rawCatchRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		// Encontrar el cuerpo delimitado por llaves tras el match de catch(...) {
		braceIdx := strings.Index(commentAndStringStripped[m[0]:], "{")
		if braceIdx != -1 {
			startBrace := m[0] + braceIdx
			depth := 0
			endBrace := -1
			for i := startBrace; i < len(commentAndStringStripped); i++ {
				if commentAndStringStripped[i] == '{' {
					depth++
				} else if commentAndStringStripped[i] == '}' {
					depth--
					if depth == 0 {
						endBrace = i
						break
					}
				}
			}
			if endBrace != -1 {
				catchBody := commentAndStringStripped[startBrace+1 : endBrace]
				// Si el cuerpo mapea a un Result, AppError, Either, Domain exception o hace rethrow, es válido
				if strings.Contains(catchBody, "Result.") ||
					strings.Contains(catchBody, "AppError") ||
					strings.Contains(catchBody, "Either") ||
					strings.Contains(catchBody, "Error") ||
					strings.Contains(catchBody, "throw") ||
					strings.Contains(catchBody, "emit(") ||
					strings.Contains(catchBody, "Failure(") {
					continue
				}
			}
		}

		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  "error-handling-layer-mapping: Avoid catching raw generic Exception without mapping to Domain Result/Exception for presentation error state.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 13. ArchViewModelMviSuggestionDetector
type ArchViewModelMviSuggestionDetector struct {
	rule types.Rule
}

func (d *ArchViewModelMviSuggestionDetector) ID() string { return d.rule.ID }

func (d *ArchViewModelMviSuggestionDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !strings.HasSuffix(filePath, "ViewModel.kt") && !viewModelClassRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	matches := mutableStateFlowRegex.FindAllStringIndex(commentAndStringStripped, -1)
	if len(matches) >= 3 {
		line, col := getLineAndCol(original, matches[0][0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("arch-viewmodel-mvi-suggestion: ViewModel manages %d separate state streams. Consider MVI architecture (_uiState.update { ... } with a unified UiState data class).", len(matches)),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 14. ComposeHeavyComposableDetector
type ComposeHeavyComposableDetector struct {
	rule types.Rule
}

func (d *ComposeHeavyComposableDetector) ID() string { return d.rule.ID }

func (d *ComposeHeavyComposableDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !composableAnnoRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	lines := strings.Split(original, "\n")
	if len(lines) > 80 && composableAnnoRegex.MatchString(commentAndStringStripped) {
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     1,
			Column:   1,
			Message:  fmt.Sprintf("compose-heavy-composable: Composable file has %d lines. Break down large UI components into smaller modular composables.", len(lines)),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 15. ComposeGraphicsLayerDetector
type ComposeGraphicsLayerDetector struct {
	rule types.Rule
}

func (d *ComposeGraphicsLayerDetector) ID() string { return d.rule.ID }

func (d *ComposeGraphicsLayerDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !composableAnnoRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	animRegex := regexp.MustCompile(`Modifier\.(?:alpha|translationX|translationY|rotationZ)\s*\(`)
	matches := animRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  "compose-graphics-layer: Use graphicsLayer { ... } for animation state properties to skip composition and layout phases during updates.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 16. ComposeRecompositionOptimizerDetector
type ComposeRecompositionOptimizerDetector struct {
	rule types.Rule
}

func (d *ComposeRecompositionOptimizerDetector) ID() string { return d.rule.ID }

func (d *ComposeRecompositionOptimizerDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !composableAnnoRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	unstableColRegex := regexp.MustCompile(`@Composable\s+fun\s+\w+\s*\([^)]*:\s*(?:List|Set|Map)<`)
	matches := unstableColRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  "compose-recomposition-optimizer: Standard List/Set/Map parameters trigger recomposition under K2 compiler. Use ImmutableList or annotate UI state with @Immutable.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 17. UIHardcodedStringsDetector
type UIHardcodedStringsDetector struct {
	rule types.Rule
}

func (d *UIHardcodedStringsDetector) ID() string { return d.rule.ID }

func (d *UIHardcodedStringsDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !composableAnnoRegex.MatchString(commentAndStringStripped) || strings.Contains(commentAndStringStripped, "@Preview") {
		return findings
	}

	matches := textLiteralRegex.FindAllStringIndex(original, -1)
	for _, m := range matches {
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  "ui-hardcoded-strings: Hardcoded UI string in Text(...). Extract to stringResource(R.string...) for localization and reuse.",
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 18. TestabilityDirectInstantiationDetector
type TestabilityDirectInstantiationDetector struct {
	rule types.Rule
}

func (d *TestabilityDirectInstantiationDetector) ID() string { return d.rule.ID }

func (d *TestabilityDirectInstantiationDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	isTarget := strings.HasSuffix(filePath, "ViewModel.kt") || useCasePathRegex.MatchString(filePath) || strings.HasSuffix(filePath, "RepositoryImpl.kt")
	if !isTarget {
		return findings
	}

	matches := directInstRegex.FindAllStringIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		matchStr := commentAndStringStripped[m[0]:m[1]]
		line, col := getLineAndCol(original, m[0])
		findings = append(findings, types.Finding{
			ID:       d.rule.ID,
			Cluster:  d.rule.Cluster,
			Rule:     d.rule.ID,
			Severity: d.rule.Severity,
			File:     filePath,
			Line:     line,
			Column:   col,
			Message:  fmt.Sprintf("testability-direct-instantiation: Direct instantiation '%s' breaks unit testing and Dependency Inversion. Inject dependencies through constructor.", matchStr),
			FixHint:  d.rule.FixHint,
			DocURL:   d.rule.DocURL,
		})
	}
	return findings
}

// 19. ArchUdfSealedEventsDetector
type ArchUdfSealedEventsDetector struct {
	rule types.Rule
}

func (d *ArchUdfSealedEventsDetector) ID() string { return d.rule.ID }

func (d *ArchUdfSealedEventsDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !strings.HasSuffix(filePath, "ViewModel.kt") && !viewModelClassRegex.MatchString(commentAndStringStripped) {
		return findings
	}

	if !strings.Contains(commentAndStringStripped, "UiEvent") && !strings.Contains(commentAndStringStripped, "UiAction") {
		// Detectar tanto acumulación de on* como mutadores directos individuales (onXChanged, setX, updateX)
		mutatorRegex := regexp.MustCompile(`\bfun\s+(?:on[A-Z]\w*Changed|set[A-Z]\w*|update[A-Z]\w*)\s*\(`)
		if m := mutatorRegex.FindStringIndex(commentAndStringStripped); m != nil {
			line, col := getLineAndCol(original, m[0])
			findings = append(findings, types.Finding{
				ID:       d.rule.ID,
				Cluster:  d.rule.Cluster,
				Rule:     d.rule.ID,
				Severity: d.rule.Severity,
				File:     filePath,
				Line:     line,
				Column:   col,
				Message:  "arch-udf-sealed-events: ViewModel exposes public mutator methods. Use sealed interface UiEvent to enforce Unidirectional Data Flow (UDF/MVI).",
				FixHint:  d.rule.FixHint,
				DocURL:   d.rule.DocURL,
			})
		} else {
			onMethodsRegex := regexp.MustCompile(`\bfun\s+on[A-Z]\w*`)
			matches := onMethodsRegex.FindAllStringIndex(commentAndStringStripped, -1)
			if len(matches) >= 3 {
				line, col := getLineAndCol(original, matches[0][0])
				findings = append(findings, types.Finding{
					ID:       d.rule.ID,
					Cluster:  d.rule.Cluster,
					Rule:     d.rule.ID,
					Severity: d.rule.Severity,
					File:     filePath,
					Line:     line,
					Column:   col,
					Message:  fmt.Sprintf("arch-udf-sealed-events: ViewModel exposes %d separate on* event methods. Consider a sealed interface UiEvent to enforce Unidirectional Data Flow (UDF).", len(matches)),
					FixHint:  d.rule.FixHint,
					DocURL:   d.rule.DocURL,
				})
			}
		}
	}
	return findings
}

// 20. ArchRepositoryImplContractDetector
type ArchRepositoryImplContractDetector struct {
	rule types.Rule
}

func (d *ArchRepositoryImplContractDetector) ID() string { return d.rule.ID }

func (d *ArchRepositoryImplContractDetector) Check(filePath string, original string, commentStripped string, commentAndStringStripped string) []types.Finding {
	var findings []types.Finding
	if !strings.HasSuffix(filePath, "RepositoryImpl.kt") && !strings.Contains(commentAndStringStripped, "RepositoryImpl") {
		return findings
	}

	classRegex := regexp.MustCompile(`class\s+([A-Za-z0-9_]+RepositoryImpl)(?:<[^>]+>)?\s*(?:\([^)]*\))?\s*(?::\s*([^{]+))?\{?`)
	matches := classRegex.FindAllStringSubmatchIndex(commentAndStringStripped, -1)
	for _, m := range matches {
		className := commentAndStringStripped[m[2]:m[3]]
		hasSuperType := false
		if m[4] != -1 && m[5] != -1 {
			supertypes := commentAndStringStripped[m[4]:m[5]]
			if strings.Contains(supertypes, "Repository") {
				hasSuperType = true
			}
		}
		if !hasSuperType {
			line, col := getLineAndCol(original, m[0])
			findings = append(findings, types.Finding{
				ID:       d.rule.ID,
				Cluster:  d.rule.Cluster,
				Rule:     d.rule.ID,
				Severity: d.rule.Severity,
				File:     filePath,
				Line:     line,
				Column:   col,
				Message:  fmt.Sprintf("arch-repository-impl-interface: Class %s must implement a domain Repository interface (DIP contract violation).", className),
				FixHint:  d.rule.FixHint,
				DocURL:   d.rule.DocURL,
			})
		}
	}
	return findings
}
