// kdoctor scan: comando principal. Escanea el proyecto y calcula Health Score.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/detektrunner"
	"github.com/adkd/adkd/internal/core/grader"
	"github.com/adkd/adkd/internal/core/rulemap"
	"github.com/adkd/adkd/internal/core/sarif"
	"github.com/adkd/adkd/internal/core/types"
	"github.com/adkd/adkd/internal/reporter/console"
	jsonrep "github.com/adkd/adkd/internal/reporter/jsonreporter"
	sarifrep "github.com/adkd/adkd/internal/reporter/sarif"
)

// ErrFailBelow se devuelve desde runScan cuando el Health Score cae por
// debajo del umbral configurado en --fail-below. Cobra lo imprimirá en
// stderr y devolverá exit code != 0 automáticamente; los defers corren.
var ErrFailBelow = errors.New("health score below fail-below threshold")

type scanFlags struct {
	asJSON           bool
	asSARIF          bool
	projectType      string
	preferStandalone bool
	detektBin        string
	projectDir       string
	failBelow        int
	outputPath       string
}

func NewScanCmd() *cobra.Command {
	f := &scanFlags{}
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the project and compute Health Score",
		Long: `kdoctor scan corre el análisis estático (Detekt SARIF), lo mapa a las
reglas de kdoctor, calcula el Health Score 0-100 y emite el reporte.

Por defecto: rich console.
--json     : JSON schema v3 (para CI / MobiAI Graph)
--sarif    : SARIF 2.1.0 (para GitHub Code Scanning)
--prefer-standalone : usa el binario ` + "`detekt`" + ` si está en PATH, no gradlew
--detekt-bin path   : ruta explícita al binario/cmd/jar de detekt (si no en PATH)
--project-dir path  : directorio del proyecto a escanear (default: cwd)
--fail-below N      : exit code !=0 si score < N
--out path          : escribir a fichero en lugar de stdout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, f)
		},
	}
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "output JSON instead of console")
	cmd.Flags().BoolVar(&f.asSARIF, "sarif", false, "output SARIF 2.1.0 (GitHub Code Scanning)")
	cmd.Flags().StringVar(&f.projectType, "type", "android", "project type: android|kmp|cmp")
	cmd.Flags().BoolVar(&f.preferStandalone, "prefer-standalone", false, "use standalone detekt binary if available")
	cmd.Flags().StringVar(&f.detektBin, "detekt-bin", "", "explicit path to detekt binary (overrides PATH lookup)")
	cmd.Flags().StringVar(&f.projectDir, "project-dir", "", "project directory to scan (default: cwd)")
	cmd.Flags().IntVar(&f.failBelow, "fail-below", 0, "non-zero exit code if health score is below this value")
	cmd.Flags().StringVar(&f.outputPath, "out", "", "write report to file instead of stdout")
	return cmd
}

func runScan(cmd *cobra.Command, f *scanFlags) error {
	wd := f.projectDir
	if wd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
		wd = cwd
	}

	// 1. Cargar reglas (single source of truth).
	rulesPath, err := resolveRulesPath()
	if err != nil {
		return err
	}
	rules, err := rulemap.LoadRules(rulesPath)
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}

	// 2. Detectar modo y correr Detekt.
	mode := detektrunner.Detect(wd, f.preferStandalone)
	sarifPath := filepath.Join(os.TempDir(), "kdoctor-detekt.sarif")
	out := cmd.OutOrStdout()
	if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
		ProjectDir:     wd,
		SARIFOutput:    sarifPath,
		UseStandalone:  mode == detektrunner.ModeStandalone,
		StandalonePath: f.detektBin,
		Stdout:         out,
	}); err != nil {
		return fmt.Errorf("detekt: %w", err)
	}

	// 3. Parsear SARIF.
	file, err := os.Open(sarifPath)
	if err != nil {
		return fmt.Errorf("open sarif %s: %w", sarifPath, err)
	}
	defer func() { _ = file.Close() }()
	raw, err := sarif.Parse(file)
	if err != nil {
		return fmt.Errorf("parse sarif: %w", err)
	}

	// 4. Mapear Detekt IDs → kdoctor.
	idx := rulemap.BuildIndex(rules)
	mapped := idx.Map(raw)

	// 5. Calcular Health Score.
	score, sum := grader.Score(mapped)
	report := types.Report{
		SchemaVersion: types.SchemaVersion,
		ProjectType:   f.projectType,
		HealthScore:   score,
		Summary:       sum,
		Findings:      mapped,
	}

	// 6. Seleccionar destino de salida.
	target := cmd.OutOrStdout()
	if f.outputPath != "" {
		f2, err := os.Create(f.outputPath)
		if err != nil {
			return err
		}
		defer func() { _ = f2.Close() }()
		target = f2
	}

	// 7. Emitir en el formato pedido.
	switch {
	case f.asJSON:
		return jsonrep.Write(report, target)
	case f.asSARIF:
		return sarifrep.Write(report, target)
	default:
		hasTty := isTerminal(cmd.OutOrStdout())
		console.RenderReport(report, target, hasTty)
	}

	// 8. Quality gate.
	if f.failBelow > 0 && score < f.failBelow {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"\n\u00d7 Health Score %d < fail-below %d\n",
			score, f.failBelow)
		return ErrFailBelow
	}
	return nil
}

// isTerminal detecta si el writer es un terminal real. Evita contaminar
// pipes con códigos ANSI. Para Fase 2 podemos sustituir por go-isatty
// si queremos detección cross-platform robusta (Windows specifics).
func isTerminal(w any) bool {
	if w == nil {
		return false
	}
	if f, ok := w.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
	return false
}
