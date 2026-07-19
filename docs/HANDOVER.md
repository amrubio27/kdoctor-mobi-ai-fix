# kdoctor — Handover document (continuidad entre sesiones IA)

> **Objetivo**: que cualquier IA (o persona) que llegue a este repo pueda leer este único documento y saber **exactamente** qué es kdoctor, qué está hecho, qué está pendiente, cómo validarlo, y cómo continuar.

> **Estado al cierre**: 12 commits en `main`, Phases 1 + 1.5 cerradas, **Tier 1 #1 cerrado** ✅. Tiers 1 #2 / 1 #3 / 2 #4 / 2 #5 / 3 #6 / 3 #7 pendientes, en orden de Tier 1 → 3.

---

## 1. TL;DR (1 párrafo)

**kdoctor** es un CLI en Go para auditar calidad de proyectos Android / KMP / CMP. Inspirado en `react-doctor`. Reutiliza el binario `detekt` como motor de análisis estático, parsea su salida SARIF 2.1, mapea reglas Detekt a reglas kdoctor (78 reglas: 11 V1 live + 53 V1 planned + 14 default-detekt mappings Phase 1.5), calcula un **Health Score 0-100** y emite reportes en consola/JSON/SARIF. Naming final fijado en `kdoctor` (anteriormente `adkd`).

---

## 2. Repo state (12 commits)

```
ee191e0 feat(rules): Phase 1.5 — 78 reglas catalog + FixHint + consistency tests
217adc7 feat(scan): end-to-end validation against RickMortyApp — Fase 1 closed
b20899f chore(license): rename adkd to kdoctor in copyright holder
9d3c1ee refactor(rebrand): rename adkd → kdoctor
b119be4 refactor(detektrunner): limpiar vars zombies y consolidar test helpers
642b337 feat(detektrunner): validar formato SARIF 2.1.0 antes de aceptar .sarif
140b8f8 fix: validate Fase 1 canvas with real Go 1.26.5
7389bf7 feat: scaffold adkd Fase 1 PoC Inspector
[new]    test(tier1-1): smoke test evalprojects + bad-project seeded + detekt config REPLACE-aware
```

