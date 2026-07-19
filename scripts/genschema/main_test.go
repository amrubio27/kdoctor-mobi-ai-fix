package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestCatalogConvergence es el guardrail contra drift entre CatalogRules
// (en este paquete) y rules/metadata.json (en disco). Si editas uno sin
// tocar el otro, este test falla con un diff claro.
func TestCatalogConvergence(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "rules", "metadata.json"))
	if err != nil || len(files) == 0 {
		t.Fatal("rules/metadata.json not found")
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var onDisk []map[string]any
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatal(err)
	}

	if len(CatalogRules) != len(onDisk) {
		t.Fatalf("drift: CatalogRules tiene %d reglas, disco tiene %d",
			len(CatalogRules), len(onDisk))
	}

	// Verifica que cada regla del slice está en disco con sus campos clave.
	onDiskByID := map[string]map[string]any{}
	for _, r := range onDisk {
		if id, ok := r["id"].(string); ok {
			onDiskByID[id] = r
		}
	}
	for _, gen := range CatalogRules {
		disk, ok := onDiskByID[gen.ID]
		if !ok {
			t.Errorf("regla %s falta en disco", gen.ID)
			continue
		}
		if disk["severity"] != gen.Severity {
			t.Errorf("%s: severity disco=%q gen=%q", gen.ID, disk["severity"], gen.Severity)
		}
		if disk["cluster"] != gen.Cluster {
			t.Errorf("%s: cluster disco=%q gen=%q", gen.ID, disk["cluster"], gen.Cluster)
		}
		if disk["detektRule"] != gen.DetektRule {
			t.Errorf("%s: detektRule disco=%q gen=%q", gen.ID, disk["detektRule"], gen.DetektRule)
		}
		if disk["status"] != gen.Status {
			t.Errorf("%s: status disco=%q gen=%q", gen.ID, disk["status"], gen.Status)
		}
	}

	// Set de IDs idéntico.
	genIDs := make([]string, 0, len(CatalogRules))
	for _, r := range CatalogRules {
		genIDs = append(genIDs, r.ID)
	}
	sort.Strings(genIDs)
	diskIDs := make([]string, 0, len(onDiskByID))
	for id := range onDiskByID {
		diskIDs = append(diskIDs, id)
	}
	sort.Strings(diskIDs)
	if !reflect.DeepEqual(genIDs, diskIDs) {
		t.Fatalf("IDs divergen:\n  gen: %v\n  disk: %v", genIDs, diskIDs)
	}
}

func TestGeneratedCatalogHas64Rules(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "rules", "metadata.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("rules/metadata.json not found")
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var rules []map[string]any
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) != 64 {
		t.Fatalf("expected 64 rules, got %d", len(rules))
	}
	required := []string{"id", "cluster", "severity", "status"}
	for i, r := range rules {
		for _, k := range required {
			if _, ok := r[k]; !ok {
				t.Fatalf("rule #%d missing %q: %+v", i, k, r)
			}
		}
		if r["status"] != "live" && r["status"] != "planned" {
			t.Fatalf("rule #%d has bad status %q", i, r["status"])
		}
	}
}

func TestLiveRulesHaveDetektMapping(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "rules", "metadata.json"))
	data, _ := os.ReadFile(files[0])
	var rules []map[string]any
	_ = json.Unmarshal(data, &rules)
	for _, r := range rules {
		if r["status"] == "live" {
			if _, ok := r["detektRule"]; !ok {
				t.Errorf("rule %s status=live but missing detektRule", r["id"])
			}
		}
	}
}

func TestCatalogRulesCountSanity(t *testing.T) {
	if len(CatalogRules) != 64 {
		t.Fatalf("CatalogRules debe tener 64 entries; tiene %d", len(CatalogRules))
	}
}
