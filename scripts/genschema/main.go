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
	{ID: "compose-missing-key", Cluster: "compose-performance", Severity: "error", Status: "live", FixHint: "Define a unique key for each item in the list using the key parameter."},
	{ID: "compose-unstable-params", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:UnstableCollections", Status: "live", FixHint: "Annotate UI State data classes with @Immutable/@Stable or use ImmutableList to avoid unnecessary recompositions under K2 compiler."},
	{ID: "compose-derived-state-missing", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-lambda-recomposition", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-heavy-composable", Cluster: "compose-performance", Severity: "info", Status: "live", FixHint: "Modularize large Composable functions (>80 lines) into smaller reusable UI components."},
	{ID: "compose-remember-missing", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:RememberMissing", Status: "live", FixHint: "Wrap mutable state in remember { mutableStateOf(...) }."},
	{ID: "compose-state-hoisting", Cluster: "compose-performance", Severity: "warning", Status: "planned", FixHint: "Move state up and receive callbacks down."},
	{ID: "compose-modifier-frequent-changes", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierReused", Status: "live", FixHint: "Hoist the Modifier to a parameter or remember it."},
	{ID: "compose-graphics-layer", Cluster: "compose-performance", Severity: "warning", Status: "live", FixHint: "Use graphicsLayer { ... } or lambda-based Modifier parameters for frequently changing animation states to skip composition and layout phases."},
	{ID: "compose-list-animated", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
	{ID: "compose-side-effect-in-compose", Cluster: "compose-performance", Severity: "error", Status: "planned"},
	{ID: "compose-runtime-import-bleeding", Cluster: "compose-performance", Severity: "error", Status: "planned", FixHint: "Don't import compose.runtime.* outside @Composable functions."},
	// 5.2 Coroutines & Async (8)
	{ID: "coroutine-viewmodel-scope", Cluster: "coroutines", Severity: "error", Status: "planned"},
	{ID: "coroutine-global-scope", Cluster: "coroutines", Severity: "error", DetektRule: "GlobalCoroutineUsage", Status: "live", FixHint: "Use injected CoroutineScope (e.g., viewModelScope)."},
	{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: "info", Status: "live", FixHint: "Inject dispatchers through the constructor to allow overriding them in tests."},
	{ID: "coroutine-supervisor-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-unstructured-concurrency", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-cancellation-leak", Cluster: "coroutines", Severity: "error", DetektRule: "CoroutineCancellation", Status: "live", FixHint: "Don't swallow CancellationException in runCatching; rethrow."},
	{ID: "coroutine-flow-buffer-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
	{ID: "coroutine-sharedflow-replay", Cluster: "coroutines", Severity: "info", Status: "planned"},
	// 5.3 Lifecycle (6)
	{ID: "lifecycle-context-leak", Cluster: "lifecycle", Severity: "error", Status: "planned"},
	{ID: "lifecycle-collect-as-state-missing", Cluster: "lifecycle", Severity: "error", Status: "planned", FixHint: "Replace collectAsState() with collectAsStateWithLifecycle() or repeatOnLifecycle to prevent background state leaks."},
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
	{ID: "arch-public-api-mutable-state", Cluster: "architecture", Severity: "error", DetektRule: "Compose:MutableStateParam", Status: "live", FixHint: "Expose StateFlow/SharedFlow as read-only (asStateFlow()/asSharedFlow()) and use atomic _uiState.update { ... } in ViewModels."},
	{ID: "arch-data-class-with-logic", Cluster: "architecture", Severity: "warning", Status: "planned"},
	{ID: "arch-named-arg-required", Cluster: "architecture", Severity: "info", Status: "planned"},
	{ID: "arch-utility-function-should-be-extension", Cluster: "architecture", Severity: "info", Status: "planned"},
	{ID: "arch-internal-in-public-api", Cluster: "architecture", Severity: "error", DetektRule: "InvalidPackageDeclaration", Status: "live", FixHint: "Do not expose internal types in public API."},
	{ID: "arch-package-cycles-kmp", Cluster: "architecture", Severity: "error", Status: "planned"},
	{ID: "arch-presentation-depends-on-data", Cluster: "architecture", Severity: "error", Status: "live", FixHint: "Presentation layer (@Composable/ViewModel) must not depend on Data layer (data.*, DataSource, DAO, Api, RepositoryImpl). Access data through UseCases or Repository interfaces."},
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
	// sec-hardcoded-secret: status="live" pero NO smoke-tested en detekt-cli
	// 1.23.x (HardcodedPassword no existe en default-detekt-config.yml
	// bundled — verifiqué con grep en /tmp/default-detekt-config.yml).
	// Forward-compat: si detekt 2.x reintroduce/renombre la regla,
	// basta agregar el stanza en examples/bad-project/detekt.yml sin tocar
	// el catalog. El rulemap con prefix-strip ya tolera prefijos vendors.
	{ID: "sec-hardcoded-secret", Cluster: "security", Severity: "error", DetektRule: "HardcodedPassword", Status: "live", FixHint: "Move secret to BuildConfig or environment variable."},
	{ID: "sec-log-pii", Cluster: "security", Severity: "error", Status: "live", FixHint: "Do not log PII (emails, passwords, tokens, etc.). Remove or mask the logged data."},
	{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: "error", Status: "live", FixHint: "Do not enable JavaScript in WebView unless absolutely necessary and secure."},
	{ID: "sec-deeplink-no-validation", Cluster: "security", Severity: "warning", Status: "planned"},
	{ID: "sec-fragment-injection", Cluster: "security", Severity: "error", Status: "planned"},
	// 5.9 KMP / CMP (4)
	{ID: "kmp-platform-api-leak", Cluster: "kmp", Severity: "error", Status: "planned"},
	{ID: "kmp-expect-actual-violation", Cluster: "kmp", Severity: "error", Status: "planned"},
	{ID: "kmp-coroutines-supervisor-in-common", Cluster: "kmp", Severity: "warning", Status: "planned"},
	{ID: "kmp-compose-multiplatform-stable-required", Cluster: "kmp", Severity: "warning", Status: "planned"},
	// 5.10 Dead code (4)
	// dead-unused-import: detekt 1.23.x usa `UnusedImports` (plural) en
	// SARIF ruleId (`detekt.style.UnusedImports`) y tambien en config YAML.
	// El catalog historicamente tenía `UnusedImport` (singular) que fallaba
	// el lookup; corregido a plural para coincidir con el bundle bundled.
	{ID: "dead-unused-import", Cluster: "dead-code", Severity: "info", DetektRule: "UnusedImports", Status: "live", FixHint: "Remove the import."},
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
	// Phase 2 Expansion (12)
	{ID: "compose-derived-state-unremembered", Cluster: "compose-performance", Severity: "error", Status: "planned", FixHint: "Wrap derivedStateOf { ... } inside remember { ... } to prevent recalculation on every recomposition."},
	{ID: "compose-unstable-collection-params", Cluster: "compose-performance", Severity: "warning", Status: "planned", FixHint: "Replace standard Kotlin List/Set/Map with ImmutableList or annotate UI state with @Immutable."},
	{ID: "compose-launcheffect-unit-key", Cluster: "compose-performance", Severity: "warning", Status: "planned", FixHint: "Avoid LaunchedEffect(Unit) for dynamic data loads; bind the key to state or viewmodel events."},
	{ID: "compose-multiple-emitters-in-composable", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:MultipleEmitters", Status: "live", FixHint: "A Composable should emit only one main UI node tree to preserve layout hierarchy integrity."},
	{ID: "coroutine-naked-try-catch-in-flow", Cluster: "coroutines", Severity: "warning", Status: "planned", FixHint: "Replace try-catch around Flow operators with the idiomatic .catch { ... } operator."},
	{ID: "coroutine-suspend-fun-naming", Cluster: "coroutines", Severity: "info", DetektRule: "SuspendFunNaming", Status: "live", FixHint: "Suspend functions should be named clearly (e.g. fetchUserData) without Async suffix."},
	{ID: "coroutine-exception-handler-missing", Cluster: "coroutines", Severity: "warning", Status: "planned", FixHint: "CoroutineScope launches without SupervisorJob or CoroutineExceptionHandler can crash parent job."},
	{ID: "arch-usecase-multiple-public-methods", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "UseCase classes should have a single public operator fun invoke(...) or execute() method."},
	{ID: "kmp-expect-actual-mutable-state", Cluster: "kmp", Severity: "warning", Status: "planned", FixHint: "Expect declarations in commonMain should avoid exposing mutable platform-specific state directly."},
	{ID: "dead-commented-code", Cluster: "dead-code", Severity: "info", DetektRule: "CommentOverPrivateFunction", Status: "live", FixHint: "Remove commented-out code blocks to improve codebase readability."},
	{ID: "arch-viewmodel-contract", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "ViewModels must receive UseCases. Repositories (interfaces) are allowed ONLY for passthrough UseCases without extra business logic."},
	{ID: "arch-usecase-contract", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "UseCases must depend only on domain Repository interfaces and expose a single execution method (operator fun invoke / execute)."},
	{ID: "arch-misplaced-domain-logic", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "Move domain/business logic (complex calculations, business validations) from ViewModel or Composable into a UseCase."},
	{ID: "arch-misplaced-data-logic", Cluster: "architecture", Severity: "error", Status: "live", FixHint: "Move data access, SQL queries, or HTTP parsing logic from ViewModel/UseCase into RepositoryImpl or DataSource."},
	{ID: "arch-model-mapping-leak", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "Do not leak Data DTOs/Entities to Domain/UI. Use Mappers to transform Data -> Domain -> UiModel."},
	{ID: "error-handling-layer-mapping", Cluster: "error-handling", Severity: "warning", Status: "live", FixHint: "Catch specific Data exceptions (Network/DB), map them to Domain Result/Exception, and expose explicit UiState.Error for presentation."},
	{ID: "arch-viewmodel-mvi-suggestion", Cluster: "architecture", Severity: "info", Status: "live", FixHint: "Managing multiple disjoint StateFlows increases complexity. Consider MVI architecture (_uiState.update { ... } with a unified UiState data class)."},
	{ID: "compose-recomposition-optimizer", Cluster: "compose-performance", Severity: "warning", Status: "live", FixHint: "Prevent unnecessary recompositions: break down large Composables (>80 lines), use lambda modifiers/graphicsLayer, and annotate state with @Immutable."},
	{ID: "ui-hardcoded-strings", Cluster: "clean-code", Severity: "info", Status: "live", FixHint: "Extract hardcoded UI text to stringResource(R.string...) for localization, testing, and reuse."},
	{ID: "testability-direct-instantiation", Cluster: "testing", Severity: "error", Status: "live", FixHint: "Inject dependencies through constructors (Hilt/Koin/Manual DI) instead of instantiating concrete RepositoryImpl or Services directly."},
	{ID: "arch-udf-sealed-events", Cluster: "architecture", Severity: "warning", Status: "live", FixHint: "Use a sealed interface (UiEvent/UiAction) to handle UI actions cleanly in a Unidirectional Data Flow."},
	{ID: "arch-repository-impl-interface", Cluster: "architecture", Severity: "error", Status: "live", FixHint: "Ensure RepositoryImpl classes implement their corresponding domain Repository interface (DIP contract)."},
	// --- detekt-compose plugin (io.nlopez.compose.rules) ---
	//
	// These ids come from the plugin own generated config, not from memory.
	// The earlier Compose entries named rules that do not exist in the plugin
	// (ReusedModifierInstance, ModifierHeightWithText, CollectAsStateWithLifecycle,
	// DerivedStateWithoutRemember), so they could never fire even with it loaded.
	{ID: "compose-lambda-in-restartable-effect", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:LambdaParameterInRestartableEffect", Status: "live", FixHint: "Wrap the lambda in rememberUpdatedState() before using it inside LaunchedEffect/DisposableEffect, or the effect will capture a stale reference."},
	{ID: "compose-remember-content-missing", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:RememberContentMissing", Status: "live", FixHint: "Wrap movableContentOf in remember { ... } so the content is not recreated on every recomposition."},
	{ID: "compose-mutable-params", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:MutableParams", Status: "live", FixHint: "Do not pass mutable objects (ArrayList, var-holding classes) to a Composable: Compose cannot detect their changes, so the UI silently goes stale."},
	{ID: "compose-modifier-composed", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierComposed", Status: "live", FixHint: "Modifier.composed is deprecated for performance reasons. Migrate the modifier to Modifier.Node."},
	{ID: "compose-modifier-missing", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierMissing", Status: "live", FixHint: "Public Composables that emit UI should take a Modifier parameter so callers can size and position them."},
	{ID: "compose-modifier-without-default", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierWithoutDefault", Status: "live", FixHint: "Give the Modifier parameter a default value of Modifier so callers may omit it."},
	{ID: "compose-modifier-not-used-at-root", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierNotUsedAtRoot", Status: "live", FixHint: "Apply the Modifier parameter to the root element of the Composable, otherwise caller layout instructions are ignored."},
	{ID: "arch-composition-local-allowlist", Cluster: "architecture", Severity: "warning", DetektRule: "Compose:CompositionLocalAllowlist", Status: "live", FixHint: "CompositionLocals are implicit dependencies that make Composables hard to test. Pass the value explicitly, or add the local to the allowlist if it is intentional."},
	{ID: "arch-compose-viewmodel-forwarding", Cluster: "architecture", Severity: "warning", DetektRule: "Compose:ViewModelForwarding", Status: "live", FixHint: "Do not forward a ViewModel down to child Composables. Pass the state it exposes and lambdas for the events instead."},
	{ID: "arch-compose-viewmodel-injection", Cluster: "architecture", Severity: "warning", DetektRule: "Compose:ViewModelInjection", Status: "live", FixHint: "Obtain the ViewModel at the screen-level Composable and pass state down, rather than injecting it deep in the tree."},
	{ID: "compose-content-emitter-returning-values", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ContentEmitterReturningValues", Status: "live", FixHint: "A Composable should either emit UI or return a value, not both."},
	{ID: "compose-material2", Cluster: "compose-performance", Severity: "info", DetektRule: "Compose:Material2", Status: "live", FixHint: "This project targets Material 3. Replace the androidx.compose.material (M2) import with androidx.compose.material3."},
	{ID: "dead-compose-preview-public", Cluster: "dead-code", Severity: "info", DetektRule: "Compose:PreviewPublic", Status: "live", FixHint: "@Preview Composables are only used by the IDE. Make them private so they are not part of the public API."},
	{ID: "clean-compose-param-order", Cluster: "clean-code", Severity: "info", DetektRule: "Compose:ComposableParamOrder", Status: "live", FixHint: "Order Composable parameters as: required, then Modifier, then optional. It makes call sites readable and trailing lambdas work."},
	{ID: "naming-compose-parameters", Cluster: "naming", Severity: "info", DetektRule: "Compose:ParameterNaming", Status: "live", FixHint: "Name lambda parameters that describe events as onSomething, and slot parameters as nouns."},
	{ID: "compose-state-autoboxing", Cluster: "compose-performance", Severity: "info", DetektRule: "Compose:MutableStateAutoboxing", Status: "live", FixHint: "Use mutableIntStateOf/mutableLongStateOf/mutableFloatStateOf instead of mutableStateOf for primitives to avoid autoboxing on every write."},
}

func main() {
	out := flag.String("out", "rules/metadata.json", "output path for the generated metadata.json")
	flag.Parse()

	if problems := validateCatalog(CatalogRules); len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", p)
		}
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

// validateCatalog enforces the invariants the catalog has to hold, instead of
// a hardcoded rule count that only ever told you the number changed.
//
// The duplicate-DetektRule check is the important one: rulemap.BuildIndex maps
// DetektRule -> Rule, so two entries sharing one detekt rule silently meant the
// second overwrote the first and one kdoctor rule became unreachable. That is
// exactly what happened to Compose:UnstableCollections and
// Compose:ReusedModifierInstance, and nothing caught it.
func validateCatalog(rules []Rule) []string {
	var problems []string
	seenID := map[string]bool{}
	seenDetekt := map[string]string{}

	for _, r := range rules {
		if r.ID == "" {
			problems = append(problems, "rule with empty ID")
			continue
		}
		if seenID[r.ID] {
			problems = append(problems, fmt.Sprintf("duplicate rule ID %q", r.ID))
		}
		seenID[r.ID] = true

		switch r.Status {
		case "live", "planned":
		default:
			problems = append(problems, fmt.Sprintf("rule %q has invalid status %q", r.ID, r.Status))
		}

		switch r.Severity {
		case "error", "warning", "info":
		default:
			problems = append(problems, fmt.Sprintf("rule %q has invalid severity %q", r.ID, r.Severity))
		}

		// Only live rules are indexed, so only their detekt keys can collide.
		if r.DetektRule != "" && r.Status == "live" {
			if prev, ok := seenDetekt[r.DetektRule]; ok {
				problems = append(problems, fmt.Sprintf(
					"detekt rule %q is claimed by both %q and %q; BuildIndex keeps only one, "+
						"so the other can never fire", r.DetektRule, prev, r.ID))
			}
			seenDetekt[r.DetektRule] = r.ID
		}
	}
	return problems
}
