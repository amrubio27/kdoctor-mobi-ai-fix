// Package console renderiza el reporte kdoctor con colores ANSI en la terminal.
//
// Detecta isatty para no contaminar pipes; en pipe mode, sin colores.
// Fase 2 podemos cambiar a charmbracelet/lipgloss para un TUI más rico;
// por ahora ANSI básico cubre todos los colores del score 0-100.
package console

import (
	"fmt"
	"io"
	"sort"

	"github.com/adkd/adkd/internal/core/types"
)

const (
	ansiReset   = "\033[0m"
	ansiRed     = "\033[31m"
	ansiBoldRed = "\033[31;1m"
	ansiYellow  = "\033[33m"
	ansiCyan    = "\033[36m"
	ansiGreen   = "\033[32m"
)

// RenderReport imprime el Report en `w` con formato rich.
// Si el writer parece no ser TTY (hasTty=false), se omiten los colores.
func RenderReport(r types.Report, w io.Writer, hasTty bool) {
	c := func(color, s string) string {
		if !hasTty {
			return s
		}
		return color + s + ansiReset
	}
	fmt.Fprintf(w, "%s %s%d/100%s\n",
		c(ansiGreen, "Health Score:"),
		pickColor(r.HealthScore, hasTty),
		r.HealthScore,
		ansiReset,
	)
	fmt.Fprintf(w, "%d errors  ·  %d warnings  ·  %d info  ·  %d total\n",
		r.Summary.Errors, r.Summary.Warnings, r.Summary.Info, r.Summary.Total)

	if len(r.Findings) == 0 {
		fmt.Fprintln(w, c(ansiGreen, "✓ No issues found."))
		return
	}

	byCluster := map[string][]types.Finding{}
	for _, f := range r.Findings {
		byCluster[f.Cluster] = append(byCluster[f.Cluster], f)
	}
	clusters := make([]string, 0, len(byCluster))
	for cl := range byCluster {
		clusters = append(clusters, cl)
	}
	sort.Strings(clusters)
	for _, cl := range clusters {
		fmt.Fprintf(w, "\n[%s] %d issues\n", cl, len(byCluster[cl]))
		for _, f := range byCluster[cl] {
			sev := pickSev(f.Severity, hasTty)
			fmt.Fprintf(w, "  %s %s:%d:%d  %s\n",
				sev, f.File, f.Line, f.Column, f.Message)
			if f.FixHint != "" {
				fmt.Fprintf(w, "    \u2192 %s\n", f.FixHint)
			}
		}
	}
}

func pickColor(score int, hasTty bool) string {
	if !hasTty {
		return ""
	}
	switch {
	case score >= 90:
		return ansiGreen
	case score >= 75:
		return ansiCyan
	case score >= 50:
		return ansiYellow
	case score >= 25:
		return ansiRed
	}
	return ansiBoldRed
}

func pickSev(s types.Severity, hasTty bool) string {
	if !hasTty {
		return string(s)
	}
	switch s {
	case types.SeverityError:
		return ansiRed + "err " + ansiReset
	case types.SeverityWarning:
		return ansiYellow + "warn" + ansiReset
	case types.SeverityInfo:
		return ansiCyan + "info" + ansiReset
	}
	return string(s)
}
