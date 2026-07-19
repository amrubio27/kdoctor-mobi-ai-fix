// Tests del rulemap. Cubre tres contratos:
//
//  1. Mapping prefix-strip (Bug 1 del plan maestro): detekt SARIF emite
//     ruleIds calificados como "detekt.<ruleset>.<Name>" o más profundos
//     ("io.gitlab.arturbosch.detekt.<Name>"), pero el catálogo guarda sólo
//     el nombre corto. El Map() debe normalizar con strings.LastIndex(".")
//     antes del lookup. TestMapPrefixStrip cubre el caso simple;
//     TestBug1MultiVendorPrefixStrip es más exhaustivo (1–4 niveles de
//     vendor prefix) y es el regression guard oficial del fix Bug 1.
//
//  2. Index doble (byDetekt + byID): los detectores regex nativos de
//     internal/core/rules emiten findings con Rule = rule.ID (no
//     DetektRule asociado). El Map() debe resolverlos via byID, no sólo
//     via byDetekt. TestMapByIDForNativeRules cubre ese contrato.
//
//  3. ApplyOverrides: kdoctor.config.yaml con excludes/severity-overrides
//     se aplica correctamente post-Mapping. Es el Tier 3#7 del roadmap.
package rulemap

import (
	"reflect"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

// TestMapPrefixStrip regression-guard: detekt SARIF outputs qualified ruleIds
// como `detekt.style.UnusedImports` pero el catálogo guarda `UnusedImports`.
// Sin el strip del prefijo "detekt.<ruleset>." antes del lookup, ningún
// finding con prefix vendor/ruleset mapearía. Este test asegura que el strip
// se mantenga incluso cuando alguien reorganice la función Map.
func TestMapPrefixStrip(t *testing.T) {
	rules := []types.Rule{
		{ID: "dead-unused-import", Cluster: "dead-code", Severity: types.SeverityInfo,
			DetektRule: "UnusedImports", Status: "live"},
		{ID: "arch-god-class", Cluster: "architecture", Severity: types.SeverityWarning,
			DetektRule: "TooManyFunctions", Status: "live"},
	}
	idx := BuildIndex(rules)
	in := []types.Finding{
		{Rule: "detekt.style.UnusedImports", Message: "import not used"},
		{Rule: "detekt.complexity.TooManyFunctions", Message: "too many methods"},
		{Rule: "detekt.empty.EmptyFunctionBlock.NonExistent", Message: "should remain unmapped"},
	}
	out := idx.Map(in)
	if out[0].ID != "dead-unused-import" {
		t.Errorf("style.UnusedImports should map to dead-unused-import, got %q", out[0].ID)
	}
	if out[1].ID != "arch-god-class" {
		t.Errorf("complexity.TooManyFunctions should map to arch-god-class, got %q", out[1].ID)
	}
	if out[2].ID != "unmapped:detekt.empty.EmptyFunctionBlock.NonExistent" {
		t.Errorf("unknown rule should preserve original prefix in unmapped:, got %q", out[2].ID)
	}
	// Casos borde: empty rule ID no debe crashear; regla sin prefijo
	// (sin dot) también debe matchear; string que es sólo un punto también.
	edge := idx.Map([]types.Finding{{Rule: ""}, {Rule: "TooManyFunctions"}, {Rule: "."}})
	if edge[0].ID != "unmapped:" {
		t.Errorf("empty rule ID should produce unmapped:, got %q", edge[0].ID)
	}
	if edge[1].ID != "arch-god-class" {
		t.Errorf("rule without prefix should still map, got %q", edge[1].ID)
	}
	if edge[2].ID != "unmapped:." {
		t.Errorf("trailing-dot rule should preserve original, got %q", edge[2].ID)
	}
}

// TestBug1MultiVendorPrefixStrip es el regression guard **oficial** del
// Bug 1 del plan maestro (Tarea 1.6 / Bug 1). Verifica que el mapping
// normaliza QUALQUIER profundidad de vendor+ruleset antes del lookup:
//
//   - Sin prefijo  → lookup directo
//   - 1 nivel      → "detekt.<ruleset>.<Name>" → strip last dot
//   - 2 niveles    → "io.gitlab.arturbosch.detekt.<Name>" → strip last dot
//   - 4 niveles    → "com.example.team.detekt.<ruleset>.<Name>" → strip last dot
//
// Si alguien cambia la lógica de `strings.LastIndex(".")` por `strings.Index`
// (sólo el PRIMER prefijo) o por un hardcode "detekt.<ruleset>.", este test
// falla inmediatamente. Crítico para no romper el lookup cuando detekt 2.x
// emita qualified ruleIds más profundos o cuando nuevas reglas custom
// adopten otros namespaces.
func TestBug1MultiVendorPrefixStrip(t *testing.T) {
	rules := []types.Rule{
		{ID: "complexity-long-method", Cluster: "complexity", Severity: types.SeverityWarning,
			DetektRule: "LongMethod", Status: "live"},
		{ID: "naming-class-convention", Cluster: "naming", Severity: types.SeverityInfo,
			DetektRule: "ClassNaming", Status: "live"},
	}
	idx := BuildIndex(rules)

	cases := []struct {
		name     string
		ruleIDIn string
		wantID   string
	}{
		{"no_prefix", "LongMethod", "complexity-long-method"},
		{"single_vendor_prefix", "detekt.complexity.LongMethod", "complexity-long-method"},
		{"two_vendor_prefixes_community_detekt", "io.gitlab.arturbosch.detekt.complexity.LongMethod", "complexity-long-method"},
		{"deep_corporate_nesting", "com.example.team.detekt.naming.ClassNaming", "naming-class-convention"},
		{"unknown_rule_with_prefix_unmapped", "detekt.unknown.NotInCatalog", "unmapped:detekt.unknown.NotInCatalog"},
		{"empty_rule_id", "", "unmapped:"},
		{"single_dot_strips_to_empty", ".", "unmapped:."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := idx.Map([]types.Finding{{Rule: tc.ruleIDIn}})
			if len(out) != 1 {
				t.Fatalf("expected 1 finding, got %d", len(out))
			}
			if out[0].ID != tc.wantID {
				t.Errorf("Rule %q mapped to %q, want %q", tc.ruleIDIn, out[0].ID, tc.wantID)
			}
		})
	}
}

