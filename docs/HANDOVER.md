# kdoctor — Handover document (continuidad entre sesiones IA)

> **Objetivo**: que cualquier IA (o persona) que llegue a este repo pueda leer este único documento y saber **exactamente** qué es kdoctor, qué está hecho, qué está pendiente, cómo validarlo, y cómo continuar.

> **Estado al cierre**: Tiers 1, 2, 3 + Fases 4 y 5 + **Round-2 polish completo (7/7 tareas)** cerradas ✅. Round-2 endureció los 7 archivos centrales del coderound-1 con regression guards, multi-context (positivo/negativo/edge), y manejo robusto de paths Windows/Unix/mixed + cache persistente del provider. **Próximo bloque sugerido**: Round-3 release prep (commit message audit, gofmt drift CI, README público, ejemplos tutorials) — ver sección 13 al final de este handover.

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
| **Tier 1 #2** | Go-based regex detectors en `internal/core/rules/` | ✅ **CERRADO** | Implementados `sec-log-pii`, `compose-missing-key`, `sec-webview-javascript-enabled`, `coroutine-dispatchers-hardcoded`. |
| **Tier 1 #3** | `--diff main` / baseline.xml | ✅ **CERRADO** | Implementados `internal/core/baseline` y `internal/core/diff`. Modificado `internal/cli/scan.go` para parsear y suprimir findings según flags. |
| **Tier 2 #4** | `kdoctor-gradle-plugin` | ✅ **CERRADO** | Subproyecto Gradle con task `kdoctorScan` no-invasivo creado en `plugins/kdoctor-gradle-plugin`. |
| **Tier 2 #5** | `kdoctor fix --ai` Quality-Focused | ✅ **CERRADO** | CLI auto-fix en Go, valida sintaxis con patchguard, genera fixes.md sin tocar código fuente. |
| Tier 3 #6 | HTML dashboard | ✅ Hecho | vite + tailwind sobre kdoctor scan --json; charts por cluster, score trend, integración MobiAI Graph |
| Tier 3 #7 | `kdoctor.config.yaml` overrides por equipo | ✅ Hecho | Cluster-aware severity + custom excludes + disable-rule per cluster |

**Orden de ejecución fijado por el usuario**: cerrar Tier 1 → 2 → 3 secuencialmente. Cada tier antes de empezar el siguiente libera un commit.

---

## 8b. Round-2 polish (post-cierre, en progreso)

Round-2 arrancó después del cierre de todos los tiers. **NO es trabajo nuevo de funcionalidad** — es endurecimiento defensivo del código existente, cerrar regresiones latentes, y pinear contratos con tests que sobreviven a refactors. Resultado buscado: el código entra en reserva para una v1.0 con confianza.

