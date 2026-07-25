// kdoctor scan: comando principal. Escanea el proyecto y calcula Health Score.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/baseline"
	"github.com/adkd/adkd/internal/core/config"
	"github.com/adkd/adkd/internal/core/detektrunner"
	"github.com/adkd/adkd/internal/core/diff"
	"github.com/adkd/adkd/internal/core/grader"
	"github.com/adkd/adkd/internal/core/rulemap"
	"github.com/adkd/adkd/internal/core/rules"
	"github.com/adkd/adkd/internal/core/sarif"
	"github.com/adkd/adkd/internal/core/types"
	"github.com/adkd/adkd/internal/mobiai"
	"github.com/adkd/adkd/internal/reporter/console"
	htmlrep "github.com/adkd/adkd/internal/reporter/html"
	jsonrep "github.com/adkd/adkd/internal/reporter/jsonreporter"
	mdrep "github.com/adkd/adkd/internal/reporter/markdown"
	sarifrep "github.com/adkd/adkd/internal/reporter/sarif"
)

// ErrFailBelow se devuelve desde runScan cuando el Health Score cae por
// debajo del umbral configurado en --fail-below. Cobra lo imprimirá en
// stderr y devolverá exit code != 0 automáticamente; los defers corren.
var ErrFailBelow = errors.New("health score below fail-below threshold")

// validateOutputFlags asegura que los formatos de salida sean mutuamente
// excluyentes y no se combinen de forma inconsistente.
func validateOutputFlags(f *scanFlags) error {
	var formats int
	if f.asJSON {
		formats++
	}
	if f.asSARIF {
		formats++
	}
	if f.asMD {
		formats++
	}
	if formats > 1 {
		return fmt.Errorf("only one output format can be used among --json, --sarif, --md")
	}
	if f.summary && (f.asJSON || f.asSARIF || f.asMD) {
		// Summary modifica la salida de consola; es compatible con --md en
		// modo "resumen markdown". Para JSON/SARIF no tiene sentido.
		if f.asJSON || f.asSARIF {
			return fmt.Errorf("--summary cannot be combined with --json or --sarif")
		}
	}
	return nil
}

