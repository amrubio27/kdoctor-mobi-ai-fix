package rulemap

import (
	"reflect"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

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
		t.Fatalf("expected b.kt last, got %s:%d", out[2].File, out[2].Line)
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