// TestMapByIDForNativeRules documenta y prueba el contrato introducido en
// Tier 1#2: los detectores regex nativos (compose-missing-key, sec-log-pii,
// sec-webview-javascript-enabled, coroutine-dispatchers-hardcoded) emiten
// findings con Rule = rule.ID. El catálogo NO tiene DetektRule asociado para
// estos — son pure-Go, post-AST (regex). El Map() debe resolverlos via
// byID, no sólo via byDetekt (que no tendría key).
//
// Si alguien refactoriza BuildIndex para keyear sólo por DetektRule, este
// test falla y los 4 detectores nativos dejan de mapear a sus IDs del
// catálogo, apareciendo como "unmapped:sec-log-pii" en reports.
func TestMapByIDForNativeRules(t *testing.T) {
	rules := []types.Rule{
		{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError,
			Status: "live", FixHint: "Do not log PII"},
		{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityError,
			Status: "live", FixHint: "Add key parameter"},
		{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo,
			Status: "live", FixHint: "Inject dispatchers"},
		{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: types.SeverityError,
			Status: "live", FixHint: "Disable JS unless necessary"},
	}
	idx := BuildIndex(rules)

	// Note: el output de Map() se ordena por (file, line, col). El orden
	// esperado es lexicográfico ascendente de file. Para evitar acoplar el
	// test al ordenamiento interno, validamos cada (file, id) vía lookup
	// por file una sola vez. (Solución table-driven — equivalente en costo
	// cognitivo a per-index, pero más fácil de extender con un 5º detector.)
	cases := []struct {
		fileIn      string
		lineIn      int
		wantID      string
		wantCluster string
	}{
		{"App.kt", 10, "sec-log-pii", "security"},
		{"IO.kt", 7, "coroutine-dispatchers-hardcoded", "coroutines"},
		{"List.kt", 25, "compose-missing-key", "compose-performance"},
		{"Web.kt", 30, "sec-webview-javascript-enabled", "security"},
	}
	in := make([]types.Finding, 0, len(cases))
	for _, c := range cases {
		in = append(in, types.Finding{Rule: c.wantID, File: c.fileIn, Line: c.lineIn})
	}

	out := idx.Map(in)
	if len(out) != len(cases) {
		t.Fatalf("expected %d findings, got %d", len(cases), len(out))
	}
	// Index the output by file for stable lookup regardless of sort order.
	byFile := make(map[string]types.Finding, len(out))
	for _, f := range out {
		byFile[f.File] = f
	}
	for _, tc := range cases {
		got, ok := byFile[tc.fileIn]
		if !ok {
			t.Errorf("file %q missing from output", tc.fileIn)
			continue
		}
		if got.ID != tc.wantID {
			t.Errorf("file %q: expected ID %q, got %q", tc.fileIn, tc.wantID, got.ID)
		}
		if got.Cluster != tc.wantCluster {
			t.Errorf("file %q: expected cluster %q, got %q", tc.fileIn, tc.wantCluster, got.Cluster)
		}
		if got.FixHint == "" {
			t.Errorf("file %q: FixHint not populated", tc.fileIn)
		}
	}
}

