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
	"strings"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
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
	renderReport(r, w, hasTty, false)
}

// RenderSummary imprime solo el resumen ejecutivo y los clusters más problemáticos.
func RenderSummary(r types.Report, w io.Writer, hasTty bool) {
	renderReport(r, w, hasTty, true)
}

func renderReport(r types.Report, w io.Writer, hasTty bool, summaryOnly bool) {
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
		// pickColor returns "" without a TTY, so the reset code must be
		// suppressed too - otherwise a bare ESC[0m leaked into piped output.
		reset(hasTty),
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

	if summaryOnly {
		renderTopClusters(w, byCluster, clusters, hasTty, c)
		return
	}

	for _, cl := range clusters {
		fmt.Fprintf(w, "\n[%s] %d issues\n", cl, len(byCluster[cl]))
		for _, f := range byCluster[cl] {
			sev := pickSev(f.Severity, hasTty)
			// Always show the rule id: it is what the user needs to silence or
			// re-severity the rule in kdoctor.config.yaml. It only appeared before
			// by accident, because native detectors repeat it inside Message -
			// detekt findings showed no id at all.
			fmt.Fprintf(w, "  %s %s  %s:%d:%d\n",
				sev, f.ID, f.File, f.Line, f.Column)
			fmt.Fprintf(w, "       %s\n", stripIDPrefix(f.Message, f.ID))
			if f.FixHint != "" {
				fmt.Fprintf(w, "       \u2192 %s\n", f.FixHint)
			}
		}
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, c(ansiCyan, "💡 Tip: Run 'kdoctor scan --html' for interactive web report or 'kdoctor scan --md' for Markdown."))
}

func renderTopClusters(w io.Writer, byCluster map[string][]types.Finding, clusters []string, hasTty bool, c func(color, s string) string) {
	if len(clusters) == 0 {
		return
	}
	fmt.Fprintln(w, "\nTop clusters:")

	// Ordenar por cantidad de issues (descendente)
	sorted := make([]string, len(clusters))
	copy(sorted, clusters)
	sort.SliceStable(sorted, func(i, j int) bool {
		return len(byCluster[sorted[i]]) > len(byCluster[sorted[j]])
	})

	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		cl := sorted[i]
		fmt.Fprintf(w, "  %d. [%s] %d issues\n", i+1, cl, len(byCluster[cl]))
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

// reset returns the ANSI reset sequence only when colours were actually
// emitted.
func reset(hasTty bool) string {
	if !hasTty {
		return ""
	}
	return ansiReset
}

// stripIDPrefix drops a leading "<id>: " from a message so the id is not
// printed twice now that it has its own column. Native detectors build their
// messages that way; detekt messages do not.
func stripIDPrefix(message, id string) string {
	if id == "" {
		return message
	}
	return strings.TrimPrefix(message, id+": ")
}
