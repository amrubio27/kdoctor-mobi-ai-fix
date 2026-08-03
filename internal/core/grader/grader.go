// Package grader implementa el motor de Health Score.
package grader

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/adkd/adkd/internal/core/types"
)

// Score calcula la puntuación determinista de salud.
func Score(findings []types.Finding) (int, types.Summary) {
	return ScoreWithKLOC(findings, 0)
}

// clusterMultiplier returns the weight multiplier for a given cluster category.
func clusterMultiplier(cluster string) float64 {
	switch strings.ToLower(cluster) {
	case "security":
		return 2.0
	case "architecture":
		return 1.5
	case "coroutines", "memory", "lifecycle":
		return 1.25
	case "compose-performance", "testing", "kmp":
		return 1.0
	case "clean-code", "complexity", "error-handling", "magic-numbers", "naming", "formatting", "dead-code", "accessibility":
		return 0.75
	default:
		return 0.75
	}
}

// isCriticalFinding checks if a finding represents a critical non-dilutable error.
func isCriticalFinding(f types.Finding) bool {
	if f.Severity != types.SeverityError {
		return false
	}
	c := strings.ToLower(f.Cluster)
	return c == "security" || c == "architecture" || c == "memory"
}

// ScoreWithKLOC calcula la puntuación ajustada por la densidad de código (KLOC)
// sin saltos bruscos (función continua), no diluible para fallos críticos de
// seguridad/arquitectura y con rendimientos decrecientes para alertas repetidas.
func ScoreWithKLOC(findings []types.Finding, totalLines int) (int, types.Summary) {
	var s types.Summary
	fileRuleCounts := make(map[string]int)

	var criticalPenalty float64
	var regularPenalty float64
	var infoPenalty float64

	for _, f := range findings {
		s.Total++
		switch f.Severity {
		case types.SeverityError:
			s.Errors++
		case types.SeverityWarning:
			s.Warnings++
		case types.SeverityInfo:
			s.Info++
		}

		// Calculate diminishing return factor per (File, Rule)
		key := f.File + ":" + f.Rule
		count := fileRuleCounts[key]
		fileRuleCounts[key] = count + 1

		var repeatFactor float64
		switch count {
		case 0:
			repeatFactor = 1.0
		case 1:
			repeatFactor = 0.75
		case 2:
			repeatFactor = 0.50
		default:
			repeatFactor = 0.25
		}

		var baseWeight float64
		switch f.Severity {
		case types.SeverityError:
			baseWeight = 5.0
		case types.SeverityWarning:
			baseWeight = 2.0
		case types.SeverityInfo:
			baseWeight = 0.5
		default:
			baseWeight = 1.0
		}

		itemPenalty := baseWeight * clusterMultiplier(f.Cluster) * repeatFactor

		if f.Severity == types.SeverityInfo {
			infoPenalty += itemPenalty
		} else if isCriticalFinding(f) {
			criticalPenalty += itemPenalty
		} else {
			regularPenalty += itemPenalty
		}
	}

	// Cap info penalty at 10.0 points max
	if infoPenalty > 10.0 {
		infoPenalty = 10.0
	}

	// Smooth continuous scale factor based on KLOC (sqrt function, no 300-line cliff)
	kloc := float64(totalLines) / 1000.0
	scaleFactor := math.Max(1.0, math.Sqrt(kloc))

	totalPenalty := criticalPenalty + ((regularPenalty + infoPenalty) / scaleFactor)

	score := 100 - int(math.Round(totalPenalty))
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score, s
}

// CountKotlinLines cuenta el número de líneas de código en archivos .kt y .kts.
func CountKotlinLines(projectDir string) int {
	if projectDir == "" {
		return 0
	}
	total := 0
	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".kt") || strings.HasSuffix(name, ".kts") {
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer func() { _ = f.Close() }()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "//") {
					total++
				}
			}
		}
		return nil
	})
	return total
}
