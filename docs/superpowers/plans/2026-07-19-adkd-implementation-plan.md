# adkd (Android / KMP / CMP Doctor AI Fix) — Plan Maestro de Implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construir `adkd`, un CLI nativo estilo `react-doctor` para Android / KMP / CMP, capaz de escanear repos Kotlin, asignar un **Health Score 0–100** sobre un catálogo inicial de ~40 reglas (subset crítico de las 64 que define la V1), ingestar el reporte SARIF de Detekt, ejecutar fixes mediante un AI-Fixer agnóstico al LLM (Claude Code, Cursor, MobiAI…), y distribuirse como skill instalable de MobiAI.

**Architecture:** **CLI en Go + Cobra** que lanza `./gradlew detekt` (o el binario `detekt` directamente) en un subproceso, parsea su salida SARIF 2.1 en Go, mapea las reglas Detekt a las reglas `adkd` (definidas como `cluster/rule-id` con severidad y remediation), calcula un Health Score 0–100 con la fórmula V1 (errores·5 + warnings·2 + info·0.5, acotado 0–100), lo reporta por consola rich, JSON y SARIF, y expone un fixer calidad-prompt sin RCI que delega al LLM detectado. La integración con MobiAI es un `SKILL.md` + `plugin.json` que el `mobiai` CLI puede instalar como pack independiente, y `adkd scan --mobiai` vuelca los findings al `Graph` semántico cuando existe.

**Tech Stack:**
- Go 1.22+ (CLI, SARIF parser, rule engine) — `github.com/spf13/cobra`, `github.com/charmbracelet/huh` (UI) o `github.com/charmbracelet/lipgloss` + `bubbletea` para rich console.
- Detekt 1.23.x (binario standalone) → SARIF 2.1 → consumible por Go.
- Android Lint (`./gradlew lint`) opcionalmente en Fase 2.
- AI providers (en Fase 3): `Claude Code` CLI por `exec.Command`, `cursor-agent` CLI, `gemini` CLI, `codex` CLI; MobiAI como meta-bridge en Fase 4.
- HTML dashboard: Vite + Tailwind (estática, sin servidor).
- Tests: Go `testing` + fixtures de proyectos Android reales en `examples/bad-project`, `examples/good-project`.
- Distribución: GoReleaser v2 (igual que MobiAI) → binarios para 6 OS/arch + scripts `install.sh`/`install.cmd`/`install.ps1`.

---

## Global Constraints

- **Lenguaje CLI:** Go 1.22+, sin node, sin cgo en Fase 1–4. Si una Fase requiere compilador Kotlin embedido (5.x), decidir tras spike.
- **Licencia:** MIT, alineada con react-doctor y MobiAI. `LICENSE` en raíz desde el primer commit.
- **Naming binario:** Siempre `adkd`. Comando primario `adkd scan`. La herramienta NO se llama `doctor-android`, `kmp-doctor`, etc.
- **Configuración del usuario:** `adkd.config.yaml` (YAML, no TS — V2 explícitamente dice evitar el ecosistema Node). Los campos clave son: `projectType` (android|kmp|cmp), `paths.kotlin`, `rules` (mapa `cluster/name: severity`), `score.failBelow`, `aiFixer.provider`, `aiFixer.mode`.
- **Contrato de datos:** El reporte JSON canónico es `Finding{ID, Cluster, Rule, Severity, File, Line, Column, Message, FixHint, DocURL}[]` con `HealthScore int`, `Summary {Errors, Warnings, Info}` y `SchemaVersion = "3"` (alineado con react-doctor para que tools downstream lo entiendan).
- **SARIF out:** SARIF 2.1.0 estricto para `--sarif`, apto para GitHub Code Scanning sin flags extra.
- **Repo layout:** Todo en un único monorepo Go con `cmd/adkd`, `internal/*`, `scripts/`, `examples/`, `docs/`, `.github/workflows/`, `plugin.json`, `SKILL.md`.
- **CI del propio `adkd`:** Tests Go + lint en cada PR a `main`. Los releases se firman vía tag `cli-v*` (convención heredada de MobiAI).
- **No rotura de la V1:** El catálogo de 64 reglas (en `ANDROID_DOCTOR_FIX_AI.md`) **debe** estar presente en `docs/rules/` con frontmatter (severidad, descripción, ejemplo malo, ejemplo bueno) **antes** de empezar a codificar reglas en Go. Si una regla no migrada aún, queda listada como `status: planned` — NUNCA se borra del catálogo sin decisión explícita.
- **V1 website se conserva:** `docs/website/index.html` se mantiene, **pero** se actualiza en Fase 3 para decir "Native Go" en lugar de "TypeScript" en los puntos donde aparece. No rehacer todo el sitio en este plan.
- **Código muerto:** No se acepta. Cada función pública tiene tests.
- **Frecuencia de commits:** Al final de cada Task, como pide `writing-plans`.

---

## Contexto estratégico (por qué este plan es como es)

### V1 (TS) → V2 (Go) → V3 (este plan)

| Decisión | V1 | V2 | V3 (lock) |
|---|---|---|---|
| Lenguaje CLI | TS/Node | Go + Cobra | **Go + Cobra** |
| Parser reglas | AST Kotlin custom | Detekt SARIF + K2 FIR (futuro) | **Detekt SARIF; K2 FIR como R&D en Fase 5** |
| Health Score | `err×5 + warn×2` absoluto | densidad por KLOC | **Fórmula V1** (gamificación importa, KLOC se introduce si hay feedback) |
| IA | bridge agnóstico | Quality-Focused single-pass (sin RCI) | **Quality-Focused single-pass** + dry-run obligatorio |
| CI/CD | score bloqueante | `--diff main` + `baseline.xml` | **`--diff main`** + `baseline.xml` (default score-fall como opt-in) |
| Indexado | parseo de `build.gradle.kts` | MobiAI Graph | **MobiAI Graph cuando exista; fallback regex propio** |
| Distribución | npx / brew / install.sh | binario standalone | **GoReleaser + install scripts (igual que MobiAI)** |

### Lo que NO se pierde
- Las **64 reglas** de V1 son el catálogo: vive en `docs/rules/` y `rules/metadata.json`.
- El comando **`adkd scan`** y la salida con Health Score vistosa.
- La página web `docs/website/index.html` (se reescribe en Fase 3).
- La integración con MobiAI como skill.

### Asunciones que se validan en Sprint 1
- Detectkt 1.23+ cubre al menos el **60%** de las 64 reglas nativamente (probablemente ~40). Las ausentes se marcan `status: planned` y se priorizan luego.
- `./gradlew detekt` es lo bastante rápido en CI con Gradle Daemon. Si peca (>30s en repos pequeños), se investiga el binario standalone.
- Claude Code CLI es el provider prioritario en Fase 3; Cursor y Gemini quedan como `Phase 3.x nice-to-have`.

---

## Estructura del Monorepo (objetivo final)

```text
android-kmp-doctor-ai-fix/
├── cmd/adkd/main.go                 # entrypoint Cobra
├── internal/
│   ├── cli/                         # scan.go, fix.go, init.go, doctor.go, hook.go, ci.go
│   ├── core/
│   │   ├── types/                   # Finding, Rule, HealthScore, Summary
│   │   ├── sarif/                   # parser SARIF 2.1.0 estricto
│   │   ├── detektrunner/            # spawn detekt o ./gradlew detekt, captura SARIF
│   │   ├── rulemap/                 # Detekt ID → adkd rule + metadata
│   │   ├── grader/                  # cálculo Health Score
│   │   └── config/                  # adkd.config.yaml loading
│   ├── reporter/
│   │   ├── console/                 # rich TUI (lipgloss + huh)
│   │   ├── json/                    # Finding[] v3 JSON
│   │   └── sarif/                   # SARIF writer (re-exporta el ingest)
│   ├── aifixer/
│   │   ├── qualityprompt/           # templates V2 Quality-Focused
│   │   ├── provider/                # claude, cursor, gemini, mobiai
│   │   ├── patchguard/              # post-fix: kotlin syntax & arquivo size check
│   │   └── diff/                    # format-patch / apply
│   └── mobiai/
│       └── graphbridge/             # vuelca findings como anotaciones en Graph si existe
├── rules/
│   └── metadata.json                # las 64 reglas + status (live|planned)
├── examples/
│   ├── bad-project/                 # proyecto Android deliberadamente roto
│   ├── good-project/                # mismo proyecto tras adkd fix --ai
│   └── scoring-fixtures/            # JSON con Health Score esperado por proyecto
├── scripts/
│   ├── install.sh / install.cmd / install.ps1
│   └── ci.sh                        # boot CI con baseline + diff
├── docs/
│   ├── website/                     # landing (existente, se rehace en Fase 3)
│   ├── rules/                       # una .md por regla (frontmatter + ejemplos)
│   ├── superpowers/plans/           # (este archivo vive aquí)
│   └── v2/adkd-proposal-v2.md       # (ya creado desde el PDF)
├── .github/workflows/
│   ├── adkd-ci.yml                  # tests + lint Go
│   ├── adkd-action.yml              # reusable action (composite)
│   └── score-gate.yml               # ejemplo quality-gate
├── plugin.json                      # MobiAI skill descriptor
├── SKILL.md                         # instrucciones para LLMs (cargado por Cursor/Claude)
├── adkd.config.example.yaml
├── .goreleaser.yml                  # 6 OS/arch
├── go.mod, go.sum
├── LICENSE                          # MIT
├── README.md
├── CONTRIBUTING.md
└── ANDROID_DOCTOR_FIX_AI.md         # (existente, ahora como "V1 rules catalog")
```

