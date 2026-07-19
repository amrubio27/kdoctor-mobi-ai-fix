// Package genschema vuelca el catálogo de 78 reglas a rules/metadata.json.
//
// Uso: go run ./scripts/genschema
//
// CatalogRules es la fuente canónica de las 64 reglas V1 más 14 default
// detekt mappings añadidas en Phase 1.5 (78 total). La exportación a JSON
// se hace en main(); el JSON vive en rules/metadata.json que es lo que
// consumen los binarios. El test TestCatalogConvergence en main_test.go
// verifica que el JSON en disco coincide con CatalogRules.
//
// Convenciones de Naming (Phase 1.5 v3):
//   - ID = `{cluster-prefix}-{descriptor}`. V1 usa abreviaciones (mem-*,
//     arch-*, a11y-*, sec-*, test-*, dead-*). 5.11 usa nombres completos
//     (complexity-*, error-handling-*, magic-numbers-*, naming-*,
//     formatting-*). El test TestClusterTaxonomyIsConsistent valida esta
//     convención contra todas las reglas (no solo 5.11).
//   - FixHint = string accionable. V1 las tiene casi todas; en Phase 1.5 v3
//     añadimos FixHints también para 5.11 para que el reporte JSON incluya
//     remediation advice.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Rule struct {
	ID         string `json:"id"`
	Cluster    string `json:"cluster"`
	Severity   string `json:"severity"`
	DetektRule string `json:"detektRule,omitempty"`
	Status     string `json:"status"`
	FixHint    string `json:"fixHint,omitempty"`
}

