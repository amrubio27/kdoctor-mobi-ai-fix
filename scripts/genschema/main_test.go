package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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
	// helper: lee un campo string tolerando clave ausente (nil) por omitempty.
	strField := func(m map[string]any, key string) string {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}

	for _, gen := range CatalogRules {
		disk, ok := onDiskByID[gen.ID]
		if !ok {
			t.Errorf("regla %s falta en disco", gen.ID)
			continue
		}
		if strField(disk, "severity") != gen.Severity {
			t.Errorf("%s: severity disco=%q gen=%q", gen.ID, strField(disk, "severity"), gen.Severity)
		}
		if strField(disk, "cluster") != gen.Cluster {
			t.Errorf("%s: cluster disco=%q gen=%q", gen.ID, strField(disk, "cluster"), gen.Cluster)
		}
		if strField(disk, "detektRule") != gen.DetektRule {
			t.Errorf("%s: detektRule disco=%q gen=%q", gen.ID, strField(disk, "detektRule"), gen.DetektRule)
		}
		if strField(disk, "status") != gen.Status {
			t.Errorf("%s: status disco=%q gen=%q", gen.ID, strField(disk, "status"), gen.Status)
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

func TestGeneratedCatalogHas78Rules(t *testing.T) {
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
	if len(rules) != 78 {
		t.Fatalf("expected 78 rules, got %d", len(rules))
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

	nativeGoRules := map[string]bool{
		"compose-missing-key":             true,
		"coroutine-dispatchers-hardcoded": true,
		"sec-log-pii":                     true,
		"sec-webview-javascript-enabled":  true,
	}

	for _, r := range rules {
		if r["status"] == "live" {
			id, _ := r["id"].(string)
			if nativeGoRules[id] {
				continue
			}
			if _, ok := r["detektRule"]; !ok {
				t.Errorf("rule %s status=live but missing detektRule", r["id"])
			}
		}
	}
}

func TestCatalogRulesCountSanity(t *testing.T) {
	if len(CatalogRules) != 78 {
		t.Fatalf("CatalogRules debe tener 78 entries (64 V1 + 14 default detekt); tiene %d", len(CatalogRules))
	}
}

// validPrefixes mapea cada cluster kdoctor a los prefijos válidos de ID. V1
// usa abreviaciones (mem-*, arch-*, a11y-*, etc.); 5.11+ usa nombres completos
// (complexity-*, error-handling-*, etc.). Esto evita tener que renombrar V1
// sólo por consistencia, pero hace cumplir el invariante a futuro:
//
//	Para cualquier regla r ∈ CatalogRules:
//	  ∃ prefix ∈ validPrefixes[r.Cluster] : strings.HasPrefix(r.ID, prefix + "-")
//
// Cluster NO registrado en validPrefixes = FAIL loudante (es nuevo y el
// contributor debe añadir el prefijo válido).
var validPrefixes = map[string][]string{
	// V1 abreviaciones
	"compose-performance": {"compose"},
	"coroutines":          {"coroutine"},
	"lifecycle":           {"lifecycle"},
	"memory":              {"mem"},
	"architecture":        {"arch"},
	"accessibility":       {"a11y"},
	"testing":             {"test"},
	"security":            {"sec"},
	"kmp":                 {"kmp"},
	"dead-code":           {"dead"},
	// 5.11 nombres completos
	"complexity":     {"complexity"},
	"error-handling": {"error-handling"},
	"magic-numbers":  {"magic-numbers"},
	"naming":         {"naming"},
	"formatting":     {"formatting"},
} // validatePrefix comprueba que rule.ID empieza por uno de los prefijos válidos
// registrados para rule.Cluster en validPrefixes. Devuelve un error formateado
// (sin tipo dedicado — Fase 1 no discrimina errores por categoría, y plegar a
// fmt.Errorf reduce el código 12 líneas sin perder mensaje diagnóstico).
func validatePrefix(rule Rule) error {
	prefixes, ok := validPrefixes[rule.Cluster]
	if !ok {
		return fmt.Errorf("rule %s (cluster=%s): cluster no registrado en validPrefixes. Hint: añádelo al map arriba", rule.ID, rule.Cluster)
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(rule.ID, prefix+"-") {
			return nil
		}
	}
	return fmt.Errorf("rule %s (cluster=%s): ID no comienza por ningún prefijo válido. Hint: usa uno de [%s] seguido de '-'", rule.ID, rule.Cluster, strings.Join(prefixes, ", "))
}

// TestClusterTaxonomyIsConsistent itera TODAS las reglas del catálogo (no
// sólo 5.11) y verifica la convención de nombres. Esto previene la regresión
// del bug original (prefijo `style-*` cuando el cluster-Canónico era
// `complexity`) para cualquier futura adición.
func TestClusterTaxonomyIsConsistent(t *testing.T) {
	for _, r := range CatalogRules {
		if err := validatePrefix(r); err != nil {
			t.Error(err)
		}
	}
}

// TestValidPrefixesCoverage es un meta-guardrail: si añades un NUEVO cluster
// a CatalogRules, tienes que añadirlo también a validPrefixes. Si no, este
// test falla con un mensaje claro.
func TestValidPrefixesCoverage(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range CatalogRules {
		seen[r.Cluster] = true
	}
	for cluster := range seen {
		if _, ok := validPrefixes[cluster]; !ok {
			t.Errorf("cluster %q aparece en CatalogRules pero no está en validPrefixes. Hint: añade el cluster y sus prefijos válidos al map.", cluster)
		}
	}
}