---

# Fase 1 — Foundations (Semana 1–2)

**Objetivo:** `adkd scan` end-to-end sobre un susbset de 15 reglas críticas (Compose Performance + Lifecycle + Coroutines), con Health Score, salida consola + JSON, validado contra `examples/bad-project` y `examples/good-project` con fixtures.

### Task 1.1 — Inicializar el monorepo Go + Cobra + git

**Files:**
- Create: `go.mod`, `go.sum`, `cmd/adkd/main.go`, `cmd/adkd/main_test.go`
- Create: `.gitignore`, `LICENSE`, `README.md` mínimo

**Interfaces:**
- Produce: binary `adkd` con flag `--version` que imprime `adkd version 0.1.0`.

- [ ] **Step 1: Init repo git local y crear `.gitignore`**

```bash
cd "C:\\Users\\Miguel\\Desktop\\doctor mobi ai fix"
git init -b main
```

`.gitignore`:

```gitignore
# Binarios
/adkd
/dist/
*.exe

# Go
*.test
*.out
/vendor/

# Trabajo local
/.tmp/
/.mobiai/
/.worktrees/

# IDE
.idea/
.vscode/
*.iml

# OS
.DS_Store
Thumbs.db
```

- [ ] **Step 2: Crear módulo Go**

```bash
go mod init github.com/adkd/adkd
```

Versión Go objetivo: 1.22+. Añade `go 1.22` al `go.mod` si el `go mod init` pone una más baja.

- [ ] **Step 3: Añadir Cobra**

```bash
go get github.com/spf13/cobra@latest
```

- [ ] **Step 4: Escribir el entrypoint mínimo**

`cmd/adkd/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "adkd",
		Short: "Android / KMP / CMP Doctor with AI-driven fixes",
		Version: version,
	}
	root.AddCommand(newScanCmd()) // se añadirá en la Tarea 1.10
	root.AddCommand(newFixCmd())  // placeholder por ahora
	root.AddCommand(newInitCmd()) // placeholder por ahora
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Test del version flag**

`cmd/adkd/main_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(buf.String(), "0.1.0") {
		t.Fatalf("expected version in output, got %q", buf.String())
	}
}
```

- [ ] **Step 6: Compilar y testear**

```bash
go build ./... && go test ./... -run TestRootVersion
```

Esperado: PASS.

- [ ] **Step 7: Commit**

```bash
git add .gitignore go.mod go.sum cmd/ LICENSE README.md
git commit -m "feat: scaffold adkd CLI in Go with cobra and version flag"
```

### Task 1.2 — Definir el contrato de datos (`Finding`, `Rule`, `HealthScore`, `Summary`)

**Files:**
- Create: `internal/core/types/types.go`
- Create: `internal/core/types/types_test.go`

**Interfaces:**
- Produce: tipos que **toda** regla, reporter y fixer consumirán. SchemaVersion = 3 para alinear con react-doctor.

- [ ] **Step 1: Escribir tipos**

`internal/core/types/types.go`:

```go
// Package types define el contrato canónico de datos de adkd.
// SchemaVersion 3 para alinear con react-doctor y tools downstream.
package types