- **Branch**: `main`, working tree dirty pre-commit (closing Tier 1 #1 changes uncommitted).
- **Working directory local**: `C:\Users\Miguel\Desktop\doctor mobi ai fix`.
- **Project local module path**: `github.com/adkd/adkd` (placeholder — pendiente GitHub handle del usuario).

---

## 3. Tech stack

- **Go 1.22+** (testado con `go1.26.5` en `C:\Program Files\Go\bin\go.exe`)
- **`github.com/spf13/cobra`** para la CLI
- **`detekt-cli` 1.23.8** como motor de análisis (en `D:\tools\detekt-cli.jar` + shim `D:\tools\detekt.cmd`)
- **JDK 25.0.3 (Temurin)** en `C:\Program Files\Eclipse Adoptium\jdk-25.0.3.9-hotspot\bin` (NO en PATH de usuario permanente; se exporta en cada shell)
- **Sin Node, sin cGo, sin dependencias nativas**. Compilación reproducible desde cero con `go mod tidy && go build -o kdoctor.exe ./cmd/kdoctor`.

---

## 4. Top-level layout (post Tier 1 #1)

```
.
├── ANDROID_DOCTOR_FIX_AI.md            # V1 rules spec (input original)
├── LICENSE                             # MIT
├── README.md
├── go.mod, go.sum                      # module github.com/adkd/adkd
├── kdoctor.config.example.yaml         # scaffold para config del usuario (Fase 2)
├── kdoctor.exe                         # BUILT binary (gitignored)
├── cmd/
│   └── kdoctor/                        # entrypoint Cobra
│       └── main.go
├── internal/
│   ├── cli/                            # scan.go, fix.go, init.go, doctor.go, hook.go, ci.go
│   │   └── scan.go                     # comando principal (Fix Bug C: --json/--sarif → detekt stdout = io.Discard)
│   ├── core/
│   │   ├── types/                      # Finding, Rule, HealthScore, Summary, Report
│   │   ├── sarif/                      # parser SARIF 2.1.0 estricto
│   │   ├── detektrunner/               # subprocess manager (standalone + gradlew modes)
│   │   │   └── runner.go               # Fix Bug A: filepath.Abs en runStandalone + runGradlew; auto-detect detekt.yml/.yaml
│   │   ├── rulemap/                    # Detekt ID → kdoctor rule
│   │   │   ├── mapping.go              # Fix Bug 1: prefix-strip del vendor antes de lookup
│   │   │   └── mapping_test.go         # Fix reviewer nit: TestMapPrefixStrip regression guard
│   │   ├── grader/                     # Health Score 100 - err*5 - warn*2 - info*0.5
│   │   └── config/                     # kdoctor.config.yaml loading
│   └── reporter/
│       ├── console/                    # rich TUI (lipgloss, when TTY)
│       ├── jsonreporter/               # schema v3 JSON
│       └── sarif/                      # SARIF 2.1.0 writer
├── rules/
│   └── metadata.json                   # 78 reglas (output of scripts/genschema)
├── scripts/
│   ├── genschema/                      # CATALOG source of truth → rules/metadata.json
│   │   ├── main.go                     # 78 Rule{...} entries (edit THIS not the JSON)
│   │   └── main_test.go                # TestCatalogConvergence + TestClusterTaxonomyIsConsistent + TestValidPrefixesCoverage
│   └── evalprojects/                   # SMOKE TEST runner
│       ├── main.go (CLI)               # Reads examples/scoring-fixtures/*.json
│       └── main_test.go                # Tier 1 #1 tests: TestEvalBadProjectFixture + TestEvalBadProjectFixtureRelative + TestEvalGoodProjectFixture
├── examples/
│   ├── bad-project/                    # SEEDED antipatterns for smoke test
│   │   ├── detekt.yml                  # NEW (Tier 1 #1): activate HardcodedPassword-equivalents on detekt 1.23.x
│   │   └── src/main/kotlin/bad/BadCode.kt  # Seeded: GlobalScope, unused import, hardcoded secrets, god class TODO
│   ├── good-project/                   # SEEDED clean code
│   │   └── src/main/kotlin/good/Greeter.kt
│   └── scoring-fixtures/
│       ├── bad.json                    # REQUIREMENTS: mustIncludeFinding + score band 80-95
│       └── good.json                   # REQUIREMENTS: score 95-100, no required findings
├── docs/
│   ├── HANDOVER.md                     # ← YOU ARE HERE
│   ├── superpowers/plans/
│   │   └── 2026-07-19-kdoctor-implementation-plan.md   # MASTER plan
│   ├── v2/
│   │   └── kdoctor-proposal-v2.md      # extracted del PDF MobiAi Cli PoC Addon.pdf
│   └── website/                        # landing page existing
└── .gitignore
```

---

## 5. Architecture (Phase 1+1.5 + Tier 1 #1)

```
┌─────────────────┐     spawn subprocess     ┌─────────────────┐
│ kdoctor scan   │ ──────────────────────── │ detekt-cli      │
│ ─────────────   │                          │ ──────────────── │
│ 1. resolveRules │   reads rules/metadata.json via scripts/genschema
│ 2. Detect mode  ├─ standalone (default Fase 1.5) OR ./gradlew (scaffold)
│ 3. RunDetekt    │ writes SARIF to os.TempDir()/kdoctor-detekt.sarif
│    ├─ RunStandalone: java -jar detekt-cli.jar
│    │  - --input <absProjectDir>           (Fix Bug A: filepath.Abs)
│    │  - --report sarif:<sarifPath>
│    │  - --max-issues 99999               (forces exit 0 even with findings)
│    │  - --excludes '**/build/**,...'
│    │  - --config detekt.yml|.yaml       (NEW Tier 1 #1: auto-detect)
│    │  - safety net: Remove() before run; if cmd.Run fails but SARIF OK, accept it
│    └─ RunGradlew: ./gradlew detekt --init-script (también Fix Bug A via absProjectDir)
│ 4. sarif.Parse(file)  → []types.Finding
│ 5. rulemap.BuildIndex(rules) + Map(findings)
│    - Fix Bug 1: strip "detekt.<ruleset>." prefix before lookup
│    - unmapped findings: id="unmapped:<orig>", cluster="unknown"
│ 6. grader.Score(findings) → int 0-100 + Summary
│ 7. Reporter.RenderReport(...) → console / JSON / SARIF
│    - Fix Bug C: --json / --sarif → detekt stdout = io.Discard (no warnings pollution)
└─────────────────┘
```

**Adapted from V2 proposal**: bypassing Node.js, using Go + detekt SARIF, single-pass quality-focused prompts for future fix phase.

---

## 6. Key types (`internal/core/types/types.go`)

```go
const SchemaVersion = "3"

type Finding struct {
    ID, Cluster, Rule, Severity, File, Message, FixHint, DocURL string
    Line, Column int
}

type Report struct {
    SchemaVersion, ProjectType string
    HealthScore int
    Summary     { Errors, Warnings, Info, Total int }
    Findings    []Finding
}

type Rule struct {  // in rules/metadata.json
    ID, Cluster, DetektRule, Status, FixHint, DocURL string
    Severity Severity  // "error" | "warning" | "info"
}
```

SchemaVersion=3 alinea con react-doctor para que tools downstream lo entiendan.

---

## 7. Validación rápida (copiar-pegar)

```bash
# Setup PATH (en CADA nueva terminal; JDK no está en PATH permanente)
export PATH="/c/Program Files/Go/bin:/c/Program Files/Eclipse Adoptium/jdk-25.0.3.9-hotspot/bin:/d/tools:$PATH"
cd "C:\\Users\\Miguel\\Desktop\\doctor mobi ai fix"

# Validar Go vet + tests + build
go vet ./...                                       # exit 0
go test ./...                                      # ~38 tests, ALL PASS (incl. TestEvalBadProject, TestEvalGood, TestMapPrefixStrip)
go build -o kdoctor.exe ./cmd/kdoctor              # exit 0, binary ~7MB

# Validar catálogo no-drift (regen metadata.json desde scripts/genschema)
go run ./scripts/genschema -out rules/metadata.json
# → stderr: "Generated 78 rules to rules/metadata.json"

# Smoke test contra RickMortyApp (proyecto real del usuario)
./kdoctor.exe scan --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp
# → exit 0, Health Score 0/100, 103 findings clustered (Phase 1 close-out)

# Smoke test Tier 1 #1 (go test ./...)
go test -v -run TestEval ./scripts/evalprojects/...
# → 3 PASS: TestEval{Good,Bad,BadRelative}ProjectFixture; score 95 en bad-project, 100 en good-project

# Sub-comandos CLI
./kdoctor.exe --version           # "kdoctor version 0.1.0"
./kdoctor.exe --help              # usage completo
./kdoctor.exe rules               # 78 reglas listadas
./kdoctor.exe doctor              # checks de deps (Go, Java, Detekt, etc.)
./kdoctor.exe scan --help         # help completo del scan
```

---

## 8. Tier roadmap (status actual)

| Tier | Item | Estado | Próximo paso |
|---|---|---|---|
| **Tier 1 #1** | examples/bad-project + smoke test in `go test` | ✅ **CERRADO** (commit pendiente) | `go test ./scripts/evalprojects/...` → 3 PASS; score bad=95, good=100; 2/11 V1 live mappable (`coroutine-global-scope`, `dead-unused-import`) |
| **Tier 1 #2** | Go-based regex detectors en `internal/core/rules/` | ⏳ Pendiente | Inicia con `sec-log-pii` (Log.* con PII), `compose-remember-missing`, etc. Los V1 planned sin detekt equivalente se implementan como Go regex post-AST. |
| **Tier 1 #3** | `--diff main` / baseline.xml | ⏳ Pendiente | git merge-base + diff SARIF output; útil para que kdoctor sólo falle en lo nuevo del PR |
| **Tier 2 #4** | `kdoctor-gradle-plugin` | ⏳ Pendiente | Subproyecto Gradle con task `kdoctorScan` no-invasivo; users aplican `apply plugin: kdoctor` |
| **Tier 2 #5** | `kdoctor fix --ai` Quality-Focused | ⏳ Pendiente | LLM CLI adapter (Claude Code primero) + dry-run obligatorio; emite fixes.md sin tocar código |
| **Tier 3 #6** | HTML dashboard | ⏳ Pendiente | vite + tailwind sobre `kdoctor scan --json`; charts por cluster, score trend, integración MobiAI Graph |
| **Tier 3 #7** | `kdoctor.config.yaml` overrides por equipo | ⏳ Pendiente | Cluster-aware severity + custom excludes + disable-rule per cluster |

**Orden de ejecución fijado por el usuario**: cerrar Tier 1 → 2 → 3 secuencialmente. Cada tier antes de empezar el siguiente libera un commit.

---

## 9. Gotchas críticos (leer antes de tocar nada)

1. **NO commitear nada en `D:\Programacion\RickMortyApp`** — el usuario lo dijo explícitamente. Es su proyecto de prueba; cualquier cambio ahí debe revertirse o consultarse primero. kdoctor es READ-ONLY sobre el proyecto del usuario.

2. **`go.mod` tiene `module github.com/adkd/adkd`** — placeholder. Cuando el usuario defina su handle de GitHub, hay que cambiar esa línea + `go mod tidy`. Reversible sin breakage.

3. **Detekt setup del usuario está en `D:\tools\`** — `detekt-cli.jar` + `detekt.cmd` shim. Fuera de PATH de usuario. kdoctor acepta `--detekt-bin=<ruta_explícita>` para no depender del PATH.

4. **JDK NO está en PATH permanente**. Hay que hacer `export PATH=...:$PATH` cada terminal. Canónica: `C:\Program Files\Eclipse Adoptium\jdk-25.0.3.9-hotspot\bin\java.exe`.

5. **CRLF vs LF en Windows**. Después de CUALQUIER edit a `.go`, correr `gofmt -w <archivo>` para normalizar y evitar drift en `git diff`.

6. **`os.TempDir()` en Windows = `C:\Users\<user>\AppData\Local\Temp\`**. SARIF lives ahí. Sarifs stale pueden MASKAR errores reales. **Safety net actual** (Bug A close-out): cada run hace `os.Remove(sarifPath)` ANTES de invocar detekt → solo acepta SARIF fresco.

7. **gitignore incluye `kdoctor.exe`** (binario compilado). No commitear; se regenera con `go build`.

8. **Perl de codestyle**: `var defaultExcludes = []string{"**/build/**", "**/.gradle/**", "**/kspCaches/**"}` se pasa a detekt como `--excludes <csv>`. Sin exclude, detekt escanea `.gradle/` y `build/` con ruido false-positive.

9. **Drift guardrail catalog** (`scripts/genschema/main_test.go::TestCatalogConvergence`): falla si `rules/metadata.json` no coincide con `CatalogRules`. **IMPORTANTE**: editar SIEMPRE primero `scripts/genschema/main.go`, después correr `go run ./scripts/genschema` para regenerar JSON, commitear ambos.

10. **`TestClusterTaxonomyIsConsistent` y `TestValidPrefixesCoverage`** son meta-tests que prohíben prefijos fuera del cluster (`style-*` cuando el cluster es `complexity`). Si añades nueva regla a `CatalogRules`, añade su cluster a `validPrefixes` map.

### Gotchas del cierre de Tier 1 #1 (NUEVOS)

11. **`HardcodedPassword` rule NO existe en detekt-cli 1.23.x.** Verifiqué con `unzip -p detekt-cli.jar default-detekt-config.yml | grep -i hardcode` — 0 hits. La regla fue eliminada o renombrada en esa versión de detekt. Por eso `sec-hardcoded-secret` está marcada `status="live"` en el catalog (forward-compat con detekt 2.x+) pero NO smoke-testada. Si actualizás a detekt 2.x, re-agrega el stanza `potential-bugs.HardcodedPassword: active: true` en `examples/bad-project/detekt.yml` y `sec-hardcoded-secret` a `mustIncludeFinding` en `bad.json`.

12. **`UnusedImports` es PLURAL en detekt 1.23.x.** El catalog dice `detektRule: "UnusedImports"` (plural). Antes tenía `UnusedImport` (singular) que fallaba el lookup. El strip de prefix en `mapping.go` tolera ambos, pero la KEY del index debe coincidir exactamente.

13. **`detekt --config` REEMPLAZA (no mergea) la default config bundled.** Si pasás `--config detekt.yml`, todas las reglas default se vuelven `undefined` y no firean salvo que las listes explícitamente. Por eso `examples/bad-project/detekt.yml` lista solo 2 reglas (las que la smoke test verifica); NO esperar reglas default como `EmptyFunctionBlock`, `MagicNumber`, etc., a menos que las agregues explícitas. Forward-compat fix: usar detekt 2.x `extends:` field o una config base que herede de default.

14. **`arch-god-class` mapping verificado manualmente, NO via smoke test.** El stanza `complexity.TooManyFunctions` aunque listada explícita no dispara `arch-god-class` en BadCode.kt bajo `--config REPLACE` (detekt 1.23.x gotcha #13). El smoke test verifica solo 2/11 V1 live. El mapping se valida indirectamente con el scan de RickMortyApp (Phase 1 close-out, commit ee191e0 — 21 findings `arch-god-class` con score 0/100, 0 errors, 103 warnings).

15. **PathTrap fix (Bug A close-out)**: `detektrunner` ahora hace `filepath.Abs(opts.ProjectDir)` antes de invocar detekt. Sin esto, `--project-dir <relative>` causaba exit 1 sin SARIF (el bug original que salió durante smoke test). Ambos modos (runStandalone + runGradlew) usan `absProjectDir` consistentemente.

16. **JSON output pollution (Bug C close-out)**: cuando emitís `--json` o `--sarif`, detekt-cli imprime warnings JVM (`WARNING: sun.misc.Unsafe...`) a stdout. kdoctor ahora redirige `detektOpts.Stdout = io.Discard` en esos modos. Sin este fix, `kdoctor scan --json > report.json` producía JSON inválido (warnings prependos); tests downstream (GitHub Code Scanning, MobiAI Graph) fallaban silenciosamente.

---

## 10. On-ramp para la siguiente sesión IA

```
1. Lee ESTE archivo entero (docs/HANDOVER.md).
2. Lee docs/superpowers/plans/2026-07-19-kdoctor-implementation-plan.md (master plan).
3. Lee docs/v2/kdoctor-proposal-v2.md (contexto estratégico).
4. Mira el log: git log --oneline -15
5. Mira el catálogo: cat rules/metadata.json | jq 'length'  → 78
6. Valida baseline (5 minutos):
   export PATH="/c/Program Files/Go/bin:/c/Program Files/Eclipse Adoptium/jdk-25.0.3.9-hotspot/bin:/d/tools:$PATH"
   cd "C:\\Users\\Miguel\\Desktop\\doctor mobi ai fix"
   go vet ./... && go test ./... && go build -o kdoctor.exe ./cmd/kdoctor
7. Identifica qué tier está ABIERTA. Sección 8 lista estado. Tier 1 #1 cerrado; siguiente: **Tier 1 #2 Go-based regex detectors**.
8. Confirma con el usuario qué item abrir antes de empezar trabajo sustantivo.
```

---

## 11. Helpers de invocación frecuente

```bash
# Scan contra RickMortyApp (proyecto real del usuario)
./kdoctor.exe scan --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp

# Scan a JSON schema v3 (CI o MobiAI Graph)
./kdoctor.exe scan --json --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp

# Scan a SARIF 2.1.0 (GitHub Code Scanning)
./kdoctor.exe scan --sarif --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp \
  --out=detekt.sarif

# Regen del catálogo (tras editar scripts/genschema/main.go)
go run ./scripts/genschema -out rules/metadata.json

# Smoke test Tier 1 #1
go test -v -run TestEval ./scripts/evalprojects/...

# Formatear todo
gofmt -w .

# Doctor
./kdoctor.exe doctor

# Rules (listado)
./kdoctor.exe rules | tail -3

# Verificar prefix-strip del rulemap (regression guard Tier 1 #1)
go test -v -run TestMapPrefixStrip ./internal/core/rulemap/...
```

---

## 12. Convenciones de código

- **Naming IDs**: V1 prefijo = namespace o abreviación (`compose-*`, `coroutine-*`, `arch-*`, `sec-*`, `mem-*`, `dead-*`, `a11y-*`, `test-*`, `kmp-*`, `lifecycle-*`). Phase 1.5+ (5.11) usa cluster como prefijo completo (`complexity-*`, `error-handling-*`, `magic-numbers-*`, `naming-*`, `formatting-*`). Tests garantizan la convención.
- **Severity**: `error | warning | info`.
- **FixHint**: string accionable en imperativo; `omitempty` si vacío.
- **Commit messages**: conventional commits (`feat(scope): ...`, `fix(scope): ...`, `refactor(scope): ...`, `test(scope): ...`, `chore(scope): ...`, `docs(scope): ...`).
- **Spanish comments / English code** (mixed). Pragmatic. kdoctor identifiers en inglés.
- **Tests**: tests unitarios usan TestXxx; tests integration usan TestIntegration; E2E fixture-based tests viven en `scripts/evalprojects/main_test.go`.

---

**Última actualización**: cierre de Tier 1 #1 (commit `[new]` aún pendiente). Después: Tier 1 #2.

**Próxima sesión IA**: confirmar con el usuario antes de abrir Tier 1 #2 (Go-based regex detectors para las V1 planned sin detekt equivalente como `sec-log-pii`, `compose-remember-missing`, etc.).
