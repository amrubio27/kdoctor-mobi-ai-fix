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

func TestLen(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{ID: "a", DetektRule: "A", Status: "live"},
		{ID: "b", DetektRule: "B", Status: "live"},
		{ID: "c", DetektRule: "C", Status: "planned"}, // ignored
	})
	if idx.Len() != 2 {
		t.Fatalf("Len expected 2, got %d", idx.Len())
	}
}