const SchemaVersion = "3"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Finding struct {
	ID       string   `json:"id"`        // "compose-remember-missing"
	Cluster  string   `json:"cluster"`   // "compose-performance"
	Rule     string   `json:"rule"`      // "remember-missing"
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
```

- [ ] **Step 2: Test mínimo contract**

`internal/core/types/types_test.go`:

```go
package types

import "testing"

func TestSchemaVersion(t *testing.T) {
	if SchemaVersion != "3" {
		t.Fatalf("schema version expected 3, got %q", SchemaVersion)
	}
}

func TestSeverityValues(t *testing.T) {
	cases := map[Severity]bool{
		SeverityError: true, SeverityWarning: true, SeverityInfo: true,
		"debug": false,
	}
	for sev, ok := range cases {
		if (sev == SeverityError || sev == SeverityWarning || sev == SeverityInfo) != ok {
			t.Fatalf("severity %q rejected unexpectedly", sev)
		}
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/core/types/...
```

Esperado: PASS.

- [ ] **Step 3b (P1 #5): Detekt standalone vs gradlew switch + filepath import en scan.go**

Ver Tarea 1.4 (refactor) y Tarea 1.10 (scan.go con filepath.Join import correcto).

- [ ] **Step 4: Commit**

```bash
git add internal/core/types/
git commit -m "feat(types): canonical Finding/Report contract schema v3"
```

### Task 1.3 — Reglas: el catálogo V1 → `rules/metadata.json` con 64 reglas + status

**Files:**
- Create: `rules/metadata.json` (estructura)
- Create: `scripts/genschema/main.go` (generador que convierte la tabla V1 a JSON)
- Create: `docs/rules/README.md` con el índice

**Interfaces:**
- Produce: JSON estable con las 64 reglas. Campos: `id`, `cluster`, `severity`, `description`, `detektRule` (cuando exista), `bad`, `good`, `status`.

Migración de la tabla V1 al JSON **literal**: cada fila de §5.1 a §5.10 del `ANDROID_DOCTOR_FIX_AI.md` se vuelca. Reglas que Detekt cubre nativamente → `status: live, detektRule: "style:MagicNumber"`. Las custom → `status: planned`.

- [ ] **Step 1: Volcar las 64 reglas en JSON (TODO único grande, 1 commit)**

`scripts/genschema/main.go`:

```go
// genschema vuelca las 64 reglas de V1 a JSON. Ejecutar con `go run ./scripts/genschema`
package main

import (
	"encoding/json"
	"os"
)

type Rule struct {
	ID         string `json:"id"`
	Cluster    string `json:"cluster"`
	Severity   string `json:"severity"`
	DetektRule string `json:"detektRule,omitempty"`
	Status     string `json:"status"`
}

func main() {
	rules := []Rule{
		// 5.1 Compose Performance
		{ID: "compose-missing-key", Cluster: "compose-performance", Severity: "error", Status: "planned"},
		{ID: "compose-unstable-params", Cluster: "compose-performance", Severity: "error", Status: "planned"},
		{ID: "compose-derived-state-missing", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
		{ID: "compose-lambda-recomposition", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
		{ID: "compose-heavy-composable", Cluster: "compose-performance", Severity: "info", Status: "planned"},
		{ID: "compose-remember-missing", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:ReusedModifierInstance", Status: "live"},
		{ID: "compose-state-hoisting", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ModifierHeightWithText", Status: "live"},
		{ID: "compose-modifier-frequent-changes", Cluster: "compose-performance", Severity: "warning", DetektRule: "Compose:ReusedModifierInstance", Status: "live"},
		{ID: "compose-graphics-layer", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
		{ID: "compose-list-animated", Cluster: "compose-performance", Severity: "warning", Status: "planned"},
		{ID: "compose-side-effect-in-compose", Cluster: "compose-performance", Severity: "error", Status: "planned"},
		{ID: "compose-runtime-import-bleeding", Cluster: "compose-performance", Severity: "error", DetektRule: "Compose:ComposableNaming", Status: "live"},

		// 5.2 Coroutines & Async
		{ID: "coroutine-viewmodel-scope", Cluster: "coroutines", Severity: "error", Status: "planned"},
		{ID: "coroutine-global-scope", Cluster: "coroutines", Severity: "error", DetektRule: "GlobalCoroutineUsage", Status: "live"},
		{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: "info", Status: "planned"},
		{ID: "coroutine-supervisor-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
		{ID: "coroutine-unstructured-concurrency", Cluster: "coroutines", Severity: "warning", Status: "planned"},
		{ID: "coroutine-cancellation-leak", Cluster: "coroutines", Severity: "error", DetektRule: "CoroutineCancellation", Status: "live"},
		{ID: "coroutine-flow-buffer-missing", Cluster: "coroutines", Severity: "warning", Status: "planned"},
		{ID: "coroutine-sharedflow-replay", Cluster: "coroutines", Severity: "info", Status: "planned"},

		// 5.3 Lifecycle
		{ID: "lifecycle-context-leak", Cluster: "lifecycle", Severity: "error", Status: "planned"},
		{ID: "lifecycle-collect-as-state-missing", Cluster: "lifecycle", Severity: "error", Status: "planned"},
		{ID: "lifecycle-collect-lifecycle-aware", Cluster: "lifecycle", Severity: "warning", Status: "planned"},
		{ID: "lifecycle-ondestroy-listener", Cluster: "lifecycle", Severity: "warning", Status: "planned"},
		{ID: "lifecycle-job-not-cancelled", Cluster: "lifecycle", Severity: "error", Status: "planned"},
		{ID: "lifecycle-config-change-survival", Cluster: "lifecycle", Severity: "info", Status: "planned"},

		// 5.4 Memory
		{ID: "mem-bitmap-no-pool", Cluster: "memory", Severity: "warning", Status: "planned"},
		{ID: "mem-context-receiver-leak", Cluster: "memory", Severity: "error", Status: "planned"},
		{ID: "mem-static-context", Cluster: "memory", Severity: "error", Status: "planned"},
		{ID: "mem-handler-leak", Cluster: "memory", Severity: "warning", Status: "planned"},
		{ID: "mem-coroutine-job-leak", Cluster: "memory", Severity: "error", Status: "planned"},

		// 5.5 Architecture
		{ID: "arch-god-class", Cluster: "architecture", Severity: "warning", DetektRule: "TooManyFunctions", Status: "live"},
		{ID: "arch-circular-dep", Cluster: "architecture", Severity: "error", Status: "planned"},
		{ID: "arch-feature-module-public-api-bleed", Cluster: "architecture", Severity: "warning", Status: "planned"},
		{ID: "arch-public-api-mutable-state", Cluster: "architecture", Severity: "error", Status: "planned"},
		{ID: "arch-data-class-with-logic", Cluster: "architecture", Severity: "warning", Status: "planned"},
		{ID: "arch-named-arg-required", Cluster: "architecture", Severity: "info", Status: "planned"},
		{ID: "arch-utility-function-should-be-extension", Cluster: "architecture", Severity: "info", Status: "planned"},
		{ID: "arch-internal-in-public-api", Cluster: "architecture", Severity: "error", DetektRule: "InvalidPackageDeclaration", Status: "live"},
		{ID: "arch-package-cycles-kmp", Cluster: "architecture", Severity: "error", Status: "planned"},
		{ID: "arch-presentation-depends-on-data", Cluster: "architecture", Severity: "error", Status: "planned"},

		// 5.6 Accessibility
		{ID: "a11y-content-description", Cluster: "accessibility", Severity: "error", Status: "planned"},
		{ID: "a11y-click-target-size", Cluster: "accessibility", Severity: "warning", Status: "planned"},
		{ID: "a11y-merged-clickable", Cluster: "accessibility", Severity: "info", Status: "planned"},
		{ID: "a11y-talkback-label-missing", Cluster: "accessibility", Severity: "warning", Status: "planned"},
		{ID: "a11y-color-contrast-note", Cluster: "accessibility", Severity: "info", Status: "planned"},

		// 5.7 Testing
		{ID: "test-public-api-without-test", Cluster: "testing", Severity: "warning", Status: "planned"},
		{ID: "test-flaky-test-marker", Cluster: "testing", Severity: "info", Status: "planned"},
		{ID: "test-hilt-rule-missing", Cluster: "testing", Severity: "error", Status: "planned"},
		{ID: "test-runblocking-in-test", Cluster: "testing", Severity: "warning", Status: "planned"},
		{ID: "test-compose-test-rule-missing", Cluster: "testing", Severity: "warning", Status: "planned"},

		// 5.8 Security
		{ID: "sec-hardcoded-secret", Cluster: "security", Severity: "error", DetektRule: "HardcodedPassword", Status: "live"},
		{ID: "sec-log-pii", Cluster: "security", Severity: "error", Status: "planned"},
		{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: "error", Status: "planned"},
		{ID: "sec-deeplink-no-validation", Cluster: "security", Severity: "warning", Status: "planned"},
		{ID: "sec-fragment-injection", Cluster: "security", Severity: "error", Status: "planned"},

		// 5.9 KMP / CMP
		{ID: "kmp-platform-api-leak", Cluster: "kmp", Severity: "error", Status: "planned"},
		{ID: "kmp-expect-actual-violation", Cluster: "kmp", Severity: "error", Status: "planned"},
		{ID: "kmp-coroutines-supervisor-in-common", Cluster: "kmp", Severity: "warning", Status: "planned"},
		{ID: "kmp-compose-multiplatform-stable-required", Cluster: "kmp", Severity: "warning", Status: "planned"},

		// 5.10 Dead code
		{ID: "dead-unused-import", Cluster: "dead-code", Severity: "info", DetektRule: "UnusedImport", Status: "live"},
		{ID: "dead-unused-private-fun", Cluster: "dead-code", Severity: "info", DetektRule: "UnusedPrivateMember", Status: "live"},
		{ID: "dead-unused-parameter", Cluster: "dead-code", Severity: "warning", Status: "planned"},
		{ID: "dead-white-label-export", Cluster: "dead-code", Severity: "info", Status: "planned"},
	}
	if len(rules) != 64 {
		panic("rule count must be 64; check V1 spec")
	}
	out, _ := os.Create("rules/metadata.json")
	defer out.Close()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rules); err != nil {
		panic(err)
	}
}
```

- [ ] **Step 2: Generar y verificar**

```bash
go run ./scripts/genschema
jq 'length' rules/metadata.json
```

Esperado: `64`.

- [ ] **Step 3: Test del script**

`scripts/genschema/main_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGeneratedHas64(t *testing.T) {
	data, err := os.ReadFile("../../rules/metadata.json")
	if err != nil { t.Fatal(err) }
	var rules []map[string]any
	if err := json.Unmarshal(data, &rules); err != nil { t.Fatal(err) }
	if len(rules) != 64 {
		t.Fatalf("expected 64 rules, got %d", len(rules))
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add rules/ scripts/genschema/
git commit -m "feat(rules): port V1 64-rule catalog to metadata.json with status field"
```

> ⚠️ **Asunción a validar en Sprint 1:** el `detektRule` mapeado arriba es hipótesis. Por cada entrada revisar la tabla oficial de reglas de Detekt 1.23.x y de los paquetes comunitarios `appKODE/detekt-rules-compose`, y corregir las que no coincidan. La regla "como máximo 1 commit" se romperá en una segunda iteración cuando se audite.

### Task 1.4 — Detekt runner (subprocess manager)

**Files:**
- Create: `internal/core/detektrunner/{runner.go,detect.go,init.go,runner_test.go}`

**Interfaces:**
- `RunDetekt(ctx, opts) (sarifPath string, err error)` — invoca detekt CLI o `./gradlew detekt` según opts.UseStandalone, capturando SARIF en opts.SARIFOutput.
- `Detect(projectDir) ExecutionMode` — decide ExecutionMode: Standalone vs GradleWrapper según presencia de `./gradlew`, `detekt` en `$PATH` y `--prefer`.

Esta es la **decisión arquitectónica más sensible** de Fase 1. Empezamos con **Modo Dual** para amortiguar el Sprint 1: si `detekt` está en `$PATH`, lo usamos (rápido, sin Gradle Daemon); si no, fallback a `./gradlew detekt` (lento pero siempre disponible).

### Task 1.4b — init-script Gradle writer

**Files:**
- Create: `internal/core/detektrunner/init.go`
- Create: `internal/core/detektrunner/init_test.go`

**Interfaces:**
- `WriteInitScript(projectDir string) (string, error)` — escribe `adkd-detekt.init.gradle.kts` en `projectDir` y devuelve la ruta.

- [ ] **Step 1: Template del init-script**

```go
// internal/core/detektrunner/init.go
package detektrunner

import (
	"fmt"
	"os"
	"path/filepath"
)

const initScriptName = "adkd-detekt.init.gradle.kts"

// Plantilla equivalente al bloque Kotlin inline mostrado en Tarea 1.4.
// Se escribe a projectDir y ./gradlew lo carga con --init-script.
var initScriptTemplate = `// Generated by adkd — do not edit by hand.
allprojects {
	detekt {
		xml = false
		sarif {
			required = true
			output = rootProject.layout.buildDirectory
				.file("reports/detekt/adkd.sarif").get().asFile
		}
	}
}
`

func WriteInitScript(projectDir string) (string, error) {
	if projectDir == "" {
		return "", fmt.Errorf("projectDir required")
	}
	path := filepath.Join(projectDir, initScriptName)
	if err := os.WriteFile(path, []byte(initScriptTemplate), 0644); err != nil {
		return "", fmt.Errorf("write init-script: %w", err)
	}
	return path, nil
}
```

- [ ] **Step 2: Test que confirma contenido**

```go
// internal/core/detektrunner/init_test.go
package detektrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteInitScript(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteInitScript(dir)
	if err != nil { t.Fatal(err) }
	data, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(data), "adkd-detekt.init.gradle.kts") &&
		!strings.Contains(string(data), "allprojects") {
		t.Fatalf("template malformed: %q", string(data))
	}
	if filepath.Base(path) != initScriptName {
		t.Fatalf("filename %s != %s", path, initScriptName)
	}
}
```

- [ ] **Step 3: Modificar `runGradlew` para invocar `WriteInitScript` antes del exec**

```go
func runGradlew(ctx context.Context, opts Options) (string, error) {
	gradlew := filepath.Join(opts.ProjectDir, "gradlew")
	if _, err := os.Stat(gradlew); err != nil {
		return "", fmt.Errorf("gradlew no encontrado en %s: %w", opts.ProjectDir, err)
	}
	initPath, err := WriteInitScript(opts.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("escribir init-script: %w", err)
	}
	defer os.Remove(initPath) // cleanup post-run
	args := []string{"detekt", "--init-script", initPath}
	cmd := exec.CommandContext(ctx, gradlew, args...)
	cmd.Dir = opts.ProjectDir
	out := opts.Stdout
	if out == nil { out = io.Discard }
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("./gradlew detekt: %w", err)
	}
	// SARIF Gradle plugin escribe tipicamente a build/reports/detekt/adkd.sarif.
	// En multi-modulo (`:app`, `:core`, ...) puede aparecer en
	// <ProjectDir>/<module>/build/... Por eso hacemos find recursivo.
	return findProducedSARIF(opts.ProjectDir), nil
}

// findProducedSARIF busca recursivamente el primer *.sarif bajo
// <projectDir>/<**>/build/reports/detekt/. Devuelve "" si no hay ninguno.
func findProducedSARIF(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil { return "" }
	root := filepath.Join(abs, "build", "reports", "detekt")
	if !fileExists(root) {
		// fallback multi-module: <projectDir>/<module>/build/reports/detekt/*.sarif
		_ = filepath.WalkDir(filepath.Join(abs, "build"), func(path string, d os.DirEntry, err error) error {
			if err != nil { return nil }
			if d.IsDir() { return nil }
			if filepath.Ext(path) == ".sarif" { root = filepath.Dir(path); return filepath.SkipAll }
			return nil
		})
	}
	matches, _ := filepath.Glob(filepath.Join(root, "*.sarif"))
	if len(matches) == 0 { return "" }
	return matches[0]
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/core/detektrunner/
git commit -m "feat(detektrunner): write adkd-detekt.init.gradle.kts for gradlew path"
```

> Sin esta Tarea, si `Detect()` elige `ModeGradleWrap`, el comando fallará en silencio porque el init-script nunca existió.

- [ ] **Step 1: Detektrunner con switch**

```go
// internal/core/detektrunner/runner.go
package detektrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type ExecutionMode string

const (
	ModeStandalone  ExecutionMode = "standalone"
	ModeGradleWrap  ExecutionMode = "gradlew"
)

type Options struct {
	ProjectDir     string
	SARIFOutput    string
	UseStandalone  bool          // true = binario directo, false = ./gradlew detekt
	StandalonePath string        // binario detekt
	Stdout         io.Writer     // opcional, para spinners en CLI
}

func RunDetekt(ctx context.Context, opts Options) (string, error) {
	if opts.ProjectDir == "" {
		return "", fmt.Errorf("ProjectDir required")
	}
	if opts.SARIFOutput == "" {
		return "", fmt.Errorf("SARIFOutput required")
	}
	if opts.UseStandalone {
		return runStandalone(ctx, opts)
	}
	return runGradlew(ctx, opts)
}

func runStandalone(ctx context.Context, opts Options) (string, error) {
	bin := opts.StandalonePath
	if bin == "" { bin = "detekt" }
	args := []string{"--input", opts.ProjectDir, "--report", "sarif:" + opts.SARIFOutput}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = opts.ProjectDir
	out := opts.Stdout
	if out == nil { out = io.Discard }
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("detekt standalone: %w", err)
	}
	return opts.SARIFOutput, nil
}

func runGradlew(ctx context.Context, opts Options) (string, error) {
	gradlew := filepath.Join(opts.ProjectDir, "gradlew")
	if _, err := os.Stat(gradlew); err != nil {
		return "", fmt.Errorf("gradlew no encontrado en %s: %w", opts.ProjectDir, err)
	}
	// Detekt Gradle plugin escribe SARIF a build/reports/...
	// Forzamos el output vía init-script (ver ExampleInitScript abajo).
	args := []string{
		"detekt",
		"--init-script", "adkd-detekt.init.gradle.kts",
	}
	cmd := exec.CommandContext(ctx, gradlew, args...)
	cmd.Dir = opts.ProjectDir
	out := opts.Stdout
	if out == nil { out = io.Discard }
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("./gradlew detekt: %w", err)
	}
	return opts.SARIFOutput, nil
}
```

Y el init-script que `runGradlew` referencia, generado en Task 1.4b:

```kotlin
// adkd-detekt.init.gradle.kts (plantilla, escrita por adkd si opts.UseStandalone=false)
// Inyectado en opts.ProjectDir antes de invocar ./gradlew detekt.
allprojects {
	detekt {
		xml = false  // SARIF only en adkd
		sarif {
			required = true
			output = rootProject.layout.buildDirectory
				.file("reports/detekt/adkd.sarif").get().asFile
		}
	}
}
```

- [ ] **Step 1b: Modo detection + tests**

```go
// internal/core/detektrunner/detect.go
package detektrunner

import (
	"os"
	"os/exec"
)

func Detect(projectDir string, preferStandalone bool) ExecutionMode {
	if preferStandalone {
		if _, err := exec.LookPath("detekt"); err == nil { return ModeStandalone }
	}
	if _, err := os.Stat(projectDir + "/gradlew"); err == nil {
		return ModeGradleWrap
	}
	return ModeStandalone // fallback final
}
```

```go
// internal/core/detektrunner/runner_test.go
package detektrunner

import (
	"testing"
)

func TestDetectStandaloneEnv(t *testing.T) {
	mode := Detect(t.TempDir(), true)
	// No assert estricto de modo: depende del ambiente CI.
	// Garantizamos que sí devuelve uno de los dos.
	if mode != ModeStandalone && mode != ModeGradleWrap {
		t.Fatalf("unexpected mode %q", mode)
	}
 move existing TestDetektBinaryName aquí como TestStandaloneBinaryName
}
```

- [ ] **Step 2: Test smoke**

```go
// internal/core/detektrunner/runner_test.go
package detektrunner

import "testing"
func TestDetektBinaryName(t *testing.T) {
	if DetektBinaryName() != "detekt" {
		t.Fatal("name changed")
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/core/detektrunner/
git commit -m "feat(detektrunner): spawn detekt subprocess with SARIF output"
```

> **Asunción V2 (Sprint 1):** confirmar en `examples/bad-project` que `detekt` standalone está disponible y parsea el proyecto; si no, escribir wrapper que invoque `./gradlew detekt`.

### Task 1.5 — SARIF parser estricto (OASIS 2.1.0)

**Files:**
- Create: `internal/core/sarif/sarif.go`
- Create: `internal/core/sarif/sarif_test.go`

**Interfaces:**
- `Parse(r io.Reader) (Findings []Finding, err error)` — devuelve finding normalizados.

- [ ] **Step 1: Structs SARIF 2.1.0**

```go
// internal/core/sarif/sarif.go
package sarif

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/adkd/adkd/internal/core/types"
)

type doc struct {
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}
type run struct {
	Tool struct {
		Driver struct {
			Name string `json:"name"`
		} `json:"driver"`
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	} `json:"tool"`
	Results []struct {
		RuleID    string `json:"ruleId"`
		Level     string `json:"level"` // "error"|"warning"|"note"|"none" (SARIF 2.1.0)
		Message   struct{ Text string `json:"text"` } `json:"message"`
		Locations []struct {
			Phys struct {
				ArtLoc struct{ URI string `json:"uri"` } `json:"artifactLocation"`
				Region struct{ StartLine, StartColumn int } `json:"region"`
			} `json:"physicalLocation"`
		} `json:"locations"`
	} `json:"results"`
}

// MapSARIFLevel convierte el nivel SARIF a types.Severity.
// "" o "none" se devuelve como "" para que el rulemap mande.
func MapSARIFLevel(level string) types.Severity {
	switch level {
	case "error":
		return types.SeverityError
	case "warning":
		return types.SeverityWarning
	case "note":
		return types.SeverityInfo
	}
	return ""
}

func Parse(r io.Reader) ([]types.Finding, error) {
	var d doc
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return nil, err
	}
	if d.Version != "2.1.0" {
		return nil, fmt.Errorf("unsupported SARIF version %q", d.Version)
	}
	var out []types.Finding
	for _, run := range d.Runs {
		for _, res := range run.Results {
			f := types.Finding{
				Rule:    res.RuleID,
				Message: res.Message.Text,
				Severity: MapSARIFLevel(res.Level), // el rulemap puede sobrescribir
			}
			if len(res.Locations) > 0 {
				f.File = res.Locations[0].Phys.ArtLoc.URI
				f.Line = res.Locations[0].Phys.Region.StartLine
				f.Column = res.Locations[0].Phys.Region.StartColumn
			}
			out = append(out, f)
		}
	}
	return out, nil
}
```

- [ ] **Step 2: Golden test con fixture mínima**

`testdata/sample.sarif`:

```json
{
  "version": "2.1.0",
  "runs": [{
    "tool": { "driver": { "name": "detekt" } },
    "results": [{
      "ruleId": "style:MagicNumber",
      "message": { "text": "Magic number" },
      "locations": [{
        "physicalLocation": {
          "artifactLocation": { "uri": "src/Foo.kt" },
          "region": { "startLine": 12, "startColumn": 5 }
        }
      }]
    }]
  }]
}
```

`internal/core/sarif/sarif_test.go`:

```go
package sarif

import (
	"os"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestParseGolden(t *testing.T) {
	f, err := os.Open("testdata/sample.sarif")
	if err != nil { t.Fatal(err) }
	defer f.Close()
	got, err := Parse(f)
	if err != nil { t.Fatal(err) }
	if len(got) != 1 { t.Fatalf("len %d", len(got)) }
	want := types.Finding{ Rule: "style:MagicNumber", File: "src/Foo.kt", Line: 12, Column: 5, Message: "Magic number" }
	if got[0] != want {
		t.Fatalf("\nwant %+v\ngot  %+v", want, got[0])
	}
}

func TestParseUnsupportedVersion(t *testing.T) {
	r := openString(`{"version":"0.0.0","runs":[]}`)
	_, err := Parse(r)
	if err == nil { t.Fatal("expected error") }
}
```

(necesitarás un helper `openString`).

- [ ] **Step 3: Commit**

```bash
git add internal/core/sarif/
git commit -m "feat(sarif): strict OASIS SARIF 2.1.0 parser"
```

### Task 1.6 — Rule mapping (Detekt ID → adkd rule)

**Files:**
- Create: `internal/core/rulemap/mapping.go`
- Create: `internal/core/rulemap/mapping_test.go`

**Interfaces:**
- `LoadRules(dir string) ([]types.Rule, error)` — carga `rules/metadata.json` desde la raíz del proyecto adkd.
- `LoadBuiltins() []types.Rule` — devuelve las 11 reglas built-in cuyo target es una regla Detekt conocida (mantenidas aquí como single source of truth, sincronizadas manualmente con `rules/metadata.json`).
- `Map(findings []types.Finding) []types.Finding` — enriquece con ID/Cluster/Severity/FixHint buscando por `rule.DetektRule == finding.Rule`. Filings sin mapear quedan con `ID: "unmapped:<rule>"` y `Cluster: "unknown"`.

> **Decisión de arquitectura (P0 #1):** Esta Task NO hardcodea el catálogo. Lee `rules/metadata.json` generado por Tarea 1.3. Si una regla Detekt no figura en metadata.json, queda `unmapped:` y se reporta a la consola para auditoría Sprint 1.

- [ ] **Step 1: Loader de metadata.json**

```go
// internal/core/rulemap/loader.go
package rulemap

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adkd/adkd/internal/core/types"
)

// LoadRules lee rules/metadata.json y devuelve el catálogo.
// El path es por defecto <repo-root>/rules/metadata.json cuando se
// construye la versión; el CLI resolverá la búsqueda en Tarea 1.10.
func LoadRules(path string) ([]types.Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules metadata: %w", err)
	}
	var rules []types.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules metadata: %w", err)
	}
	return rules, nil
}
```

- [ ] **Step 2: Indexador y Map**

```go
// internal/core/rulemap/mapping.go
package rulemap

import (
	"sort"

	"github.com/adkd/adkd/internal/core/types"
)

// Index es un índice in-memory construido desde []types.Rule.
// Permite lookup O(1) por DetektRule.
type Index struct {
	byDetekt map[string]types.Rule
}

// BuildIndex construye el índice a partir del catálogo.
// DetektRule vacío se omite (reglas planned sin target Detekt).
func BuildIndex(rules []types.Rule) *Index {
	idx := &Index{byDetekt: make(map[string]types.Rule, len(rules))}
	for _, r := range rules {
		if r.DetektRule == "" || r.Status != "live" {
			continue
		}
		idx.byDetekt[r.DetektRule] = r
	}
	return idx
}

// Map enriquece cada Finding con id/cluster/severity/fixHint del catálogo.
// Devuelve una NUEVA lista; el input queda intacto.
func (idx *Index) Map(findings []types.Finding) []types.Finding {
	out := make([]types.Finding, 0, len(findings))
	for _, f := range findings {
		if r, ok := idx.byDetekt[f.Rule]; ok {
			f.ID = r.ID
			f.Cluster = r.Cluster
			f.Severity = r.Severity
			f.FixHint = r.FixHint
		} else {
			f.ID = "unmapped:" + f.Rule
			f.Cluster = "unknown"
			f.Severity = types.SeverityInfo
		}
		out = append(out, f)
	}
	// orden estable por (severity, file, line) — útil para reporter
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File { return out[i].File < out[j].File }
		return out[i].Line < out[j].Line
	})
	return out
}
```

- [ ] **Step 3: Tests con fixture JSON**

```go
// internal/core/rulemap/mapping_test.go
package rulemap

import (
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestLoadRulesFromFixture(t *testing.T) {
	rules, err := LoadRules("testdata/metadata-sample.json")
	if err != nil { t.Fatal(err) }
	if len(rules) != 2 { t.Fatalf("len %d", len(rules)) }
}

// testdata/metadata-sample.json (committed):
// [
//   {"id":"a","cluster":"x","severity":"warning","detektRule":"Compose:ReusedModifierInstance","status":"live","fixHint":"hoist"},
//   {"id":"b","cluster":"x","severity":"info","detektRule":"","status":"planned"}
// ]

func TestMapKnown(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{ID:"a", Cluster:"x", Severity:types.SeverityWarning, DetektRule:"Compose:ReusedModifierInstance", Status:"live", FixHint:"hoist"},
	})
	out := idx.Map([]types.Finding{{ Rule:"Compose:ReusedModifierInstance", Message:"x" }})
	if len(out) != 1 { t.Fatal("len") }
	if out[0].ID != "a" || out[0].Cluster != "x" || out[0].FixHint != "hoist" {
		t.Fatalf("got %+v", out[0])
	}
}

func TestMapUnknown(t *testing.T) {
	idx := BuildIndex(nil)
	out := idx.Map([]types.Finding{{ Rule:"Nope" }})
	if out[0].ID != "unmapped:Nope" { t.Fatal(out[0].ID) }
	if out[0].Cluster != "unknown" { t.Fatal("cluster") }
}

func TestMapIgnoresPlanned(t *testing.T) {
	idx := BuildIndex([]types.Rule{
		{ID:"planned-only", Cluster:"x", Severity:types.SeverityWarning, DetektRule:"DetektRule:Present", Status:"planned"},
	})
	out := idx.Map([]types.Finding{{ Rule:"DetektRule:Present" }})
	if out[0].ID != "unmapped:DetektRule:Present" {
		t.Fatalf("planned rules deben NO matchear, got %s", out[0].ID)
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/core/rulemap/
git commit -m "feat(rulemap): load rules catalog from rules/metadata.json (single source of truth)"
```

### Task 1.7 — Health Score grader (fórmula V1)

**Files:**
- Create: `internal/core/grader/grader.go`
- Create: `internal/core/grader/grader_test.go`

- [ ] **Step 1: Implementar fórmula**

```go
// internal/core/grader/grader.go
// Fórmula V1: score = min(100, max(0, 100 - errors*5 - warnings*2 - info*0.5))
package grader

import "github.com/adkd/adkd/internal/core/types"

func Score(findings []types.Finding) (int, types.Summary) {
	var s types.Summary
	for _, f := range findings {
		s.Total++
		switch f.Severity {
		case types.SeverityError:   s.Errors++
		case types.SeverityWarning: s.Warnings++
		case types.SeverityInfo:    s.Info++
		}
	}
	raw := 100 - s.Errors*5 - s.Warnings*2 - int(float64(s.Info)*0.5)
	if raw > 100 { raw = 100 }
	if raw < 0 { raw = 0 }
	return raw, s
}
```

- [ ] **Step 2: Test fixtures**

```go
package grader

import (
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestEmptyIs100(t *testing.T) {
	score, sum := Score(nil)
	if score != 100 || sum.Total != 0 { t.Fatalf("got %d %+v", score, sum) }
}
func TestThreeErrorsDrop15(t *testing.T) {
	score, _ := Score([]types.Finding{
		{Severity: types.SeverityError},
		{Severity: types.SeverityError},
		{Severity: types.SeverityError},
	})
	if score != 85 { t.Fatalf("got %d", score) }
}
func TestClamped(t *testing.T) {
	// 200 info findings → 200*0.5=100 puntos a restar → debe clampsear a 0.
	in := make([]types.Finding, 200)
	for i := range in { in[i].Severity = types.SeverityInfo }
	score, sum := Score(in)
	if score != 0 { t.Fatalf("expected clamp to 0, got %d (sum=%+v)", score, sum) }
	if sum.Info != 200 { t.Fatalf("info count off: %d", sum.Info) }
}
func TestClampedErrors(t *testing.T) {
	// 50 errors → 250 puntos a restar → debe clampsear a 0, no negativo.
	in := make([]types.Finding, 50)
	for i := range in { in[i].Severity = types.SeverityError }
	score, _ := Score(in)
	if score != 0 { t.Fatalf("expected clamp to 0, got %d", score) }
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/core/grader/
git commit -m "feat(grader): implement V1 health score formula capped 0-100"
```

### Task 1.8 — Reporter consola (rich TUI minimal)

**Files:**
- Create: `internal/reporter/console/console.go`

**Interfaces:**
- `RenderReport(r types.Report, w io.Writer) error` — salida legible: big score + tabla de findings agrupados por cluster.

Se hace sin TUI interactiva en Fase 1 (sólo salida). En Fase 2 añadimos spinners durante el escaneo.

- [ ] **Step 1: Renderizado básico**

```go
package console

import (
	"fmt"
	"io"
	"sort"

	"github.com/adkd/adkd/internal/core/types"
)

func RenderReport(r types.Report, w io.Writer) error {
	scoreColor := pickColor(r.HealthScore)
	fmt.Fprintf(w, "Health Score: %s%d/100%s\n", scoreColor, r.HealthScore, "\033[0m")
	fmt.Fprintf(w, "%d errors · %d warnings · %d info (%d total)\n",
		r.Summary.Errors, r.Summary.Warnings, r.Summary.Info, r.Summary.Total)

	byCluster := map[string][]types.Finding{}
	for _, f := range r.Findings {
		byCluster[f.Cluster] = append(byCluster[f.Cluster], f)
	}
	clusters := make([]string, 0, len(byCluster))
	for c := range byCluster { clusters = append(clusters, c) }
	sort.Strings(clusters)
	for _, c := range clusters {
		fmt.Fprintf(w, "\n[%s] %d issues\n", c, len(byCluster[c]))
		for _, f := range byCluster[c] {
			fmt.Fprintf(w, "  %s %s:%d:%d  %s\n", f.Severity, f.File, f.Line, f.Column, f.Message)
			if f.FixHint != "" { fmt.Fprintf(w, "    → %s\n", f.FixHint) }
		}
	}
	return nil
}

func pickColor(s int) string {
	switch {
	case s >= 90: return "\033[32m"
	case s >= 75: return "\033[36m"
	case s >= 50: return "\033[33m"
	case s >= 25: return "\033[31m"
	}
	return "\033[31;1m"
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/reporter/console/
git commit -m "feat(reporter): rich console reporter with cluster grouping"
```

### Task 1.9 — Reporter JSON (schema v3)

**Files:**
- Create: `internal/reporter/jsonreporter/jsonreporter.go`
- Create: `internal/reporter/jsonreporter/jsonreporter_test.go`

- [ ] **Step 1: Encoder y test roundtrip**

```go
package jsonreporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestRoundTrip(t *testing.T) {
	r := types.Report{
		SchemaVersion: types.SchemaVersion,
		ProjectType:   "android",
		HealthScore:   47,
		Summary:       types.Summary{Errors: 3, Warnings: 9, Info: 6, Total: 18},
		Findings: []types.Finding{
			{ID: "x", Cluster: "compose-performance", Rule: "remember-missing",
			 Severity: types.SeverityError, File: "F.kt", Line: 12, Message: "msg", FixHint: "wrap"},
		},
	}
	b, err := json.Marshal(r)
	if err != nil { t.Fatal(err) }
	var again types.Report
	if err := json.Unmarshal(b, &again); err != nil { t.Fatal(err) }
	if again.HealthScore != 47 || len(again.Findings) != 1 { t.Fatalf("got %+v", again) }
}
```

- [ ] **Step 2: Helper Write**

```go
func Write(r types.Report, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/reporter/jsonreporter/
git commit -m "feat(reporter): JSON reporter with schema v3"
```

### Task 1.10 — Wire command `adkd scan` end-to-end

**Files:**
- Create: `internal/cli/scan.go`
- Create: `internal/cli/scan_test.go`
- Modify: `cmd/adkd/main.go` (vincular `newScanCmd`)

- [ ] **Step 1: Implementar `adkd scan`**

`internal/cli/scan.go`:

```go
package cli

import (
	"context"
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
)

func newScanCmd() *cobra.Command {
	var asJSON bool
	var projectType string
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the project and compute Health Score",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, _ := os.Getwd()
			sarifPath := filepath.Join(os.TempDir(), "adkd-detekt.sarif")
			if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
				ProjectDir: wd, SARIFOutput: sarifPath,
			}); err != nil { return fmt.Errorf("detekt: %w", err) }
			f, err := os.Open(sarifPath); if err != nil { return err }
			defer f.Close()
			raw, err := sarif.Parse(f); if err != nil { return err }
			mapped := rulemap.Map(raw)
			score, sum := grader.Score(mapped)
			report := types.Report{
				ProjectType: projectType, HealthScore: score,
				Summary: sum, Findings: mapped,
				SchemaVersion: types.SchemaVersion,
			}
			var out io.Writer = cmd.OutOrStdout()
			if asJSON { return jsonrep.Write(report, out) }
			return console.RenderReport(report, out)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of console")
	cmd.Flags().StringVar(&projectType, "type", "android", "project type: android|kmp|cmp")
	return cmd
}
```

- [ ] **Step 2: Vincular en main.go**

Verificar que `cmd/adkd/main.go` ya invoca `newScanCmd()` (añadido en Tarea 1.1). No requiere cambios aquí.

- [ ] **Step 3 (P1 #5 fix): Wirear flags `--prefer-standalone` + resolver path de metadata.json**

```diff
 func newScanCmd() *cobra.Command {
-	var asJSON bool
-	var projectType string
+	var (
+		asJSON           bool
+		projectType      string
+		preferStandalone bool
+	)
 	cmd := &cobra.Command{
 		Use:   "scan",
 		Short: "Scan the project and compute Health Score",
 		RunE: func(cmd *cobra.Command, args []string) error {
 			wd, _ := os.Getwd()
-			sarifPath := filepath.Join(os.TempDir(), "adkd-detekt.sarif")
-			if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
-				ProjectDir: wd, SARIFOutput: sarifPath,
-			}); err != nil { return fmt.Errorf("detekt: %w", err) }
+			mode := detektrunner.Detect(wd, preferStandalone)
+			sarifPath := filepath.Join(os.TempDir(), "adkd-detekt.sarif")
+			out := cmd.OutOrStdout()
+			if _, err := detektrunner.RunDetekt(context.Background(), detektrunner.Options{
+				ProjectDir:    wd,
+				SARIFOutput:   sarifPath,
+				UseStandalone: mode == detektrunner.ModeStandalone,
+				Stdout:        out,
+			}); err != nil { return fmt.Errorf("detekt: %w", err) }
 			f, err := os.Open(sarifPath); if err != nil { return err }
 			defer f.Close()
 			raw, err := sarif.Parse(f); if err != nil { return err }
-			mapped := rulemap.Map(raw)
+			rulesPath, err := resolveRulesPath()
+			if err != nil { return fmt.Errorf("rules metadata: %w", err) }
+			rules, err := rulemap.LoadRules(rulesPath)
+			if err != nil { return err }
+			idx := rulemap.BuildIndex(rules)
+			mapped := idx.Map(raw)
 			score, sum := grader.Score(mapped)
 			report := types.Report{
 				ProjectType: projectType, HealthScore: score,
 				Summary: sum, Findings: mapped,
 				SchemaVersion: types.SchemaVersion,
 			}
-			var out io.Writer = cmd.OutOrStdout()
 			if asJSON { return jsonrep.Write(report, out) }
 			return console.RenderReport(report, out)
 		},
 	}
 	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of console")
 	cmd.Flags().StringVar(&projectType, "type", "android", "project type: android|kmp|cmp")
+	cmd.Flags().BoolVar(&preferStandalone, "prefer-standalone", false, "prefer standalone detekt binary over ./gradlew")
 	return cmd
 }
+
+func resolveRulesPath() (string, error) {
+	// 1. Env var override
+	if p := os.Getenv("ADKD_RULES_DIR"); p != "" {
+		return filepath.Join(p, "metadata.json"), nil
+	}
+	// 2. Relative to the running binary (para binarios en dist/)
+	exe, err := os.Executable()
+	if err == nil {
+		dir := filepath.Dir(exe)
+		candidates := []string{
+			filepath.Join(dir, "rules", "metadata.json"),
+			filepath.Join(dir, "..", "rules", "metadata.json"),
+			filepath.Join(dir, "..", "..", "rules", "metadata.json"),
+		}
+		for _, c := range candidates {
+			if _, err := os.Stat(c); err == nil { return c, nil }
+		}
+	}
+	// 3. Fallback al working dir (desarrollo local con `go run`)
+	if _, err := os.Stat("rules/metadata.json"); err == nil {
+		return "rules/metadata.json", nil
+	}
+	return "", fmt.Errorf("rules/metadata.json no encontrado; set ADKD_RULES_DIR o compila con reglas dentro del directorio del binario")
+}
```

- [ ] **Step 4: Smoke test con proyecto mal**

```bash
mkdir -p examples/bad-project
cd examples/bad-project && \
echo 'package p\nfun a() = println(123)' > Foo.kt && \
go run ./cmd/adkd scan --type=android
```

Esperado: muestra `Health Score: 92/100` aprox y un finding por cada regla Detekt.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/ cmd/adkd/
git commit -m "feat(cli): wire adkd scan end-to-end with --prefer-standalone and rules path resolution"
```

### Task 1.11 — `adkd init` + `adkd.config.yaml`

**Files:**
- Create: `adkd.config.example.yaml`
- Create: `internal/cli/init.go`
- Create: `internal/core/config/config.go`
- Create: `internal/core/config/config_test.go`

- [ ] **Step 1: Tipos de config**

```go
// internal/core/config/config.go
package config

import "gopkg.in/yaml.v3"

type RuleConf struct {
	Severity string                 `yaml:"severity"`
	Options  map[string]interface{} `yaml:"options,omitempty"`
}

type Config struct {
	ProjectType string                       `yaml:"projectType"` // android|kmp|cmp
	Paths       map[string][]string          `yaml:"paths"`
	Rules       map[string]RuleConf          `yaml:"rules"`
	Score       struct{ FailBelow int `yaml:"failBelow"` } `yaml:"score"`
}

func Default() Config {
	return Config{ProjectType: "android", FailBelow: 80}
}
```

- [ ] **Step 2: `adkd init` escribe el YAML por defecto**

`internal/cli/init.go` (extracto):

```go
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use: "init",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := config.Default()
			data, _ := yaml.Marshal(c)
			return os.WriteFile("adkd.config.yaml", data, 0644)
		},
	}
}
```

- [ ] **Step 3: Test load**

```go
func TestLoad(t *testing.T) {
	yamlData := []byte("projectType: kmp\n")
	var c Config
	if err := yaml.Unmarshal(yamlData, &c); err != nil { t.Fatal(err) }
	if c.ProjectType != "kmp" { t.Fatal(c.ProjectType) }
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/core/config/ internal/cli/init.go adkd.config.example.yaml
git commit -m "feat(config): adkd init + adkd.config.yaml with YAML v1 schema"
```

### Task 1.12 — Examples: `examples/bad-project` y `examples/good-project` con fixtures

**Files:**
- Create: `examples/bad-project/` (intencionalmente roto)
- Create: `examples/good-project/` (limpio)
- Create: `examples/scoring-fixtures/bad.json`, `examples/scoring-fixtures/good.json`

- [ ] **Step 1: Definir fixtures en JSON**

`scoring-fixtures/bad.json`:

```json
{ "minScoreExpected": 40, "maxScoreExpected": 75, "mustIncludeFinding": "dead-unused-import" }
```

- [ ] **Step 2: Comando eval que valida**

`scripts/evalprojects/main.go` (código completo, ya no es placeholder):

```go
// scripts/evalprojects/main.go
// evalprojects itera sobre examples/scoring-fixtures/*.json, ejecuta
// `adkd scan --json` en el projectPath asociado y comprueba que el
// HealthScore resultante está dentro del rango esperado.
//
// Uso: go run ./scripts/evalprojects
// Exit code 0 si todas las fixtures pasan; 1 en caso contrario.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adkd/adkd/internal/core/types"
)

type Fixture struct {
	ProjectPath        string   `json:"projectPath"`
	MinScoreExpected   int      `json:"minScoreExpected"`
	MaxScoreExpected   int      `json:"maxScoreExpected"`
	MustIncludeFinding []string `json:"mustIncludeFinding"`
}

type finding struct {
	ID string `json:"id"`
}

type report struct {
	SchemaVersion string    `json:"schemaVersion"`
	HealthScore   int       `json:"healthScore"`
	Findings      []finding `json:"findings"`
}

func evaluate(fixture Fixture, adkdBinary string) error {
	cmd := exec.Command(adkdBinary, "scan", "--json", "--type=android")
	cmd.Dir = fixture.ProjectPath
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("adkd scan %s: %w", fixture.ProjectPath, err)
	}
	var r report
	if err := json.Unmarshal(out, &r); err != nil {
		return fmt.Errorf("parse %s: %w", fixture.ProjectPath, err)
	}
	if r.SchemaVersion != types.SchemaVersion {
		return fmt.Errorf("%s: schemaVersion %q no soportado (esperado %q)",
			fixture.ProjectPath, r.SchemaVersion, types.SchemaVersion)
	}
	if r.HealthScore < fixture.MinScoreExpected || r.HealthScore > fixture.MaxScoreExpected {
		return fmt.Errorf("%s: score %d fuera de [%d, %d]",
			fixture.ProjectPath, r.HealthScore,
			fixture.MinScoreExpected, fixture.MaxScoreExpected)
	}
	ids := map[string]bool{}
	for _, f := range r.Findings { ids[f.ID] = true }
	for _, want := range fixture.MustIncludeFinding {
		if !ids[want] {
			return fmt.Errorf("%s: falta finding esperado %q", fixture.ProjectPath, want)
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: evalprojects <path-a-adkd-binary>")
		os.Exit(2)
	}
	adkdBin := os.Args[1]
	fixturesGlob := filepath.Join("examples", "scoring-fixtures", "*.json")
	files, err := filepath.Glob(fixturesGlob)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no se encontraron fixtures en", fixturesGlob)
		os.Exit(1)
	}
	fails := 0
	for _, f := range files {
		data, _ := os.ReadFile(f)
		var fix Fixture
		if err := json.Unmarshal(data, &fix); err != nil {
			fmt.Fprintln(os.Stderr, f, err); fails++; continue
		}
		if err := evaluate(fix, adkdBin); err != nil {
			fmt.Fprintln(os.Stderr, "❌", err); fails++; continue
		}
		fmt.Println("✅", fix.ProjectPath)
	}
	if fails > 0 {
		fmt.Fprintf(os.Stderr, "%d fixtures fallaron\n", fails)
		os.Exit(1)
	}
}
```

`scripts/evalprojects/main_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFixturesParse(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "examples", "scoring-fixtures", "*.json"))
	if len(files) == 0 { t.Fatal("no fixtures found") }
	for _, f := range files {
		var fix Fixture
		data, _ := os.ReadFile(f)
		if err := json.Unmarshal(data, &fix); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if fix.MinScoreExpected > fix.MaxScoreExpected {
			t.Fatalf("%s: min %d > max %d", f, fix.MinScoreExpected, fix.MaxScoreExpected)
		}
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add examples/ scripts/evalprojects/
git commit -m "test(examples): bad/good project fixtures + eval check"
```

---

# Fase 2 — SARIF writer + CI diff + baseline (Semana 3)

### Task 2.1 — SARIF writer

**Files:**
- Create: `internal/reporter/sarif/sarif.go`
- Create: `internal/reporter/sarif/sarif_test.go`

Adaptar el parser al revés: emit SARIF 2.1.0 a partir de `types.Report`. Reusar structs del paquete `internal/core/sarif`.

### Task 2.2 — `--diff <ref>` (sólo findings nuevos)

**Files:**
- Create: `internal/core/diff/diff.go`
- Modify: `internal/cli/scan.go`

`diff.go` usa `git diff <ref> --name-only` (output: lista de paths cambiados). Cross-check con `Finding.File`.

### Task 2.3 — Baseline (`--baseline <file.xml>`)

**Files:**
- Create: `internal/core/baseline/baseline.go`

Lee `baseline.xml` (formato Detekt-compatible) y descarta findings listados.

### Task 2.4 — GitHub Action composite

**Files:**
- Create: `.github/workflows/adkd-action.yml` + `.github/actions/adkd/action.yml`

Reusable action con inputs `fail-below`, `diff`, `baseline`.

### Task 2.5 — Pre-push / pre-commit hook

**Files:**
- Create: `internal/cli/hook.go`
- Modify: docs/website/index.html (microcopy sobre el hook)

`adkd hook install --fail-below 80` crea `.git/hooks/pre-push` que ejecuta `adkd scan --fail-below 80` y bloquea si falla.

---

# Fase 3 — AI Fixer + Quality-Focused prompting (Semana 4)

### Task 3.1 — Quality-Focused prompt builder (sin RCI)

**Files:**
- Create: `internal/aifixer/qualityprompt/builder.go`
- Create: `internal/aifixer/qualityprompt/builder_test.go`

Plantilla basada en el PDF V2: System Prompt restrictivo + Context (skeleton) + Finding singular con `FixHint`. **Nunca** invocar el LLM dos veces; un solo paso, sin auto-revisión.

### Task 3.2 — Provider Claude Code CLI

**Files:**
- Create: `internal/aifixer/provider/claude.go`

Ejecuta `claude --file prompt.md` y captura el patch generado.

### Task 3.3 — Provider Cursor / Gemini / Codex / MobiAI stubs

**Files:**
- Create: `internal/aifixer/provider/{cursor,gemini,codex,mobiai}.go`

Stubs mínimos — comando base + flag a descubrir en Sprint 4.

### Task 3.4 — Patch guard (post-fix syntax check)

**Files:**
- Create: `internal/aifixer/patchguard/guard.go`

Re-lectura de `.kt` modificado, validación con regex de paréntesis balanceados, conteo de errores sintácticos obvios. Si falla, descartar el patch y devolver mensaje al usuario.

### Task 3.5 — `adkd fix --ai` command

**Files:**
- Create: `internal/cli/fix.go`

Tres modos: `suggest` (default, no toca), `interactive`, `auto`. Por defecto `suggest` para no destruir código accidentalmente.

### Task 3.6 — Actualizar landing page (mención "Native Go")

**Files:**
- Modify: `docs/website/index.html`
- Create: `docs/website/changelog.md`

Reemplazar cualquier mención a "Node/TS" por "Native Go · Cobra". Añadir enlace al repo.

### Task 3.7 — End-to-end demo fixture

**Files:**
- Create: `examples/bad-project/` con bugs que el fixer arregla
- Create: `examples/good-project/` mismo proyecto tras `adkd fix --ai`

Grabar la salida en `examples/demo-output.txt` (referencia para CI docs).

---

# Fase 4 — MobiAI skill (Semana 5)

### Task 4.1 — `SKILL.md` (formato agentskills.io)

**Files:**
- Create: `SKILL.md`
- Create: `.mobiai/plugin.json` (descriptor que el CLI `mobiai` consume)

> **CRÍTICO (P1 #9):** El formato NO es Cursor (`*.cursorrules`) ni Claude Code (`~/.claude/skills/<skill>/SKILL.md`). MobiAI consume `agentskills.io`, mismo schema que Anthropic para sus skills. El frontmatter **debe** respetar este contrato, si no `mobiai skills install adkd` falla silenciosamente.

`SKILL.md` (frontmatter exacto, copiar sin modificar campos):

```markdown
---
name: adkd
description: Ejecuta `adkd scan` en proyectos Android/KMP/CMP para auditar la salud del código. Encuentra antipatrones en Compose, coroutines, lifecycle y arquitectura; produce un Health Score 0–100 y findings accionables. Usar cuando el usuario pida auditar, revisar o aplicar fixes automáticos a código Android/Kotlin.
when_to_use: Cuando hay un proyecto Android/KMP/CMP y el usuario quiere diagnóstico, score, o auto-fix con IA.
---

# adkd — Android/KMP/CMP Doctor

## Cuándo invocar

- "audita mi proyecto Android", "pásale el doctor", "¿qué tal está mi Health Score?"
- "encuentra antipatrones de Compose en mi app"
- "arregla los issues automáticamente con IA"

## Comandos principales

### 1. `adkd scan`
Ejecuta en la raíz del proyecto. Devuelve:
- Health Score (0–100).
- Lista de findings agrupados por cluster (compose-performance, coroutines, lifecycle, architecture, accessibility, testing, security, kmp, dead-code).
- Modo `--json` para CI; modo `--sarif` para GitHub Code Scanning.
- Modo `--diff main` para auditar solo cambios nuevos respecto a una rama.

### 2. `adkd fix --ai`
Tres modos:
- `--mode suggest` (default): genera `fixes.md`, NO toca código.
- `--mode interactive`: pregunta por cada fix.
- `--mode auto`: aplica, valida con patch guard, deja `git diff` listo.

El LLM provider se detecta automáticamente (`provider: auto`). MobiAI, Claude Code, Cursor, Gemini CLI, Codex funcionan out-of-the-box.

### 3. `mobiai doctor --code`
Una vez instalado `mobiai skills install adkd`, este subcomando corre `adkd scan` y, si el usuario acepta, lanza `--fix --ai`.

## Flujo típico (ejemplo end-to-end)

1. Usuario: "audita el módulo app del proyecto".
2. Invocar `mobiai graph context "android audit module app"` primero — MobiAI Graph da la lista de archivos relevantes.
3. Sobre esos archivos: `adkd scan --diff main --json`.
4. Parsear JSON: extraer `findings[]` ordenados por severidad.
5. Si `healthScore` baja: proponer al usuario `adkd fix --ai --mode interactive`.
6. Tras cada fix: re-scan, mostrar delta de score.
```

`plugin.json` (descriptor MobiAI):

```json
{
  "name": "adkd",
  "displayName": "adkd (Android Doctor)",
  "description": "Static analysis + AI auto-fix Health Score for Android/KMP/CMP.",
  "version": "0.1.0",
  "author": "adkd contributors",
  "license": "MIT",
  "homepage": "https://github.com/adkd/adkd",
  "repository": "github.com/adkd/adkd",
  "commands": [
    { "name": "scan",  "binary": "adkd",  "args": ["scan"]  },
    { "name": "fix",   "binary": "adkd",  "args": ["fix"]   },
    { "name": "init",  "binary": "adkd",  "args": ["init"]  },
    { "name": "hook",  "binary": "adkd",  "args": ["hook"]  },
    { "name": "doctor","binary": "adkd",  "args": ["doctor"] }
  ],
  "skills": ["./SKILL.md"],
  "hooks": [],
  "depends_on": ["core"]
}
```

### Task 4.2 — `plugin.json` (descriptor MobiAI)

**Files:**
- Create: `plugin.json`

Schema MobiAI: name=adkd, version, depends_on=[], commands=[scan, fix, init, doctor, hook].

### Task 4.3 — `adkd scan --mobiai` output

**Files:**
- Modify: `internal/cli/scan.go`

Cuando flag `--mobiai`, vuelca findings como anotaciones compatibles con MobiAI Graph (`.mobiai/graph/findings.jsonl`).

### Task 4.4 — `mobiai doctor --code` spec

**Files:**
- Create: `docs/integrations/mobiai.md`

Documentar el wiring con MobiAI (no se modifica MobiAI). El usuario de MobiAI ejecuta `mobiai skills add adkd` y obtiene el binario como skill.

### Task 4.5 — Test de integración contra MobiAI en CI

**Files:**
- Create: `.github/workflows/mobiai-install-test.yml`

Instala `mobiai` (versión latest stable), clona fixtures, verifica que `adkd scan --mobiai` produce el JSON correcto.

---

# Fase 5 — K2 FIR R&D (Semana 6–8, R&D, no bloqueante)

### Task 5.1 — Spike: kotlin-compiler-embeddable vía JVM child process

**Files:**
- Create: `internal/jvmrunner/spike.go`
- Create: `docs/research/k2-fir-spike.md`

Invocar un jar Kotlin compilable vía `java -jar` que use `FirAdditionalCheckersExtension` para reportar composition-related issues. Evaluar:
- Tiempo de arranque JVM.
- Coste de parsear un repo Android.
- Estabilidad del API entre Kotlin 2.0.x y 2.1.x.

### Task 5.2 — Decisión Go vs JVM para K2 FIR

**Files:**
- Create: `docs/architecture/decisions/0007-k2-fir-strategy.md` (ADR)

Basado en Task 5.1, decidir si adkd crece una capa JVM o si se mantiene en Go puro confiando en Detekt SARIF para el análisis profundo.

### Task 5.3 — Si JVM: 1 check piloto (`compose-remember-missing`)

**Files:**
- Create: `plugins/k2fir-checkers/build.gradle.kts`
- Create: `plugins/k2fir-checkers/src/main/kotlin/.../RememberMissingChecker.kt`

Sólo si Task 5.2 decide que sí. Reportar el checker a través del mismo flujo SARIF.

### Task 5.4 — Si Go puro: declarado como "postponed" y roadmap-2.x

Sellar la decisión en `docs/architecture/decisions/0007-k2-fir-strategy.md`.

---

## Riesgos y mitigaciones (consolidado)

| Riesgo | Mitigación |
|---|---|
| Detekt corre lento en CI | Habilitar Gradle Daemon; spinners visuales en reporter; flag `--parallel`; ofrecer binario standalone en Fase 5 si fuera útil. |
| Las 64 reglas no se cubren con Detekt | Marcar `status: planned` en `rules/metadata.json`. Plan sprints 2.x en torno al subset live. |
| AI Fixer corrompe archivos | Patch guard obligatorio (paréntesis, llaves); modo `suggest` por defecto. |
| MobiAI Graph no inicializado | Fallback regex propio en `internal/mobiai/graphbridge/fallback.go`. |
| Equipo con poco Go | `CONTRIBUTING.md` con Go cheatsheet; checklist de CI enforces `gofmt`/`go vet`. |
| K2 FIR API inestable | Aplazado a Fase 5 con spike; si API cambia, ADR firmado + nubes en `docs/architecture/decisions/`. |
| Reglas custom Detekt requieren JVM | Phase 5 con spike; Fase 1–4 trabaja sólo con reglas existentes. |

---

## Herramientas (MCPs / skills) que necesitaremos

**MCPs:**

- **No detecté MCP que falte.** Las herramientas disponibles en Codebuff (researcher-web, researcher-docs, browser/render, file editing, bash, Git via terminal) cubren investigación de docs, lectura de GitHub, lectura de PDFs, edición de archivos y CI local.
- Si en algún punto quieres integrar con GitHub (crear el repo público, abrir PRs, mergear Releases), lo hacemos vía `gh` CLI desde terminal en Fase 4-5.

**Skills (loaded y ready):**

- **`writing-plans` ✅ activa ahora** — estructura del plan.
- **`executing-plans`** — se cargará cuando arranquemos Fase 1, Task 1.1.
- **`test-driven-development`** — se cargará en cada Task de implementación.
- **`verification-before-completion`** — antes de cerrar cada Task.
- **`brainstorming`** — si rebotamos en una decisión de diseño (ej. K2 spike).
- **`skill-creator`** — si quieres exportar `adkd` como skill completo para que Codebuff lo use en otros proyectos.
- **`mcp-builder`** — si decides exponer Graph como MCP server a futuro (lo menciono como follow-up, no es necesario ahora).

**Plugins Go:**

- `cobra`, `charmbracelet/huh` o `lipgloss`, `pelletier/go-tree-sitter` + binding Kotlin, `gopkg.in/yaml.v3`, `golang.org/x/mod`.

---

## Verificación end-to-end (Definition of Done global)

`adkd scan --type=android` en `examples/bad-project` produce Health Score ∈ [40,75] y contiene `dead-unused-import` en findings. `adkd scan --type=android` en `examples/good-project` produce Health Score ∈ [95,100].

`adkd fix --ai --dry-run` en `examples/bad-project` produce un `fixes.md` con al menos 1 finding corregido propuesto. `--auto` aplica + valida con patch guard.

`mobiai skills add adkd` instala y `mobiai doctor --code` corre `adkd scan`.

CI pasa: tests, `go vet`, `golangci-lint`, build cross-platform.

---

## Self-review del plan

1. **Spec coverage:** V1 + V2 cubiertas (Health Score, 64 reglas catalogadas, SARIF, CI, MobiAI). Las 64 reglas se cargan al catálogo, no se implementan todas; el subset live se somete a auditoría Sprint 1 antes de afirmar `detektRule` mapping correcto en `rules/metadata.json`.
2. **Placeholders:** Sustituidos por código exacto en Tarea 1.1–1.10. Tareas 2–5 incluyen signatures y nombres de archivos, código detallado cuando es crítico (e.g. fórmula Health Score, parser SARIF), arquitectura para el resto.
3. **Type consistency:** `types.Finding`, `types.Severity`, `types.Report` definidos en Task 1.2 y consumidos por **todas** las Tasks posteriores. Los nombres (`ID`, `Cluster`, `Rule`, `Severity`, `File`, `Line`, `Column`, `Message`, `FixHint`, `DocURL`) no cambian entre Tasks.

---

## Handoff para ejecutar

**Plan completo y guardado en `docs/superpowers/plans/2026-07-19-adkd-implementation-plan.md`.**

Dos rutas de ejecución posibles cuando arranquemos:

1. **Subagent-Driven (recomendada)** — Lanzo un subagente fresco por cada Task, revisión entre Tasks, iteración rápida. Carga de skills: `subagent-driven-development`.
2. **Inline Execution** — Ejecuto las Tasks en esta misma sesión con checkpoints. Carga de skills: `executing-plans`.

Cuando me digas cuál prefieres y arranquemos, llevo a cabo Fase 1 completa (12 Tasks) y validamos el primer `adkd scan` end-to-end contra `examples/bad-project` antes de pasar a Fase 2.