**Intención**: cerrar las 7 tareas en orden (#5 → #6 → #7) y luego auditoría final pre-release. **Si surge una nueva tarea durante el trabajo, añadir a esta sección con su propio número** (no resucitar tiers viejos ni meterlo en otras secciones). Marca ✅ al cerrar; nunca borres líneas — el historial en este archivo es trazabilidad.

**Convención de numbering**. Las tareas van del #1 al #7 secuencial. **Sub-tareas como `#2.1`** son **polish del #N inmediatamente anterior** que surgió tras cerrar el #N y re-analizar (e.g. feedback del code-reviewer o cambio de criterio). Próximo AI: si creas sub-tarea por polish/iteración, usa la siguiente libre (e.g. `#7.1` tras cerrar #7). Si es trabajo completamente nuevo y sustantivo (no sub-polish), asigna el siguiente número entero disponible (e.g. `#8` tras cerrar #7). Mantener el rationale arriba de la tabla con la numeración actual para que el lector verifique la coherencia.

| # | Tarea | Estado | Detalle |
|---|---|---|---|
| **1** | Tests rulemap restaurados + Bug 1 guard | ✅ cerrado | 7 tests del round-1 restaurados verbatim en `internal/core/rulemap/mapping_test.go`: `TestMapPrefixStrip`, `TestLoadRulesFromFixture`, `TestMapKnown`, `TestMapUnknown`, `TestMapIgnoresPlannedRules`, `TestMapStableOrderByLocation`, `TestLen`. + 1 test nuevo pin del Bug 1 multi-vendor: `TestBug1MultiVendorPrefixStrip` (verifica strip de varios prefijos `detekt.style.X`, `detekt.complexity.Y` antes del lookup). Cubre prefix-strip, prefix-sin-dot, prefix-trailing-dot, mapped-vs-unmapped, planned-status-skip, stable-order-by-location. |
| **2** | Stubs falsos de `examples/bad-project/BadCode.kt` + `report.json` regen | ✅ cerrado | Eliminados stubs `WebViewSettings` / `object Dispatchers` / `object Log` / `fun items(...)` que mintían sobre errores fatales. `report.json` regenerado con 6 findings reales (determinista cuando se corre con `report.json --out`). Antipatterns vuelven a `settings: Any` (built-in) que detekt 1.23.x reporta como `UnresolvedReference` warnings (NO error fatal de parse). Header del archivo documenta este contrato. `scoring-fixtures/bad.json` actualizado a banda `[75, 90]` con los 6 IDs en `mustIncludeFinding`. **Tests añadidos en este trabajo**: extensión de `internal/core/rulemap/mapping_test.go` con tests que pinean cambios estructurales del `rulemap` (lookup `byID` para native rules + función `ApplyOverrides` para cluster/rule-level severity override + excludes + off-disabling): `TestMapByIDForNativeRules` (verifica que reglas con `DetektRule=""` se indexan también por `ID` directo, sin falsos `unmapped:`), `TestApplyOverrides` (happy path con excludes + cluster/rule-level overrides), `TestApplyOverrides_SeverityChanges` (cluster override cambia severity sin drop), `TestApplyOverrides_GlobFallback` (globs `**/*.kt` matchean recursivamente), `TestApplyOverrides_CaseInsensitiveSeverity` (`"OFF"`, `"disabled"`, `"none"` dropean normalizado via ToLower), `TestApplyOverrides_EarlyReturn` (no-op si excludes=nil y overrides=nil). |
| **2.1** | Polish del #2 (README + cluster override vivo + tests de precedencia) | ✅ cerrado | `examples/bad-project/README.md` creado: qué vive en el fixture, cómo reproducir el smoke test, output esperado (tabla con schema version, score breakdown, IDs), section sobre UnresolvedReference warnings a stderr, forward-compat note sobre detekt 2.x. `examples/bad-project/kdoctor.config.yaml` extendido: cluster-level override `security: warning` (downcast error→warning, NO drop) añade cobertura viva de Tier 3#7 sin romper `mustIncludeFinding`. `rule-level sec-log-pii: error` gana por precedencia sobre cluster-level `security: warning` (score=82 determinístico, no 85). Tests nuevos: `TestApplyOverrides_ClusterWarningDowncast` + `TestApplyOverrides_RuleLevelWinsOverCluster` pinan el contrato en `internal/core/rulemap/mapping_test.go`. |
| **3** | Endurecer `examples/scoring-fixtures/bad.json` band `[75, 90]` + rationale durable | ✅ cerrado | Banda `[60, 85]` → `[75, 90]`. Centro 82.5 con score real 82 determinístico (post-overrides). Opción (a) `[80, 95]` (round-1 original) rechazada porque requeriría eliminar el override `security: warning`, perdiendo cobertura viva de Tier 3#7. `mustIncludeFinding` sin cambios (6 IDs: 3 detekt + 3 nativos: `coroutine-global-scope`, `dead-unused-import`, `compose-missing-key`, `sec-log-pii`, `sec-webview-javascript-enabled`, `coroutine-dispatchers-hardcoded`). `examples/scoring-fixtures/README.md` nuevo: schema invariants, tabla de trade-off (a) vs (b), justificación del headroom ±8, catálogo de `mustIncludeFinding`, "Cómo añadir un nuevo fixture", historial round-1 vs round-2. README es durable (no se pierde con squash/merge de PRs grandes como pasaría con rationale en commit message). |
| **4** | Patchguard lexer-aware en `internal/aifixer/patchguard/guard.go` | ✅ cerrado | Reescrito como mini-lexer Kotlin-aware con state machine + stack de 7 modos (`modeCode`, `modeLineComment`, `modeBlockComment`, `modeSingleString`, `modeRawString`, `modeCharLiteral`, `modeTemplate`). Ignora contenido dentro de strings (con escapes `\" \n \\`), raw strings `"""..."""`, char literals `'\''` (con escapes), templates `${expr}` (con snapshot `templateOpen` para reconocer el cierre), y comentarios line/block. Errores preservan formato round-1: `"unbalanced braces: net count is N"`, `"unbalanced parentheses: net count is N"`. **3 tests round-1 preservados verbatim** (`TestValidateValidCode`, `TestValidateUnbalancedBraces`, `TestValidateUnbalancedParentheses`) **+ 14 nuevos edge tests** (total 17): `TestValidateStringWithBraces`, `TestValidateUnbalancedBracesInsideStringIgnored`, `TestValidateRawStringWithBraces`, `TestValidateCommentLine`, `TestValidateCommentBlock`, `TestValidateCharLiteral`, `TestValidateCharLiteralEscaped`, `TestValidateTemplateSimple`, `TestValidateTemplateWithLambda`, `TestValidateTemplateWithNestedString`, `TestValidateStringWithEscape`, `TestValidateTemplateUnbalancedFails`, `TestValidateEmptyString`, `TestValidateUnbalancedBracesInsideBlockComment`. **Limitaciones documentadas en godoc**: templates anidados 2+ niveles no soportados, raw strings con `"""` interno terminan prematuramente. Si surge necesidad de nested templates, convertir `templateOpen` (int) a `[]int` snapshot stack. **Iteración mid-implementación**: el primer write tuvo bug de compilación `undefined: mode` línea 43 (faltaba `type mode int`). Fix con typed alias inline: `type mode int` y `const (modeCode mode = iota; ...)`. Si el lexer no compila tras refactor, causa típica es añadir nuevos modos a la const block sin extender el typed alias (Go no propaga el type entre constants iota). |
| **5** | `diff.go` y `baseline.go` matching por paths absolutos via `pathutil` package | ✅ cerrado | Nuevo paquete `internal/core/pathutil` con `NormalizePath(input, projectRoot)` + `SuffixMatch(candidate, pathToFind)` boundary-aware. Reescrito `diff.FilterFindingsByDiff` con nuevo `FilterFindingsByDiffWithRoot(findings, diffMap, projectRoot)`: pre-normaliza diffMap keys vía pathutil, luego exact-match o boundary suffix-match fallback. Reescrito `baseline.IsSuppressed` con nuevo `IsSuppressedWithRoot(finding, ids, projectRoot)`: parsea `<RuleShortName>:<Path>:<Signature>`, matching **one-directional finding→baseline** (sin symmetric regression que suprimia múltiples archivos del mismo nombre en distintas carpetas). `scan.go` pasa `wd` (projectDir) a las funciones WithRoot; además detecta git root vía nuevo `diff.GetGitRoot(wd)` para monorepo submodules, con **stderr warning explícito** `"warning: could not detect git root ..."` si falla. `pathutil` simplificado: usa sólo `filepath.Join` + `filepath.Clean` + `filepath.Abs` condicional al `filepath.IsAbs` host-recognize check (sin cross-platform cwd pollution). **Tests añadidos**: 12 en pathutil (`TestNormalizePath_*` × 8 cubriendo absolute Unix, absolute Windows drive, mixed slashes, relative joined to root, `..` resolution, empty input, relative sin root, drive letter con path, UNC; `TestSuffixMatch_*` × 4 cubriendo boundary table-driven + `BoundaryRegression` pinning del fix vs round-1 strings.HasSuffix fragility). 5 en diff (round-1 verbatim + 4 nuevos: `FilterFindingsByDiffWithRoot_RelativeJoinedToAbsolute`, `_BoundaryRegression` con `OtherFoo.kt` vs `Foo.kt`, `_WindowsPathMixedSlashes`, `TestNormalizePath_RoundTrip` helper, **y** `TestGetGitRoot_NotInGitRepo` pineando substrin `"git rev-parse"` en el error message para guard contra refactors perdiendo la causa del subprocess failure). 4 en baseline (round-1 verbatim con 5 sub-cases, + 3 nuevos: `TestIsSuppressedWithRoot_BoundaryRegression` con `OtherMyFile.kt`/`MyFile.kt.bak` rejection, `TestIsSuppressedWithRoot_ProjectRoot` con paths absolutizados vs relativos vs reglas distintas, `TestIsSuppressedWithRoot_EmptyRuleDefensive` cubriendo `finding.Rule=""` + empty-path baseline IDs `"UnusedImports::no-path"`). **Limitaciones documentadas en godoc**: dos diffMap keys normalizing al mismo path colisionan (raro en práctica); cuando `--project-dir` es subdirectory of git root, scan.go detecta automáticamente via GetGitRoot; cuando wd no está en repo git, stderr warning + fallback a wd preserva comportamiento round-1. |
| **6** | `qualityprompt.BuildPrompt` con slicing ±N líneas | ✅ cerrado | Nueva función `BuildPromptWithContext(finding, sourceCode, contextLines)` agrega en `internal/aifixer/qualityprompt/builder.go`. `BuildPrompt(finding, sourceCode)` **preservado verbatim** (round-1 backward compat, mismo template). Constantes: `DefaultContextLines = 10`, `genericChangeHint` (fallback si `Finding.FixHint` está vacío), `findingMarkerInline = "  <-- FINDING"` (marker que se concatena en la línea del finding). Header format: `File: <basename>` + `Issue at line N` + `Rule ID:` + `Cluster:` + `Severity:` + `Message:` (opcional) + `Change Hint:` (con fallback). Body format: `Lines X-Y of <basename> (line N marked):` con `kotlin` block, líneas numeradas con `%*d:` width-padded y marker en línea N. Helpers privados: `splitLines` (sin trailing empty entry), `sliceRange(findingLine, n, numLines) → (start, end)` con defensive clamping (line≤0 clamp a 1, line>numLines clamp a last, contextLines≤0 fallback a DefaultContextLines), `fileBase` (vía `path.Base` después de normalizar `\\` → `/` para Windows-style paths cross-platform), `numWidth` (1-6 chars). `cli/fix.go` wired con nuevo flag `--context-lines int` (default 10) que pasa N a `BuildPromptWithContext`. **Tests añadidos (12 nuevos + 1 round-1 verbatim = 13 total)**: `TestBuildPromptWithContext_HappyPathAtMiddle` (finding=line 12, N=10, block 2-22 con marker verificado, off-range excluded, header pinning `Change Hint` literal usando constant); `_ClampsAtFirstLine` (line=1 → "Lines 1-11"); `_ClampsAtLastLine` (line=20 last → "Lines 10-20"); `_SingleLineFile` (file=1 línea → "Lines 1-1"); `_DefensiveClampOutOfRangeLine` (line=999 archivo 10-line → "Lines 7-10"); `_DefensiveClampZeroOrNegativeLine` (line=0 → "Lines 1-11", header echoes zero para LLM nota data quality issue); `_EmptySource` (source="" → "(empty source)" placeholder); `_FallbackHintWhenFixHintEmpty` (FixHint vacío → `genericChangeHint`); `_DefaultContextLinesWhenNonPositivePass` (N=0/-5 → fallback 10); `_HeaderAbsolutePathToBasename` (4 variantes: Unix, Windows backslash, mixed, relative — todas output `App.kt` después de backslash normalization); `_RoundTripsAndDoesNotLeakOffRangeSourceLines` (50-line file, line=25, N=5 → block 20-30). Unit tests aislados: `TestSliceRange` (8 casos del pure helper con trace math documentado en comments) + `TestSplitLines` (6 casos del line-splitter). **Limitaciones documentadas**: `contextLines=1e6` no tiene upper cap (math clamps correctamente pero waste work); marker inline `  <-- FINDING` se concatena DESPUÉS del source, no en línea siguiente (cosmetic trade-off acceptable para Kotlin idiomático).
| **7** | `claude.go` verificación runtime del flag `-p` con cache persistente | ✅ cerrado | `internal/aifixer/provider/claude.go` reescrito con runtime verification + cache + parsers puros. `ClaudeCacheTTL = 24 * time.Hour` (constante). `type ClaudeCache { DetectedAt time.Time; Version string; SupportsP bool }` en JSON at `~/.kdoctor/cache/claude-version.json` via `os.UserHomeDir`. **Pure parsers (testeables sin exec)**: `parseSupportsP(string) bool` busca `-p`, `--print`, `--non-interactive` con whitespace-boundary matching via `matchesFlag(s, fg)` helper — pinea false positives (-pdf, --printed, file-p); `parseVersion(versionOut, helpOut) string` strips `Claude`/`claude` prefix + leading `v` + valida via `looksLikeVersion(s)` (split on `.`, all-numeric segments — pre-releases excludidas por diseño); `defaultCachePath()` retorna `<home>/.kdoctor/cache/claude-version.json`. **Orchestration**: `ensureCapabilities(cachePath, ttl, detect)` lee cache fresh con supports=true → nil; fresh con supports=false → return `formatSupportsPMissing(version, cachePath)`; miss/stale → ejecuta `defaultDetection` (que invoca `claude --version` + `claude --help`, tolera uno fallando), persiste via `saveCache`, returnea según resultado. `Fix(prompt)` ahora llama `ensureCapabilities` ANTES de escribir temp file — side-effect deferred till pre-check passes. **Atomic write**: `saveCache` usa temp file + `os.Rename` (evita torn-write en crash mid-write). **Best-effort persistence**: `_ = saveCache(...)` no bloquea scan si disk permission denied. **DetectionFunc inyectable**: `var defaultDetection DetectionFunc` permite tests sin binary real. **Error UX**: `formatSupportsPMissing` retorna mensaje accionable con version detectada, requirement `v1.0.0+`, upgrade hint `npm update -g @anthropic-ai/claude-code`, y cache reset instruction. **Tests añadidos (15 nuevos)**: `TestParseSupportsP` (table-driven 9 cases incl. false-positive guards `-pdf`/`--printed`/comma-bounded), `TestMatchesFlag` (micro-test boundary 9 cases), `TestParseVersion` (8 cases incl. fallback a helpOutput cuando versionOutput vacío, pre-release excluded), `TestLooksLikeVersion` (8 cases), `TestLoadSaveCacheRoundtrip`, `TestLoadCacheStaleReturnsNil`, `TestLoadCacheMissingFileReturnsNilNoError`, `TestSaveCacheAtomic` (verifies `.tmp` file no sobrevive post-rename), `TestEnsureCapabilities_HitsFreshCache` (detector NOT called), `TestEnsureCapabilities_DetectsOnMiss`, `TestEnsureCapabilities_FailsOnNegative` (negative result persisted too), `TestEnsureCapabilities_NegativeCacheRespectedUntilTTL`, `TestEnsureCapabilities_DetectorErrorSurfaced`, `TestEnsureCapabilities_StaleCacheTriggersRedetect`, `TestFormatSupportsPMissing`, `TestDefaultCachePath`. **Limitaciones documentadas en godoc**: pre-releases `1.2.3-beta` excluded por `looksLikeVersion` (sólo numerales); `defaultDetection` no propaga non-fatal exit errors de `claude --help` (silently treats empty helpOutput como success — logged en backlog); cache writability failures swallowed best-effort (logged en backlog).

**Trabajo acumulado round-2**:
- 2 archivos README nuevos (`examples/bad-project/README.md`, `examples/scoring-fixtures/README.md`).
- 2 archivos JSON actualizados (`examples/bad-project/report.json` regenerado, `examples/scoring-fixtures/bad.json` banda cambiada).
- 1 archivo YAML modificado (`examples/bad-project/kdoctor.config.yaml` con cluster override vivo).
- 1 archivo Go reescrito (`internal/aifixer/patchguard/guard.go` mini-lexer).
- 2 archivos de tests extendidos (`internal/core/rulemap/mapping_test.go`, `internal/aifixer/patchguard/guard_test.go` con 13 nuevos edge tests).
- 1 archivo `.kt` purgado de stubs falsos (`examples/bad-project/src/main/kotlin/bad/BadCode.kt`).

**Bonus/feedback de reviewers (no bloqueante, en backlog)**:
- Vaciar risk: cobertura gap en overrides (empty string, whitespace) — dejar como TODO en godoc de `ApplyOverrides`.
- Forward-compat notes sobre detekt 2.x son especulación pura (la spike del R&D es código caliente) — cambiar "will become" por "may become" cuando sea posible.
- Test-YAML coupling (YAML referencia nombres literales de tests) — referenciar la función `ApplyOverrides` en godoc no el test name (más durable).
- README score=82 drift potential — snapshot-match el report.json en CI.

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
7. Identifica qué fase está ABIERTA. Sección 8 lista estado de tiers (todos cerrados). **Sección 8b lista el round-2 polish** — round-2 también cerrado (7/7 tareas). **Sección 13 lista el round-3 release prep** — esa es la fase abierta sugerido. Confirmar con el usuario antes de empezar trabajo sustantivo. Si el usuario trae trabajo nuevo, añadirlo como nueva sección en este HANDOVER.md antes de empezar (no meta trabajo nuevo en `Gotchas` ni reabra tiers viejos).
8. Confirma con el usuario qué item del round-2 abrir antes de empezar trabajo sustantivo. Si el usuario menciona trabajo nuevo fuera del round-2, añadirlo como nueva sección en este HANDOVER.md antes de empezar (no meta trabajo nuevo en `Gotchas` ni reabra tiers viejos).
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

# Round-2 polish — tests específicos por tarea cerrada
go test -v -count=1 -run 'TestApplyOverrides_' ./internal/core/rulemap/...   # Tareas #1, #2.1 (precedencia overrides)
go test -v -count=1 ./internal/aifixer/patchguard/...                        # Tarea #4 (lexer-aware, 16 tests)
go test -v -count=1 -run TestEval ./scripts/evalprojects/...                 # Tarea #2, #3 (smoke test bad/good project con band [75, 90])
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

**Última actualización**: Round-2 polish 100% cerrado (tareas #1–#7 ✅) + **Round-3 release prep #8 + #9 + #9.1 + #9.2 cerrados ✅**. Tag v1.0.0 frozen en d8e90dc (anchor symbolic de round-2); HEAD ahora 4 commits ahead (5899fc8 drift fix, c17a15a CI workflows, 17f7831 ci hardening, af2669b YAML hotfix). Total commits: 14. Validación final post-Round-3 #9.2: `gofmt -l .` empty ✅, `go vet ./...` exit 0 ✅, `go test -count=1 ./...` 19 packages PASS ✅, `go build -o kdoctor.exe ./cmd/kdoctor` BUILD_OK ✅. Solo ambient cruft uncommitted (10 dirs/files pre-Phase-1.5). HEAD = `af2669b088c6419481a6044fde5ec60179533442`.

**Próxima sesión IA**: confirmar con el usuario el bloque de **Round-3 release prep** (sección 13) antes de empezar. No reabra tiers viejos — todas las tiers del plan maestro están cerradas; cualquier nuevo trabajo es round-3+ o trabajo ad-hoc post-release. Antes de tocar código: leer sección 8b entera (round-2 history) + 9 (gotchas) + 10 (on-ramp) + 13 (round-3 backlog) + esta nota de cierre.

---

## 13. Round-3 release prep (siguiente bloque sugerido)

Round-2 fue **endurecimiento defensivo del coderound-1**. Round-3 es **release prep para v1.0**: convertir el proyecto en algo publicable, no seguir añadiendo features. Tareas preliminares propuestas (no confirmadas), ordenadas por prioridad y dependencia:

| # | Tarea | Estado | Detalle preliminar |
|---|---|---|---|
| **8** | Commit message audit + squash del round-2 | ✅ cerrado | 11 commits landed (1352f5b → d8e90dc) con conventional-commits (feat/refactor/test/docs/chore(scope): ...) + Codebuff footer 🤖/Co-Authored-By. Tag annotated `v1.0.0` apuntando a d8e90dc. Commit `d8e90dc` adicional cierra orphan files `internal/aifixer/provider/{provider.go,stubs.go}` (pre-existing Tier 2#5 closure) para atomicidad de package — mensaje honest sobre scope mix. Todos los commit messages explican el *por qué*, no sólo el *qué*. |
| **9** | `gofmt -w .` + `go vet` + CI workflow fast gate | ✅ cerrado | **Commit `5899fc8`** `chore(fmt): fix gofmt drift in 4 files` — `gofmt -w .` post-round-2. Drift detectado en: `internal/aifixer/patchguard/guard.go` + `guard_test.go` (round-2 #4 lexer), `internal/core/config/config.go` (round-2 #0 Excludes + gofmt), y `internal/core/grader/grader.go` (pre-existing Tier 1 closure, folded atómico). Post-fix: `gofmt -l .` returns vacío. **Commit `c17a15a`** `ci(workflows): preserve mobiai-install-test.yml + add ci.yml fast gate (Round-3 #9)` — 2 archivos en `.github/workflows/`: `ci.yml` (NUEVO fast gate: gofmt drift + vet + race-tests + build + tidy-drift, ~2 min warm-cache vs ~3min cold) + `mobiai-install-test.yml` (PRESERVADO verbatim del draft pre-existente, slow integration ~5min con `npm install -g mobiai@latest`). Concurrency `ci-fast-${{ github.ref }}` cancela in-progress runs. Reemplazado `staticcheck` por `go test -race -count=1` (más value/CI-minute para el round-2 hardening). Si en el futuro necesitamos lint de idiomatic bugs, agregar `golangci-lint` step. |
| **10** | README.md rewrite para público | ⏳ sugerido | README actual (interno) cubre onboarding hacia la IA. Rewrite con: hero (qué es kdoctor), quick start (3 comandos), feature grid, comparison vs detekt/react-doctor, badges, screenshot del HTML dashboard Tier 3#6. **Próximo paso natural** tras cerrar #9 + #9.1. |
| **9.1** | Polish del #9 (CI hardening: paths-ignore + mobiai version pin + error-hardening) | ✅ cerrado | **Commit `17f7831`** `ci(workflows): pin mobiai version + add paths-ignore + set -euo pipefail (Round-3 #9.1)` post code-reviewer. Cambios: (a) mobiai-install-test.yml: `npm install -g 'mobiai@^1.0.0'` (semver range, no `@latest` mutable; cierra 🔴 supply-chain risk reviewer-flagged); (b) ambos workflows ci.yml + mobiai-install-test.yml: añadir `paths-ignore: ['**.md', 'docs/**', 'CHANGELOG.md', 'examples/**']` (ahorra ~2-3 min CI per push de solo-docs); (c) mobiai-install-test.yml: `set -euo pipefail` + drop de los `|| echo "..."` / `|| true` swallowers para que errores reales sean visibles (cambia `echo` a `::error::` annotations). Tag v1.0.0 ahora stale (HEAD ahead) — mantener como anchor symbolic local; crear v1.0.1 cuando se decida push a remote. |
| **9.2** | Hotfix del #9.1 (YAML header smushed by str_replace) | ✅ cerrado | **Commit `af2669b`** `ci(workflows): fix broken YAML header in ci.yml (Round-3 #9.2 hotfix)`. El str_replace del #9.1 colapsó accidentalmente el `\n\n` entre `name: CI Fast Gate` y el comment block `# Round-3 #9.1 polish: ...`, generando valor invalido `"name: CI Fast Gate# Round-3 #9.1 polish: paths-ignore filter..."`. Reescrito ci.yml via `write_file` con estructura limpia: top-level comment block precede `name:`, Round-3 #9.1 polish comment block precede `on:`, paths-ignore filter bajo `on:` intacto. mobiai-install-test.yml NO afectado (sus str_replaces del #9.1 eran targeted sin block insert at header). Manual YAML structure validation: cada top-level key en su propia línea ✅. Tres commits (#9 + #9.1 + #9.2) acceptable atomicidad para local symbolic; en remote push usar `git rebase -i HEAD~3 --autosquash` para colapsar fixups antes de PR. |
| **11** | Tier 2 #5 `kdoctor fix --mode auto` (patch apply + rollback) | ⏳ sugerido | Round-2 #4 endureció el **patchguard** (validación). Cuando corre, sólo emite `fixes.md`. Para v1.0: implementar `--mode auto` que toma el patch del LLM, lo aplica, valida el archivo reparsed Kotlin syntax (kdoctor tiene.kt fixture parser?), y rollback on error. Beneficio: kdoctor no es solo "informational" sino "hands-off repairer". |
| **11** | Tier 2 #5 `kdoctor fix --mode auto` (patch apply + rollback) | ⏳ sugerido | Round-2 #4 endureció el **patchguard** (validación). Cuando corre, sólo emite `fixes.md`. Para v1.0: implementar `--mode auto` que toma el patch del LLM, lo aplica, valida el archivo reparsed Kotlin syntax (kdoctor tiene.kt fixture parser?), y rollback on error. Beneficio: kdoctor no es solo "informational" sino "hands-off repairer". |
| **12** | Tests E2E con proyecto Android real | ⏳ sugerido | Round-1 smoke test usa `examples/bad-project` (15-line seeded fixtures). Round-2E2E con RickMortyApp (real user project, read-only) revealaría drift entre fixture y realidad (Map keys, SARIF quirks, detekt 1.23.x vs 2.x). Test runner: `scripts/e2e-rick-morty/main.go` con `--smoke` flag. |
| **13** | `make` build targets + Dockerfile | ⏳ sugerido | Hoy `go build -o kdoctor.exe ./cmd/kdoctor` es el único modo. Makefile da `make build`, `make test`, `make lint`, `make smoke`, `make e2e-rick-morty`. Dockerfile minimo (golang:1.26.5-alpine → 12mb binary) para CI y distribución. |
| **14** | `kdoctor init` project bootstrapper | ⏳ sugerido | Tier 1+ ran scaffolding manual. `kdoctor init` debería: detectar tipo de proyecto (kmp/jvm/cmp/android), generar `kdoctor.config.yaml` con cluster-aware defaults, generar `detekt.yml` con REPLACE-aware config de defaults, generar `.gitignore` entries, generar README badge snippet. Tier 2 #4 plugin Gradle ya hace parte — consolidar. |
| **15** | Release notes v1.0 + semver tag | ⏳ sugerido | Round-3 closes con `git tag v1.0.0`, `CHANGELOG.md` con resumen de lo cerrado en tiers 1-3 + round-2-3, GitHub release con binaries pre-built (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64). |
| **16** | MobiAI Graph integration end-to-end | ⏳ sugerido | `--mobiai` flag ya existe (Tier 1 #3 output). Round-3: integrar con MobiAI Graph API (POST a `/graph/findings` con token), dashboard embedded en MobiAI Web (no standalone HTML), SSO via MobiAI session. Beneficio: kdoctor es first-class citizen del MobiAI ecosystem. |

**Sub-tareas adicionales posibles (si surge polish durante round-3)**:
- 🔴 Vitest-style test snapshot matching (round-2 #3 README score=82 drift) → snapshot de report.json en CI.
- 🔴 `scripts/genschema` self-update (gen-tests relationship).
- 🔴 Mocking del ClaudeProvider en tests integration (sacudir el global `defaultDetection`) → injection via interface.

**Convenciones round-3**:
- Mismas convenciones que round-2: tabla cerrada, no borrar filas ✅ cerradas, añadir nuevas filas con `#N` siguiente libre.
- Si una tarea round-3 se vuelve polish, usa `#N.1` style.
- Sub-tareas adicionales que surjan: añadir como puntos sueltos al final del bloque en lugar de inflar la tabla.

**Trigger para arrancar round-3**: usuario confirma verbalmente o el flujo explícito "sigue con round-3". Si no, mantener proyecto en standby con round-2 cerrado a la espera.
