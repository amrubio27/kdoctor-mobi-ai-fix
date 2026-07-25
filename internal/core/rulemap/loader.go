// Package rulemap carga el catálogo de reglas y mapea IDs Detekt a IDs kdoctor.
package rulemap

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adkd/adkd/internal/core/types"
)

//go:embed metadata.json
var embeddedMetadata []byte

// RulesSource indica de dónde provienen las reglas cargadas.
type RulesSource string

const (
	SourceProject  RulesSource = "project (.kdoctor/rules.json)"
	SourceExplicit RulesSource = "explicit path"
	SourceUser     RulesSource = "user cache (~/.kdoctor/rules/metadata.json)"
	SourceLocal    RulesSource = "local repo file (rules/metadata.json)"
	SourceEmbedded RulesSource = "embedded binary fallback"
)

// LoadResult contiene las reglas cargadas y el origen.
type LoadResult struct {
	Rules  []types.Rule
	Source RulesSource
	Path   string
}

// LoadRules mantiene compatibilidad hacia atrás leyendo de un path específico.
func LoadRules(path string) ([]types.Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules metadata %s: %w", path, err)
	}
	var rules []types.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules metadata: %w", err)
	}
	return rules, nil
}

// LoadRulesCascade resuelve las reglas siguiendo el orden offline-first:
// 1. Path explícito (si se pasa via flag o env KDOCTOR_RULES_DIR)
// 2. Reglas del proyecto en <projectDir>/.kdoctor/rules.json
// 3. Caché global del usuario (~/.kdoctor/rules/metadata.json)
// 4. Archivo local del repositorio (rules/metadata.json)
// 5. Embedded binary fallback (go:embed)
func LoadRulesCascade(projectDir, explicitPath string) (*LoadResult, error) {
	// 1. Explicit path
	if explicitPath != "" {
		if rules, err := LoadRules(explicitPath); err == nil {
			return &LoadResult{Rules: rules, Source: SourceExplicit, Path: explicitPath}, nil
		}
	}
	if p := os.Getenv("KDOCTOR_RULES_DIR"); p != "" {
		candidate := filepath.Join(p, "metadata.json")
		if rules, err := LoadRules(candidate); err == nil {
			return &LoadResult{Rules: rules, Source: SourceExplicit, Path: candidate}, nil
		}
	}

	// 2. Project local rule override
	if projectDir != "" {
		projRules := filepath.Join(projectDir, ".kdoctor", "rules.json")
		if rules, err := LoadRules(projRules); err == nil {
			return &LoadResult{Rules: rules, Source: SourceProject, Path: projRules}, nil
		}
	}

	// 3. User global cache ~/.kdoctor/rules/metadata.json
	userCache, err := GetUserCachePath()
	if err == nil {
		if rules, err := LoadRules(userCache); err == nil {
			return &LoadResult{Rules: rules, Source: SourceUser, Path: userCache}, nil
		}
	}

	// 4. Executable / CWD local repository relative rules
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "rules", "metadata.json"),
			filepath.Join(dir, "..", "rules", "metadata.json"),
		}
		for _, c := range candidates {
			if rules, err := LoadRules(c); err == nil {
				return &LoadResult{Rules: rules, Source: SourceLocal, Path: c}, nil
			}
		}
	}
	if rules, err := LoadRules("rules/metadata.json"); err == nil {
		return &LoadResult{Rules: rules, Source: SourceLocal, Path: "rules/metadata.json"}, nil
	}

	// 5. Embedded binary fallback
	var rules []types.Rule
	if err := json.Unmarshal(embeddedMetadata, &rules); err != nil {
		return nil, fmt.Errorf("parse embedded rules metadata: %w", err)
	}
	return &LoadResult{Rules: rules, Source: SourceEmbedded, Path: "binary:embedded"}, nil
}

// GetUserCachePath devuelve la ruta ~/.kdoctor/rules/metadata.json.
func GetUserCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kdoctor", "rules", "metadata.json"), nil
}