func TestLoadRulesFromFixture(t *testing.T) {
	rules, err := LoadRules("testdata/metadata-sample.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("len %d", len(rules))
	}
}

func TestMapKnown(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{
			ID: "a", Cluster: "x", Severity: types.SeverityWarning,
			DetektRule: "Compose:ReusedModifierInstance", Status: "live", FixHint: "hoist",
		},
	})
	out := idx.Map([]types.Finding{{Rule: "Compose:ReusedModifierInstance", Message: "x"}})
	if len(out) != 1 {
		t.Fatal("len")
	}
	want := types.Finding{
		ID: "a", Cluster: "x", Severity: types.SeverityWarning,
		Rule: "Compose:ReusedModifierInstance", Message: "x", FixHint: "hoist",
	}
	if !reflect.DeepEqual(out[0], want) {
		t.Fatalf("\nwant %+v\ngot  %+v", want, out[0])
	}
}

func TestMapUnknown(t *testing.T) {
	idx := BuildIndex(nil)
	out := idx.Map([]types.Finding{{Rule: "Nope"}})
	if out[0].ID != "unmapped:Nope" {
		t.Fatal(out[0].ID)
	}
	if out[0].Cluster != "unknown" {
		t.Fatal("cluster")
	}
}

func TestMapIgnoresPlannedRules(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{
			ID: "planned-only", Cluster: "x", Severity: types.SeverityWarning,
			DetektRule: "DetektRule:Present", Status: "planned",
		},
	})
	out := idx.Map([]types.Finding{{Rule: "DetektRule:Present"}})
	if out[0].ID != "unmapped:DetektRule:Present" {
		t.Fatalf("planned rules deben NO matchear, got %s", out[0].ID)
	}
}

func TestMapStableOrderByLocation(t *testing.T) {
	idx := BuildIndex([]types.Rule{{ID: "a", Cluster: "x", Severity: types.SeverityInfo, DetektRule: "A", Status: "live"}})
	in := []types.Finding{
		{Rule: "A", File: "b.kt", Line: 5},
		{Rule: "A", File: "a.kt", Line: 10},
		{Rule: "A", File: "a.kt", Line: 3},
	}
	out := idx.Map(in)
	if out[0].File != "a.kt" || out[0].Line != 3 {
		t.Fatalf("expected a.kt:3 first, got %s:%d", out[0].File, out[0].Line)
	}
	if out[2].File != "b.kt" {
		t.Fatalf("expected b.kt last, got %s", out[2].File)
	}
}

// TestLen valida el wrapper del dual-index. Importante: el contrato actual
// es `Len() = len(byID)`, lo que significa que reglas `live` SIN DetektRule
// (los detectores regex nativos de Tier 1#2) entran en el conteo via byID.
// Ese path es asimétrico respecto a las reglas "Detekt-backed": esas tienen
// key en AMBOS maps y se cuentan una sola vez via byID.
// Si alguien refactoriza Len() para usar `len(byDetekt)`, las reglas nativas
// (compose-missing-key, sec-log-pii, sec-webview-javascript-enabled,
// coroutine-dispatchers-hardcoded) no entrarían en el conteo y Len()
// reportaría un número más bajo que el real.
func TestLen(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{ID: "a", DetektRule: "A", Status: "live"},
		{ID: "b", DetektRule: "B", Status: "live"},
		{ID: "c", DetektRule: "C", Status: "planned"}, // ignored
		{ID: "d", Status: "live"},                     // native-only: no DetektRule
	})
	if idx.Len() != 3 {
		t.Fatalf("Len expected 3 (a, b, d), got %d", idx.Len())
	}
}

