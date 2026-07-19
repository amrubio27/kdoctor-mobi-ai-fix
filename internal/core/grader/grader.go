// Package grader implementa el motor de Health Score.
//
// Fórmula V1:
//   raw = 100 - errors*5 - warnings*2 - info*0.5
//   score = clamp(raw, 0, 100)
//
// Nota: la pérdida de precisión en info weight (0.5 truncado a int) es
// aceptable para el PoC; se documenta en Tarea 1.7 del plan. Si necesitas
// resolución 0.5 en UI, expón el campo como float en una segunda iteración.
package grader

import "github.com/adkd/adkd/internal/core/types"

func Score(findings []types.Finding) (int, types.Summary) {
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
	deduction := s.Errors*5 + s.Warnings*2 + int(float64(s.Info)*0.5)
	raw := 100 - deduction
	if raw > 100 {
		raw = 100
	}
	if raw < 0 {
		raw = 0
	}
	return raw, s
}