type scanFlags struct {
	asJSON            bool
	asSARIF           bool
	projectType       string
	preferStandalone  bool
	detektBin         string
	projectDir        string
	failBelow         int
	outputPath        string
	diffRef           string
	baselinePath      string
	mobiai            bool
	mobiaiURL         string
	mobiaiToken       string
	mobiaiFailOnError bool
	asMD              bool
	summary           bool
	verbose           bool
	updateRules       bool
	asHTML            bool
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
--out path          : escribir a fichero en lugar de stdout
--diff ref          : filtrar por líneas modificadas/añadidas respecto al git ref
--baseline path     : suprimir findings listados en el baseline.xml
--md                : generar kdoctor-report.md en el directorio del proyecto
--html              : generar kdoctor-report.html interactivo en el directorio del proyecto
--summary           : mostrar solo resumen ejecutivo y top clusters
--update-rules      : actualizar reglas desde GitHub antes de escanear
--verbose           : mostrar salida de detekt (por defecto se ocultan los warnings JVM)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputFlags(f); err != nil {
				return err
			}
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
	cmd.Flags().StringVar(&f.diffRef, "diff", "", "filter findings to only those added/modified compared to the git reference")
	cmd.Flags().StringVar(&f.baselinePath, "baseline", "", "suppress findings listed in the specified baseline.xml")
	cmd.Flags().BoolVar(&f.mobiai, "mobiai", false, "output findings as JSONL to .mobiai/graph/findings.jsonl")
	cmd.Flags().StringVar(&f.mobiaiURL, "mobiai-url", os.Getenv("KDOCTOR_MOBIAI_URL"), "MobiAI Graph endpoint URL (also env KDOCTOR_MOBIAI_URL)")
	cmd.Flags().StringVar(&f.mobiaiToken, "mobiai-token", os.Getenv("KDOCTOR_MOBIAI_TOKEN"), "MobiAI Graph bearer token (also env KDOCTOR_MOBIAI_TOKEN)")
	cmd.Flags().BoolVar(&f.mobiaiFailOnError, "mobiai-fail-on-error", false, "fail the scan if the MobiAI Graph upload fails")
	cmd.Flags().BoolVar(&f.asMD, "md", false, "generate kdoctor-report.md in the project directory")
	cmd.Flags().BoolVar(&f.asHTML, "html", false, "generate interactive kdoctor-report.html in the project directory")
	cmd.Flags().BoolVar(&f.summary, "summary", false, "show only summary and top clusters")
	cmd.Flags().BoolVar(&f.updateRules, "update-rules", false, "update rules catalog from remote GitHub before scanning")
	cmd.Flags().BoolVar(&f.verbose, "verbose", false, "show detekt output (default hides JVM warnings)")
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

	if f.updateRules {
		if _, _, err := rulemap.FetchLatestRules(""); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to update rules from remote: %v\n", err)
		}
	}

	// 1. Cargar reglas (cascada offline-first: proyecto > cache usuario > local > embedded).
	loadResult, err := rulemap.LoadRulesCascade(wd, "")
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}
	ruleCatalog := loadResult.Rules
	if f.verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d rules from %s (%s)\n", len(ruleCatalog), loadResult.Source, loadResult.Path)
	}

	// 2. Detectar modo y correr Detekt.
	mode := detektrunner.Detect(wd, f.preferStandalone)
	sarifPath := filepath.Join(os.TempDir(), "kdoctor-detekt.sarif")
	// Ocultar por defecto la salida del subproceso detekt para evitar que
	// los warnings del JVM (sun.misc.Unsafe) contaminen el reporte. El flag
	// --verbose permite volver a mostrar esa salida.
	var detektOut io.Writer
	if f.verbose {
		detektOut = cmd.OutOrStdout()
	} else {
		detektOut = io.Discard
	}
	if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
		ProjectDir:     wd,
		SARIFOutput:    sarifPath,
		UseStandalone:  mode == detektrunner.ModeStandalone,
		StandalonePath: f.detektBin,
		Stdout:         detektOut,
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

	// 4. Correr detectores regex nativos en Go.
	nativeFindings, err := rules.RunRegexDetectors(wd, ruleCatalog)
	if err != nil {
		return fmt.Errorf("run native rules: %w", err)
	}
	raw = append(raw, nativeFindings...)

	// 5. Mapear Detekt IDs y IDs nativos → kdoctor.
	idx := rulemap.BuildIndex(ruleCatalog)
	mapped := idx.Map(raw)

	// 5a. Aplicar overrides de kdoctor.config.yaml
	configPath := filepath.Join(wd, "kdoctor.config.yaml")
	cfg, err := config.Load(configPath)
	if err == nil {
		mapped = rulemap.ApplyOverrides(mapped, cfg.Excludes, cfg.Rules)
	}

	// 5b. Baseline suppression
	if f.baselinePath != "" {
		baselineIDs, err := baseline.LoadBaseline(f.baselinePath)
		if err != nil {
			return fmt.Errorf("load baseline: %w", err)
		}
		var filtered []types.Finding
		for _, finding := range mapped {
			// Tarea #5 del round-2: pass wd so pathutil.Join can absolutize
			// baseline's relative paths against the scanned project. The
			// round-1 wrapper (projectRoot="") is kept for backward compat.
			if !baseline.IsSuppressedWithRoot(finding, baselineIDs, wd) {
				filtered = append(filtered, finding)
			}
		}
		mapped = filtered
	}

	// 5c. Diff filtering
	if f.diffRef != "" {
		baseRef, err := diff.GetMergeBase(f.diffRef, wd)
		if err != nil {
			return fmt.Errorf("diff merge-base: %w", err)
		}
		diffMap, err := diff.GetLineDiff(baseRef, wd)
		if err != nil {
			return fmt.Errorf("diff line ranges: %w", err)
		}
		// Tarea #5 del round-2: detect the actual git root so diff paths
		// (which `git diff` emits relative to git root, not to wd) line up
		// with the normalized finding.File paths. If wd is not inside a git
		// repo (e.g. detached tarball review) GetGitRoot errors out, in
		// which case we fall back to wd for backward compatibility.
		projectRootForFilter := wd
		if gitRoot, gitErr := diff.GetGitRoot(wd); gitErr == nil {
			projectRootForFilter = gitRoot
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"warning: could not detect git root (--project-dir=%s): %v\n"+
					"  falling back to --project-dir for diff-path normalization; "+
					"results on monorepo submodules may be imprecise.\n",
				wd, gitErr)
		}
		mapped = diff.FilterFindingsByDiffWithRoot(mapped, diffMap, projectRootForFilter)
	}

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
	case f.asMD:
		return renderMarkdown(report, wd, target, f.outputPath, f.summary)
	case f.asHTML:
		htmlTarget := target
		outPath := f.outputPath
		if outPath == "" {
			outPath = filepath.Join(wd, "kdoctor-report.html")
			fHtml, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create html report file: %w", err)
			}
			defer func() { _ = fHtml.Close() }()
			htmlTarget = fHtml
		}
		if err := htmlrep.RenderHTML(report, htmlTarget); err != nil {
			return fmt.Errorf("render html report: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Interactive HTML web report generated: %s\n", outPath)
		return nil
	default:
		hasTty := isTerminal(cmd.OutOrStdout())
		if f.summary {
			console.RenderSummary(report, target, hasTty)
		} else {
			console.RenderReport(report, target, hasTty)
		}
	}

	// 7b. Emitir a MobiAI graph si se solicitó.
	mobiaiEnabled := f.mobiai || f.mobiaiURL != ""
	if mobiaiEnabled {
		mobiaiDir := filepath.Join(wd, ".mobiai", "graph")
		if err := os.MkdirAll(mobiaiDir, 0755); err != nil {
			return fmt.Errorf("create mobiai dir: %w", err)
		}
		mobiaiFile := filepath.Join(mobiaiDir, "findings.jsonl")
		fOut, err := os.Create(mobiaiFile)
		if err != nil {
			return fmt.Errorf("create mobiai output file: %w", err)
		}
		defer func() { _ = fOut.Close() }()
		enc := json.NewEncoder(fOut)
		for _, finding := range report.Findings {
			if err := enc.Encode(finding); err != nil {
				return fmt.Errorf("encode finding for mobiai: %w", err)
			}
		}

		// Upload to MobiAI Graph if an endpoint was configured.
		if f.mobiaiURL != "" {
			c := mobiai.New(f.mobiaiURL, f.mobiaiToken)
			if err := c.UploadFindings(context.Background(), wd, report.Findings); err != nil {
				if f.mobiaiFailOnError {
					return fmt.Errorf("mobiai upload: %w", err)
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: mobiai upload failed: %v\n", err)
			}
		}
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

func renderMarkdown(report types.Report, wd string, target io.Writer, outputPath string, summary bool) error {
	write := func(w io.Writer) error {
		if summary {
			return mdrep.WriteSummary(report, w)
		}
		return mdrep.Write(report, w)
	}

	if outputPath != "" {
		if err := write(target); err != nil {
			return fmt.Errorf("write markdown report: %w", err)
		}
		// Confirmation va al stdout del comando, no al archivo de salida.
		fmt.Fprintf(os.Stdout, "Markdown report written to %s\n", outputPath)
		return nil
	}

	path := filepath.Join(wd, "kdoctor-report.md")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create markdown report: %w", err)
	}
	defer func() { _ = f.Close() }()
	if err := write(f); err != nil {
		return fmt.Errorf("write markdown report: %w", err)
	}
	fmt.Fprintf(target, "Markdown report written to %s\n", path)
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