// TestApplyOverrides (Tier 3#7): config de equipo — excludes + severity overrides.
// - Si f.File hace match con algún patrón de cfg.Excludes, se descarta.
// - Si cfg.Rules tiene un override para f.ID o f.Cluster, se aplica.
// - Si la severidad resultante es "off"/"disabled"/"none", se descarta.
func TestApplyOverrides(t *testing.T) {
	findings := []types.Finding{
		{ID: "rule1", Cluster: "compose-performance", Severity: types.SeverityWarning, File: "src/Foo.kt"},
		{ID: "rule2", Cluster: "compose-performance", Severity: types.SeverityInfo, File: "src/Bar.kt"},
		{ID: "rule3", Cluster: "architecture", Severity: types.SeverityError, File: "src/Baz.kt"},
	}

	excludes := []string{"**/Bar.kt"}
	overrides := map[string]string{
		"compose-performance": "error", // cluster level override
		"rule1":               "off",   // rule level wins over cluster
	}

	out := ApplyOverrides(findings, excludes, overrides)

	if len(out) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(out))
	}
	// rule1 dropped ("off")
	// rule2 dropped (excluded by pattern)
	// rule3 kept with severity unchanged
	if out[0].ID != "rule3" {
		t.Errorf("expected rule3, got %s", out[0].ID)
	}
	if out[0].Severity != types.SeverityError {
		t.Errorf("expected error severity (untouched), got %s", out[0].Severity)
	}
}

// TestApplyOverrides_SeverityChanges verifica que los overrides de severity
// (rule-level y cluster-level) se aplican correctamente sin perder findings.
func TestApplyOverrides_SeverityChanges(t *testing.T) {
	findings := []types.Finding{
		{ID: "rule1", Cluster: "perf", Severity: types.SeverityWarning, File: "Foo.kt"},
	}
	overrides := map[string]string{
		"perf": "info", // cluster → info (downgrade)
	}
	out := ApplyOverrides(findings, nil, overrides)
	if len(out) != 1 || out[0].Severity != types.SeverityInfo {
		t.Errorf("cluster override failed: got %+v", out)
	}
}

// TestApplyOverrides_GlobFallback cubre el path `**` glob en ApplyOverrides.
// El round-2 análisis lo marcó como implementación toy (strings.Contains en
// lugar de un engine real). Lo testeamos para que cualquier refactor futuro
// que migre a una librería de globs robusta (doublestar, etc.) mantenga el
// comportamiento observable. Con el patrón `"**/Legacy.kt"`, el archivo
// `src/main/Legacy.kt` debe excluirse.
func TestApplyOverrides_GlobFallback(t *testing.T) {
	findings := []types.Finding{
		{ID: "rule1", Cluster: "compose", Severity: types.SeverityWarning, File: "src/main/Legacy.kt"},
		{ID: "rule2", Cluster: "compose", Severity: types.SeverityWarning, File: "src/main/New.kt"},
	}
	excludes := []string{"**/Legacy.kt"}
	out := ApplyOverrides(findings, excludes, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 finding (Legacy.kt excluded), got %d", len(out))
	}
	if out[0].ID != "rule2" {
		t.Errorf("expected rule2 to survive, got %s", out[0].ID)
	}
}

// TestApplyOverrides_CaseInsensitiveSeverity cubre el path `strings.ToLower`
// en el manejo del flag de severidad. Usuarios en YAML pueden tipear
// "OFF" / "Disabled" / "NONE" por costumbre o copiando de otros tools —
// el parser lo debe respetar.
func TestApplyOverrides_CaseInsensitiveSeverity(t *testing.T) {
	cases := []string{"OFF", "Off", "off", "DISABLED", "Disabled", "NONE", "None"}
	for _, sev := range cases {
		t.Run("severity_"+sev, func(t *testing.T) {
			findings := []types.Finding{
				{ID: "rule1", Cluster: "x", Severity: types.SeverityWarning, File: "A.kt"},
			}
			overrides := map[string]string{"rule1": sev}
			out := ApplyOverrides(findings, nil, overrides)
			if len(out) != 0 {
				t.Errorf("severity %q should drop the finding, got %d remaining", sev, len(out))
			}
		})
	}
}