// CatalogRules es el catálogo canónico. Si añades una regla aquí, ejecuta:
//
//	go run ./scripts/genschema -out rules/metadata.json
var CatalogRules = []Rule{
	// 5.1 Compose Performance (12)
	{ID: "compose-missing-key", Cluster: "compose-performance", Severity: "error", Status: "planned"},
	{ID: "compose-unstable-params", Cluster: "compose-performance", Severity: "error", Status: "planned"},
	{ID: "compose-derived-state-missing", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-lambda-recomposition", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-heavy-composable", Cluster: "compose-performance", Severity: "info", Status: "planned"},
	{ID: "compose-remember-missing", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:ReusedModifierInstance", Status: "live", FixHint: "Wrap mutable state in remember { mutableStateOf(...) }."},
	{ID: "compose-state-hoisting", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierHeightWithText", Status: "live", FixHint: "Move state up and receive callbacks down."},
	{ID: "compose-modifier-frequent-changes", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ReusedModifierInstance", Status: "live", FixHint: "Hoist the Modifier to a parameter or remember it."},
	{ID: "compose-graphics-layer", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-list-animated", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-side-effect-in-compose", Cluster: "compose-performance", Severity: "error", Status: "planned"},
	{ID: "compose-runtime-import-bleeding", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:ComposableNaming", Status: "live", FixHint: "Don't import compose.runtime.* outside @Composable functions."},
	// 5.2 Coroutines & Async (8)
	{ID: "coroutine-viewmodel-scope", Cluster: "coroutines", Severity: "error", Status: "planned"},
	{ID: "coroutine-global-scope", Cluster: "coroutines", Severity: "error", DetektRule: "GlobalCoroutineUsage", Status: "live", FixHint: "Use injected CoroutineScope (e.g., viewModelScope)."},
	{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: "info", Status: "planned"},
	{ID: "coroutine-supervisor-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-unstructured-concurrency", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-cancellation-leak", Cluster: "coroutines", Severity: "error", DetektRule: "CoroutineCancellation", Status: "live", FixHint: "Don't swallow CancellationException in runCatching; rethrow."},
	{ID: "coroutine-flow-buffer-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-sharedflow-replay", Cluster: "coroutines", Severity: "info", Status: "planned"},
	// 5.3 Lifecycle (6)
	{ID: "lifecycle-context-leak", Cluster: "lifecycle", Severity: "error", Status: "planned"},
	{ID: "lifecycle-collect-as-state-missing", Cluster: "lifecycle", Severity: "error", Status: "planned"},
	{ID: "lifecycle-collect-lifecycle-aware", Cluster: "lifecycle", Severity: "warning", Status: "planned"},
	{ID: "lifecycle-ondestroy-listener", Cluster: "lifecycle", Severity: "warning", Status: "planned"},
	{ID: "lifecycle-job-not-cancelled", Cluster: "lifecycle", Severity: "error", Status: "planned"},
	{ID: "lifecycle-config-change-survival", Cluster: "lifecycle", Severity: "info", Status: "planned"},
	// 5.4 Memory (5)
	{ID: "mem-bitmap-no-pool", Cluster: "memory", Severity: "warning", Status: "planned"},
	{ID: "mem-context-receiver-leak", Cluster: "memory", Severity: "error", Status: "planned"},
	{ID: "mem-static-context", Cluster: "memory", Severity: "error", Status: "planned"},
	{ID: "mem-handler-leak", Cluster: "memory", Severity: "warning", Status: "planned"},
	{ID: "mem-coroutine-job-leak", Cluster: "memory", Severity: "error", Status: "planned"},
	// 5.5 Architecture (10)
	{ID: "arch-god-class", Cluster: "architecture", Severity: "warning", DetektRule: "TooManyFunctions", Status: "live", FixHint: "Split class by responsibility."},
	{ID: "arch-circular-dep", Cluster: "architecture", Severity: "error", Status: "planned"},
	{ID: "arch-feature-module-public-api-bleed", Cluster: "architecture", Severity: "warning", Status: "planned"},
	{ID: "arch-public-api-mutable-state", Cluster: "architecture", Severity: "error", Status: "planned"},
	{ID: "arch-data-class-with-logic", Cluster: "architecture", Severity: "warning", Status: "planned"},
	{ID: "arch-named-arg-required", Cluster: "architecture", Severity: "info", Status: "planned"},
	{ID: "arch-utility-function-should-be-extension", Cluster: "architecture", Severity: "info", Status: "planned"},
	{ID: "arch-internal-in-public-api", Cluster: "architecture", Severity: "error", DetektRule: "InvalidPackageDeclaration", Status: "live", FixHint: "Do not expose internal types in public API."},
	{ID: "arch-package-cycles-kmp", Cluster: "architecture", Severity: "error", Status: "planned"},
	{ID: "arch-presentation-depends-on-data", Cluster: "architecture", Severity: "error", Status: "planned"},
	// 5.6 Accessibility (5)
	{ID: "a11y-content-description", Cluster: "accessibility", Severity: "error", Status: "planned"},
	{ID: "a11y-click-target-size", Cluster: "accessibility", Severity: "warning", Status: "planned"},
	{ID: "a11y-merged-clickable", Cluster: "accessibility", Severity: "info", Status: "planned"},
	{ID: "a11y-talkback-label-missing", Cluster: "accessibility", Severity: "warning", Status: "planned"},
	{ID: "a11y-color-contrast-note", Cluster: "accessibility", Severity: "info", Status: "planned"},
	// 5.7 Testing (5)
	{ID: "test-public-api-without-test", Cluster: "testing", Severity: "warning", Status: "planned"},
	{ID: "test-flaky-test-marker", Cluster: "testing", Severity: "info", Status: "planned"},
	{ID: "test-hilt-rule-missing", Cluster: "testing", Severity: "error", Status: "planned"},
	{ID: "test-runblocking-in-test", Cluster: "testing", Severity: "warning", Status: "planned"},
	{ID: "test-compose-test-rule-missing", Cluster: "testing", Severity: "warning", Status: "planned"},
	// 5.8 Security (5)
	{ID: "sec-hardcoded-secret", Cluster: "security", Severity: "error", DetektRule: "HardcodedPassword", Status: "live", FixHint: "Move secret to BuildConfig or environment variable."},
	{ID: "sec-log-pii", Cluster: "security", Severity: "error", Status: "planned"},
	{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: "error", Status: "planned"},
	{ID: "sec-deeplink-no-validation", Cluster: "security", Severity: "warning", Status: "planned"},
	{ID: "sec-fragment-injection", Cluster: "security", Severity: "error", Status: "planned"},
	// 5.9 KMP / CMP (4)
	{ID: "kmp-platform-api-leak", Cluster: "kmp", Severity: "error", Status: "planned"},
	{ID: "kmp-expect-actual-violation", Cluster: "kmp", Severity: "error", Status: "planned"},
	{ID: "kmp-coroutines-supervisor-in-common", Cluster: "kmp", Severity: "warning", Status: "planned"},
	{ID: "kmp-compose-multiplatform-stable-required", Cluster: "kmp", Severity: "warning", Status: "planned"},
	// 5.10 Dead code (4)
	{ID: "dead-unused-import", Cluster: "dead-code", Severity: "info", DetektRule: "UnusedImport", Status: "live", FixHint: "Remove the import."},
	{ID: "dead-unused-private-fun", Cluster: "dead-code", Severity: "info", DetektRule: "UnusedPrivateMember", Status: "live", FixHint: "Remove unused private declaration."},
	{ID: "dead-unused-parameter", Cluster: "dead-code", Severity: "warning", Status: "planned"},
	{ID: "dead-white-label-export", Cluster: "dead-code", Severity: "info", Status: "planned"},

	// 5.11 Default detekt mappings (14) — Phase 1.5.
	// Mapeo de reglas estilo/naming/complexity que detekt-cli emite por
	// defecto, para que el scan no devuelva findings en cluster=[unknown].
	// IDs siguen convención `{cluster-prefix}-*` (validada por
	// TestClusterTaxonomyIsConsistent).
	// Nuevos clusters: complexity, error-handling, magic-numbers, naming,
	// formatting.
	// 5.11.1 complexity (4)
	{ID: "complexity-long-method", Cluster: "complexity", Severity: "warning", DetektRule: "LongMethod", Status: "live", FixHint: "Break the function into smaller helpers by responsibility."},
	{ID: "complexity-long-parameter-list", Cluster: "complexity", Severity: "warning", DetektRule: "LongParameterList", Status: "live", FixHint: "Introduce a parameter object (e.g., ConstructorParams data class) or builder pattern."},
	{ID: "complexity-large-class", Cluster: "complexity", Severity: "warning", DetektRule: "LargeClass", Status: "live", FixHint: "Split class by Single Responsibility Principle."},
	{ID: "complexity-cyclomatic-complex-method", Cluster: "complexity", Severity: "warning", DetektRule: "CyclomaticComplexMethod", Status: "live", FixHint: "Reduce branches via guard clauses, polymorphism or strategy pattern."},
	// 5.11.2 error-handling (2)
	{ID: "error-handling-generic-exception-caught", Cluster: "error-handling", Severity: "warning", DetektRule: "TooGenericExceptionCaught", Status: "live", FixHint: "Catch specific exceptions (e.g., IOException, TimeoutException) instead of Exception/Throwable."},
	{ID: "error-handling-throws-count", Cluster: "error-handling", Severity: "warning", DetektRule: "ThrowsCount", Status: "live", FixHint: "Reduce throws via Result/Either types or wrap into a domain exception."},
	// 5.11.3 magic-numbers (1)
	{ID: "magic-numbers-literal", Cluster: "magic-numbers", Severity: "info", DetektRule: "MagicNumber", Status: "live", FixHint: "Replace with a named constant (private const val MAX_RETRIES = 3)."},
	// 5.11.4 naming (5)
	{ID: "naming-function-convention", Cluster: "naming", Severity: "warning", DetektRule: "FunctionNaming", Status: "live", FixHint: "Composable: PascalCase. Helper functions: lowerCamelCase. Avoid snake_case."},
	{ID: "naming-class-convention", Cluster: "naming", Severity: "info", DetektRule: "ClassNaming", Status: "live", FixHint: "Class names must be PascalCase (e.g., UserRepository not userRepository)."},
	{ID: "naming-variable-convention", Cluster: "naming", Severity: "info", DetektRule: "VariableNaming", Status: "live", FixHint: "Variables must be lowerCamelCase (e.g., userName not UserName)."},
	{ID: "naming-constructor-parameter-convention", Cluster: "naming", Severity: "info", DetektRule: "ConstructorParameterNaming", Status: "live", FixHint: "Constructor parameters begin lowercase and may use underscores (e.g., private val id)."},
	{ID: "naming-matching-declaration-name", Cluster: "naming", Severity: "warning", DetektRule: "MatchingDeclarationName", Status: "live", FixHint: "File name must match the top-level declaration (e.g., UserRepository.kt contains class UserRepository)."},
	// 5.11.5 formatting (2)
	{ID: "formatting-newline-at-eof", Cluster: "formatting", Severity: "info", DetektRule: "NewLineAtEndOfFile", Status: "live", FixHint: "Add a trailing newline at EOF."},
	{ID: "formatting-max-line-length", Cluster: "formatting", Severity: "warning", DetektRule: "MaxLineLength", Status: "live", FixHint: "Break the line below 120 chars (default detekt threshold)."},
}

func main() {
	out := flag.String("out", "rules/metadata.json", "output path for the generated metadata.json")
	flag.Parse()

	if len(CatalogRules) != 78 {
		fmt.Fprintf(os.Stderr, "ERROR: expected 78 rules (64 V1 + 14 default detekt), got %d\n", len(CatalogRules))
		os.Exit(1)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", *out, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(CatalogRules); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Generated %d rules to %s\n", len(CatalogRules), *out)
}
