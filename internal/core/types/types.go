// Package types define el contrato canónico de datos de adkd.
//
// SchemaVersion 3 para alinear con react-doctor y tools downstream.
//
// IMPORTANTE (Phase 2-5): Los tipos BaselineFinding, DiffEngine, PromptContext,
// FixMode y MobiaiAnnotation son contrato adelantado del plan
// (docs/superpowers/plans/2026-07-19-adkd-implementation-plan.md).
// No eliminar sin reemplazarlos por las implementaciones de Fase 2-5:
//   - BaselineFinding: Tarea 2.3 (suppression via baseline.xml).
//   - DiffEngine: Tarea 2.2 (--diff <ref> con git).
//   - PromptContext, FixMode: Tarea 3.1-3.5 (Quality-Focused prompt builder).
//   - MobiaiAnnotation: Tarea 4.3 (vuelco como anotaciones a .mobiai/graph/).
package types

const SchemaVersion = "3"

// Severity clasifica el peso didáctico/operativo de un Finding.
// Mapeable directamente a SARIF level (MapSARIFLevel en internal/core/sarif).
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Rule es la representación interna de una regla del catálogo.
// Se hidrata desde rules/metadata.json (ver Tarea 1.3 y Tarea 1.6).
type Rule struct {
	ID         string   `json:"id"`
	Cluster    string   `json:"cluster"`
	Severity   Severity `json:"severity"`
	DetektRule string   `json:"detektRule,omitempty"`
	Status     string   `json:"status"` // "live" | "planned"
	FixHint    string   `json:"fixHint,omitempty"`
	DocURL     string   `json:"docUrl,omitempty"`
}

// Finding es la salida de cada regla individual aplicada a un punto del código.
type Finding struct {
	ID       string   `json:"id"`
	Cluster  string   `json:"cluster"`
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Message  string   `json:"message"`
	FixHint  string   `json:"fixHint,omitempty"`
	DocURL   string   `json:"docUrl,omitempty"`
}

type Summary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

type Report struct {
	SchemaVersion string    `json:"schemaVersion"`
	ProjectType   string    `json:"projectType"` // android|kmp|cmp
	HealthScore   int       `json:"healthScore"`
	Summary       Summary   `json:"summary"`
	Findings      []Finding `json:"findings"`
}

// BaselineFinding es un Finding suprimido por baseline.xml (Fase 2).
// El campo Fingerprint debe coincidir con el cálculo estable definido
// en internal/core/baseline.
type BaselineFinding struct {
	Fingerprint string `xml:"ID,attr"`
	File        string `xml:"file,attr"`
	Rule        string `xml:"rule,attr"`
	Severity    string `xml:"severity,attr"`
}

// DiffEngine consulta a Git qué archivos cambiaron desde un ref (Fase 2).
// Lo consume Task 1.10 cuando se especifique --diff=<ref>.
type DiffEngine interface {
	// ChangedSince devuelve paths relativos a la raíz del repo
	// que han cambiado desde el ref (ej. "main", "HEAD~1").
	ChangedSince(ref string) ([]string, error)
}

// PromptContext agrupa el material que el prompt builder recibe
// para construir un prompt Quality-Focused de un solo paso (sin RCI).
// Usado en Fase 3.
type PromptContext struct {
	Role       string   // "Senior Android engineer…"
	ArchBound  []string // ["Estás en commonMain: no uses androidMain …"]
	Skeleton   string   // firmas públicas del archivo y dependencias
	Finding    Finding
	FixHint    string
	ExtraRules []string // directivas de calidad (ej. "no magic strings")
}

// FixMode define los tres modos del fixer. Se serializa snake_case en flags.
type FixMode string

const (
	FixModeSuggest     FixMode = "suggest"     // default: genera fixes.md, no toca código
	FixModeInteractive FixMode = "interactive" // pregunta por cada fix
	FixModeAuto        FixMode = "auto"        // aplica todo + valida con patch guard
)

// MobiaiAnnotation es la unidad volcada a .mobiai/graph/findings.jsonl
// (Fase 4). Tiene el shape que espera el compat-layer de MobiAI Graph.
type MobiaiAnnotation struct {
	URI       string `json:"uri"`
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startColumn"`
	RuleID    string `json:"ruleId"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

func (f Finding) ToMobiaiAnnotation() MobiaiAnnotation {
	return MobiaiAnnotation{
		URI:       f.File,
		StartLine: f.Line,
		StartCol:  f.Column,
		RuleID:    f.ID,
		Severity:  string(f.Severity),
		Message:   f.Message,
	}
}