// TestApplyOverrides_EarlyReturn cubre la rama defensiva:
//
//	if len(excludes) == 0 && len(overrides) == 0 {
//	    return findings
//	}
//
// Garantiza identidad (input == output) sin allocations ni mutaciones.
// Si alguien refactoriza esa rama y por error descarta findings en alguna
// condición límite, este test falla.
func TestApplyOverrides_EarlyReturn(t *testing.T) {
	findings := []types.Finding{
		{ID: "r1", Cluster: "x", Severity: types.SeverityWarning, File: "A.kt"},
		{ID: "r2", Cluster: "y", Severity: types.SeverityError, File: "B.kt"},
	}
	out := ApplyOverrides(findings, nil, nil)
	if len(out) != len(findings) {
		t.Fatalf("expected %d findings unchanged, got %d", len(findings), len(out))
	}
	for i := range findings {
		if !reflect.DeepEqual(out[i], findings[i]) {
			t.Errorf("position %d: expected %+v, got %+v", i, findings[i], out[i])
		}
	}
}

// TestApplyOverrides_ClusterWarningDowncast pin el contrato del cluster-
// level override: cuando `overrides[f.Cluster] = "warning"` (downcast), el
// finding SE MANTIENE en el output pero con severity=WARNING. NO se dropea.
// (ApplyOverrides solo dropea con off/disabled/none).
//
// Cubre la rama cluster-override del `examples/bad-project/kdoctor.config.yaml`:
// `security: warning`. Si alguien refactoriza ApplyOverrides para también
// dropear con override "info" o "warning", este test falla.
func TestApplyOverrides_ClusterWarningDowncast(t *testing.T) {
	findings := []types.Finding{
		{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: types.SeverityError, File: "Web.kt"},
		{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError, File: "App.kt"},
	}
	overrides := map[string]string{"security": "warning"}

	out := ApplyOverrides(findings, nil, overrides)
	if len(out) != 2 {
		t.Fatalf("downcast should NOT drop findings: expected 2, got %d", len(out))
	}
	wantSeverities := map[string]types.Severity{
		"sec-webview-javascript-enabled": types.SeverityWarning,
		"sec-log-pii":                    types.SeverityWarning,
	}
	for _, f := range out {
		want, ok := wantSeverities[f.ID]
		if !ok {
			t.Errorf("unexpected ID %q in output", f.ID)
			continue
		}
		if f.Severity != want {
			t.Errorf("%q expected severity %q, got %q", f.ID, want, f.Severity)
		}
	}
}

// TestApplyOverrides_RuleLevelWinsOverCluster pin el contrato de
// precedencia: cuando AMBAS overrides, rule-override y cluster-override,
// matchean el mismo finding, la RULE-LEVEL override gana. Implementado
// en `internal/core/rulemap/mapping.go::ApplyOverrides`:
//
//	if sev, ok := overrides[f.ID]; ok {
//	    effectiveSeverity = sev        // rule-level gana
//	} else if sev, ok := overrides[f.Cluster]; ok {
//	    effectiveSeverity = sev        // cluster-level fallback
//	}
//
// Caso concreto (el `examples/bad-project/kdoctor.config.yaml` real):
//
//	security: warning        (cluster override: downcast)
//	sec-log-pii: error       (rule override: mantiene error)
//
// Esperado:
//   - sec-log-pii: rule-level `error` > cluster-level `warning` → mantiene error
//   - sec-webview-javascript-enabled: sin rule-override → solo cluster → warning
//
// Si alguien refactoriza ApplyOverrides y rompe la precedencia (e.g. cluster
// antes que rule), este test falla y el round-2 polish del fixture rompe
// silenciosamente.
func TestApplyOverrides_RuleLevelWinsOverCluster(t *testing.T) {
	findings := []types.Finding{
		{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError, File: "App.kt"},
		{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: types.SeverityError, File: "Web.kt"},
	}
	overrides := map[string]string{
		"security":    "warning", // cluster: baja a warning
		"sec-log-pii": "error",   // rule: gana sobre cluster
	}

	out := ApplyOverrides(findings, nil, overrides)
	if len(out) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(out))
	}
	wantSeverities := map[string]types.Severity{
		"sec-log-pii":                    types.SeverityError,   // rule wins
		"sec-webview-javascript-enabled": types.SeverityWarning, // only cluster
	}
	for _, f := range out {
		want, ok := wantSeverities[f.ID]
		if !ok {
			t.Errorf("unexpected ID %q in output", f.ID)
			continue
		}
		if f.Severity != want {
			t.Errorf("%q: expected severity %q (rule-wins-over-cluster / cluster-only), got %q",
				f.ID, want, f.Severity)
		}
	}
}
