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

// ScoreWithKLOC calcula la puntuación ajustada por la densidad de código (KLOC)
// si totalLines > 0, evitando penalizaciones excesivas e injustas en grandes proyectos.
func ScoreWithKLOC(findings []types.Finding, totalLines int) (int, types.Summary) {
	var s types.Summary
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
	}

	weightedPenalties := float64(s.Errors)*5.0 + float64(s.Warnings)*2.0 + float64(s.Info)*0.5

	var raw int
	if totalLines > 300 {
		kloc := float64(totalLines) / 1000.0
		density := weightedPenalties / math.Max(0.5, kloc)
		raw = 100 - int(math.Round(density*3.5))
	} else {
		raw = 100 - int(weightedPenalties)
	}

	if raw > 100 {
		raw = 100
	}
	if raw < 0 {
		raw = 0
	}
	return raw, s
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
