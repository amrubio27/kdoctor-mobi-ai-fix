# Changelog

All notable changes to kdoctor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Nota sobre el versionado.** Los tags de este repo quedaron desordenados:
> `v1.0.0` (2026-07-21) es **anterior** a la serie `v0.1.0`…`v0.6.0`, y falta
> `v0.3.0`. Los tags publicados no se borran, porque alguien puede haberlos
> fijado.
>
> Esta release salta a **`v1.1.0`** precisamente para dejar el lío atrás: al
> superar a `v1.0.0`, el tag más alto en semver vuelve a ser también el más
> reciente en el tiempo. Los dos criterios —ordenar por versión y ordenar por
> fecha, que es lo que hace el *latest* de GitHub y por tanto `install.sh` e
> `install.ps1`— coinciden otra vez. A partir de aquí, semver normal.

## [v1.1.0] — 2026-09-22

### Fixed — el Health Score no medía lo que decía medir
- **Un proyecto real de 16,6 KLOC daba 0/100**, y un 0 no distingue "mejorable"
  de "desahuciado". Tres causas, encontradas midiendo:
  - `sqrt(KLOC)` **no normaliza**. Los hallazgos crecen linealmente con el
    tamaño, así que dividir por la raíz deja un residuo creciente: con la misma
    densidad de deuda, 4 KLOC daba 15 y 16/60/200 KLOC daban 0. Ahora se divide
    por KLOC, que es lo que "calibrado por KLOC" debía significar.
  - **Las reglas de diseño disparaban dentro del código de test.** En ese
    proyecto, los 36 hallazgos críticos de arquitectura venían **todos** de
    fuentes de test: un test de ViewModel importa legítimamente de la capa de
    datos. Las reglas de arquitectura y diseño ya no se evalúan ahí; las de
    seguridad sí, porque una credencial filtrada lo está en cualquier fichero.
  - **Los hallazgos críticos eran inmunes al tamaño.** Dos reglas al tope fijaban
    30 puntos con 2 KLOC o con 500, así que el score dejaba de responder "cuánta
    deuda hay" para responder "¿tienes dos reglas críticas?". Ahora se dividen por
    `sqrt(KLOC)` mientras el resto se divide por KLOC: siguen dominando sin clavar
    el resultado.
- Otro mapeo erróneo del mismo tipo que los de Compose:
  `InvalidPackageDeclaration` (el fichero no está en la carpeta que declara su
  paquete) apuntaba a `arch-internal-in-public-api` (fuga de tipos internos en la
  API pública). Como ese id es `architecture/error`, contaba como crítico no
  diluible y ponía un techo duro de 15 puntos a cualquier proyecto con tres
  ficheros así, normalmente de test.

Calibrado contra referencias públicas, no eligiendo constantes a ojo:

| proyecto | score |
|---|---|
| `skydoves/pokedex-kmp` | 85 |
| IoTInventoryApp (16,6 KLOC) | 84 (antes 0) |
| `bad-project` (fixture) | 55 |
| `ArisGuimera/Curso-TestingAndroid` | 46 |
| `good-project` (fixture) | 100 |

Límite conocido: por debajo de ~1 KLOC el score no es fiable. El divisor tiene
suelo en 1.0, así que un fixture de 120 líneas no se divide por 0,12, y a ese
tamaño un puñado de saltos de línea ausentes mueve mucho el número. La densidad
necesita código que medir.

### Fixed — funcionalidad anunciada que no funcionaba
- **`--fail-below` no tenía efecto salvo en consola.** El `switch` de formato de
  salida hacía `return` en cada rama, así que el quality gate (y el volcado a
  MobiAI) sólo se alcanzaban en modo consola. `kdoctor scan --json --fail-below N`
  —el uso exacto de CI, del servidor MCP y del plugin Gradle— **siempre salía con
  exit 0**.
- **`scan --mobiai --json` no escribía nada**, que es justo el flujo documentado
  en `docs/integrations/mobiai.md`. Mismo `return` prematuro.
- **`fix --mode=auto` podía destruir código.** Si el proveedor de IA fallaba, se
  fabricaba un parche `// Provider failed`; ese comentario tiene las llaves
  balanceadas, así que patchguard lo aceptaba y se escribía **encima del código
  real**.
- **SARIF con `ruleId` inválido.** Las declaraciones `rules[]` usaban `f.Rule` y
  los `results[]` emitían `f.ID`, de modo que GitHub Code Scanning recibía ids
  que no correspondían a ninguna regla declarada. Además se descartaban los
  findings sin `Rule`.
- **Rutas inconsistentes en los informes.** Los detectores nativos emitían rutas
  relativas y detekt absolutas (`file:///C:/Users/...`), así que el SARIF subido a
  GitHub no resolvía con nada y cada JSON filtraba la ruta local. Nuevo
  `pathutil.RelativeToProject`, aplicado antes de cualquier filtrado o informe.
- **`score.failBelow` se parseaba y nunca se leía.** Ahora se aplica cuando
  existe `kdoctor.config.yaml`; un `--fail-below` explícito sigue teniendo
  prioridad, y un proyecto sin config no gana un umbral por sorpresa.
- **Errores silenciados**: un `kdoctor.config.yaml` mal formado saltaba todos los
  overrides sin avisar; la consola colaba un `ESC[0m` en salidas redirigidas.

### Fixed — la cadena de detekt, que en la práctica nunca corría
- **El init-script de Gradle usaba una DSL inexistente** (`detekt { xml = false }`),
  y `Detect()` elegía la vía gradlew por la mera existencia de `./gradlew` —que
  todo proyecto Android/KMP tiene—. Resultado: detekt no producía findings casi
  nunca, y con él se perdían las reglas que delega.
- **Resolución de Java que evita las versiones incompatibles.** Un JDK 25 en el
  PATH mata a detekt (`IllegalArgumentException: 25.0.1`) antes de analizar nada.
  Nuevo `internal/core/toolchain`: busca en `KDOCTOR_JAVA`, `JAVA_HOME`, PATH, el
  JBR de Android Studio y `~/.gradle/jdks`, y elige uno **compatible** (11-21).
- **`--config` reemplazaba el ruleset por defecto** en lugar de ampliarlo, así que
  sólo disparaban las reglas listadas en el fichero: un `detekt.yml` mínimo daba
  1 finding en 16 KLOC. Añadido `--build-upon-default-config`.
- **Falsos positivos que enterraban la señal**: 221 `FunctionNaming` en ficheros
  de test, porque Kotlin nombra los tests con backticks. Corregido en la
  auto-configuración y en la plantilla de `kdoctor init`.
- Un `detekt.yml` de proyecto que detekt rechaza (exit 3) ya no anula el escaneo:
  se reintenta con la configuración de kdoctor y se avisa de cómo regenerarlo.

### Added
- **Provisión automática de detekt.** Se descarga de Maven Central a
  `~/.kdoctor/tools/`, con versión pineada y verificación SHA-256. Un tercero
  instala kdoctor y escanea sin configurar nada.
- **Plugin de reglas Compose** (`io.nlopez.compose.rules` 0.4.22) provisionado
  igual. Son las reglas que diferencian a kdoctor de un detekt pelado.
- **`kdoctor fix` emite un plan de remediación** en JSON o Markdown —ventana de
  código, `fixHint` y rango exacto de líneas— para que lo aplique el agente que
  ya tiene un modelo. kdoctor **no llama a ningún LLM ni edita ficheros**.
  `kdoctor fix --validate` pasa patchguard sobre lo que el agente escribió.
- **`kdoctor doctor` reescrito**: informa de Java, detekt, Gradle y **cuántas
  reglas podrá evaluar el próximo escaneo**. `--provision` adelanta la descarga.
- Flags `--detekt-mode auto|standalone|gradle` y `--no-download`.
- Filtro por cluster en el informe HTML, combinable con el de severidad.

### Changed
- **Ruta del módulo Go**: `github.com/adkd/adkd` a
  `github.com/amrubio27/kdoctor-mobi-ai-fix`. `go install ...@latest` ya funciona.
- **Catálogo: 100 a 116 reglas** (64 activas: 20 nativas + 44 vía detekt). Cuatro
  nombres Compose del catálogo **no existían en el plugin**, y dos reglas
  compartían clave `detektRule`, con lo que `BuildIndex` hacía inalcanzable a una
  de cada par. El conteo hardcodeado se sustituyó por `validateCatalog()`, que
  comprueba ids duplicados, severidades, estados y colisiones.
- **Release**: añadidos `linux/arm64` (que `install.sh` ya ofrecía, con **404
  garantizado**) y `kdoctor-mcp`; versión inyectada por ldflags; `sha256sums.txt`;
  Go 1.23.x alineado con CI.
- `SKILL.md` alineado al formato real de MobiAI, con sección *Pre-flight*.
- Los tests de integración dejan de auto-saltarse: probaban con una ruta
  hardcodeada de otra máquina, así que **nunca corrían** para nadie más. Eran los
  únicos que ejercitaban un escaneo real de principio a fin.

### Removed
- `internal/aifixer/provider` (invocaba `claude --file`, una flag inexistente, y
  cuatro stubs sin llamadores) y `internal/aifixer/applier` con `applyFix`.
- `internal/jvmrunner`, el spike que el ADR-0007 canceló, y `html.RenderHTMLJSON`.
- Rutas de máquinas ajenas, los artefactos de build de Gradle versionados y
  `fixes.md`.

## [v0.6.0] — 2026-09-03

### Fixed & Enhanced
- **Detekt Binary Resolution (`--detekt-bin`)**: Explicit `--detekt-bin` is now authoritative and takes absolute precedence over `./gradlew`.
- **Cross-Platform `.jar` Execution**: Passing a `.jar` to `--detekt-bin` now launches correctly via `java -jar <jar>` across Windows/macOS/Linux.
- **Detekt 2.x Compatibility**: Added dynamic detekt version probing; `--max-issues` flag is omitted for Detekt 2.x CLI runs. Added alias support between `UnusedImports` (1.x) and `UnusedImport` (2.x).
- **False Positive Elimination in Clean Architecture**:
  - `arch-presentation-depends-on-data`: Excluded `@OptIn(Experimental*Api)` annotations and Compose experimental APIs from being falsely flagged as data implementations.
  - `error-handling-layer-mapping`: Inspected `catch` bodies to permit domain error mapping (`Result.failure`, `AppError`, `Either`, domain exceptions and rethrows).
- **Health Score Anti-Dilution Damping**: Individual critical rule penalties are capped at 15.0 pts max, preventing false-positive noise from artificially dropping valid projects to 0/100.
- **Contrato Clean Architecture & MVI**:
  - Added new rule `arch-repository-impl-interface` ensuring `*RepositoryImpl` implements a domain `*Repository` interface.
  - Strengthened `arch-udf-sealed-events` to detect public mutator methods (`on*Changed`, `set*`, `update*`) in ViewModels.
- **CLI & UX Hardening**:
  - `kdoctor init` is now completely idempotent when configuration files exist.
  - Added fail-soft scanning mode falling back gracefully to native architecture rules if Detekt fails.
  - Detailed scanner strategy logging when running with `--verbose`.

## [v0.5.0] — 2026-08-03

### Changed & Fixed

- **Rules Updater Repository Target**: Fixed `kdoctor rules update` default URL to point to `https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/rules/metadata.json`.
- **Environment Override for Rules URL**: Added `KDOCTOR_RULES_URL` environment variable support to easily override the remote rules catalog endpoint.

## [v0.4.0] — 2026-08-03

### Added & Refactored (Clean Architecture, Compose Performance & Continuous Health Score)

- **Continuous Health Score Algorithm (v0.4)**: Eliminated the 300-line cliff with smooth continuous scaling ($\max(1.0, \sqrt{\text{KLOC}})$). Cluster-weighted severity scaling (Security 2.0x, Architecture 1.5x, Coroutines/Memory 1.25x, UI 0.75x). Protected critical Security/Architecture errors from KLOC dilution, capped total Info penalty at 10.0 pts, and applied diminishing returns per (File, Rule) pair.
- **Clean Architecture & SOLID Rules**: Added 11 new native Go detectors:
  - `arch-presentation-depends-on-data`: Prevents presentation layer from depending directly on data layer.
  - `arch-viewmodel-contract`: Enforces UseCases in ViewModels (permits interface Repositories ONLY for passthrough UseCases).
  - `arch-usecase-contract`: Enforces single public function (`invoke`/`execute`) and domain interface dependencies.
  - `arch-misplaced-domain-logic` & `arch-misplaced-data-logic`: Detects domain logic in UI/ViewModel and data/SQL logic in UseCases.
  - `arch-model-mapping-leak`: Enforces `DataModel` -> `DomainModel` -> `UiModel` mapping pipeline.
  - `error-handling-layer-mapping`: Maps Data exceptions to Domain Result and explicit `UiState.Error`.
  - `arch-viewmodel-mvi-suggestion`: Recommends MVI pattern when managing 3+ disjoint StateFlows.
  - `testability-direct-instantiation`: Detects direct instantiation of concrete repositories/services without DI.
  - `arch-udf-sealed-events`: Suggests `sealed interface UiEvent` for Unidirectional Data Flow.
- **Compose Performance & UI**:
  - `compose-graphics-layer`: Enforces `graphicsLayer { ... }` or lambda modifiers for dynamic animation states.
  - `compose-heavy-composable`: Warns on large `@Composable` functions (>80 lines) for UI decomposition.
  - `compose-recomposition-optimizer`: Detects unstable collection parameters.
  - `ui-hardcoded-strings`: Detects hardcoded UI strings in `Text(...)`.

## [v1.0.0] — 2026-07-19

### Initial Stable Release

`kdoctor` is a CLI tool (Go 1.22+) that audits the quality of Android / KMP / CMP
projects. It wraps `detekt` SARIF 2.1.0 output, maps Detekt rules into a curated
`kdoctor` catalog (78 rules: 11 V1 live + 53 V1 planned + 14 default-detekt
mappings Phase 1.5), computes a **Health Score 0-100** (errors -5, warnings -2,
info -0.5), and emits rich console / JSON / SARIF reports.

### Added (Tiers 1–3 + Phases 4–5)

- **Tier 1 #1** — Seeded `examples/bad-project` fixture + smoke test runner
  in `go test`. 11 V1 priority rules live in catalog.
- **Tier 1 #2** — Four Go-based regex detectors for rules without detekt
  equivalence: `sec-log-pii`, `compose-missing-key`,
  `sec-webview-javascript-enabled`, `coroutine-dispatchers-hardcoded`.
- **Tier 1 #3** — `--diff <ref>` filtering + `--baseline <path>` suppression
  for PR-scoped scans (`internal/core/diff`, `internal/core/baseline`).
- **Tier 2 #4** — Gradle plugin `kdoctor-gradle-plugin` exposing
  `kdoctorScan` task (non-invasive, user-opt-in).
- **Tier 2 #5** — `kdoctor fix --ai` LLM-driven fixer with Claude provider,
  patchguard validator, and quality-focused prompt templates. `--mode auto`
  extracts the code block, applies the patch, validates the patched file,
  and rolls back on validation failure.
- **Tier 3 #6** — HTML dashboard (vite + tailwind): cluster distribution charts,
  score trend, integration with MobiAI Graph via `--mobiai` flag.
- **Tier 3 #7** — `kdoctor.config.yaml` live overrides: cluster-aware severity,
  custom excludes, per-rule/cluster disable.

### Hardened (Round-2 Polish)

- **#1** — Rulemap: 7 round-1 tests restored verbatim + Bug 1 multi-vendor
  prefix-strip regression guard (`TestBug1MultiVendorPrefixStrip`).
- **#2** — Purged fake stubs (`WebViewSettings`, `Dispatchers`, `Log`,
  `items(...)`) in `examples/bad-project/BadCode.kt` that caused parse
  errors; regenerated `report.json` with 6 deterministic findings.
- **#2.1** — Added `examples/bad-project/README.md` + cluster-level override
  alive in `kdoctor.config.yaml` (`security: warning`) + ApplyOverrides
  precedence-test pinning in `rulemap`.
- **#3** — `examples/scoring-fixtures/bad.json` score band updated to
  `[75, 90]` (deterministic score 82 post-overrides) with durable rationale
  in `examples/scoring-fixtures/README.md`.
- **#4** — Patchguard rewritten as Kotlin-aware mini-lexer with state machine
  (7 modes: code, line/block comments, single/raw strings, char literals,
  templates); ignores braces/parens inside strings; 14 new edge tests
  preserving round-1 verbatim.
- **#5** — New `internal/core/pathutil` package: `NormalizePath` +
  `SuffixMatch` boundary-aware. `diff.FilterFindingsByDiffWithRoot` and
  `baseline.IsSuppressedWithRoot` for absolute-path matching; git-root
  auto-detection in `scan.go` with explicit stderr fallback warning.
- **#6** — `qualityprompt.BuildPromptWithContext` ±N context-line slicing
  (header + body + `<-- FINDING` marker inline); `--context-lines` CLI flag
  (default = 10) wired in `kdoctor fix`.
- **#7** — Claude provider: runtime verification of `-p` (or `--print` /
  `--non-interactive`) flag, 24-hour persistent cache at
  `~/.kdoctor/cache/claude-version.json`, pure parsers
  (`parseSupportsP`, `parseVersion`, `looksLikeVersion`); 15 new unit
  tests covering cache TTL, atomic write, detector injection, error UX.

### Release Prep & Public Documentation (Round-3)

- **#8** — Commit message audit + atomic history for v1.0.0; 14 commits
  landed from Tier 1 #1 through Round-3 #10.
- **#9** — GitHub Actions CI fast gate (`.github/workflows/ci.yml`):
  `gofmt` drift check, `go vet ./...`, `go test -race`, binary build,
  and `go mod tidy` drift check on push/PR to `main`.
- **#9.1** — CI hardening: pinned `mobiai@^1.0.0` in integration workflow,
  added `paths-ignore` for docs-only changes, replaced swallowers with
  `set -euo pipefail` and `::error::` annotations.
- **#9.2** — Fixed YAML header corruption introduced during #9.1 polish.
- **#10** — Public rewrite of `README.md`: hero with badges, feature grid,
  3-step quickstart, install options, usage sections, config examples,
  CI snippet, comparison table, and honest scoping of `fix --mode auto`.
- **#13** — Makefile + Dockerfile for local and containerized builds.
  Makefile includes `build`, `test`, `test-race`, `lint`, `smoke`,
  `e2e-rick-morty`, `dashboard-build`, and `help` targets. Dockerfile is
  a multi-stage build shipping the kdoctor binary with a JRE and
  Detekt 1.23.8; `.dockerignore` keeps the build context small.
- **#12** — E2E test runner for a real Android project
  (`scripts/e2e-rick-morty`). Runs `kdoctor scan --json` against the
  configured `RICK_MORTY_APP` path, validates the schema version, score
  range, and the presence of `arch-god-class`. Skips cleanly when the
  project is not available.
- **#14** — `kdoctor init` project bootstrapper. Auto-detects project
  type (`android`, `kmp`, `cmp`, `compose`, `jvm`, `gradle`, `plain`)
  from filesystem heuristics, generates `kdoctor.config.yaml`,
  `detekt.yml`, and `.gitignore` entries, with `--force` and `--type`
  overrides. Covered by `internal/cli/init_test.go`.
- **#16** — MobiAI Graph integration. New `internal/mobiai` client posts
  findings to a configurable endpoint (`--mobiai-url` / `--mobiai-token`).
  Local `.mobiai/graph/findings.jsonl` output is preserved when no
  endpoint is configured. Covered by `internal/mobiai/client_test.go`.

### Stats

- 19 Go packages with tests.
- All tests PASS in the full suite (`go test -count=1 ./...`).
- Catalog: 78 rules total (`scripts/genschema/main.go`).
- Health Score formula: `100 - errors*5 - warnings*2 - info*0.5`.
- Project types supported: `kmp`, `cmp`, `jvm`, `android`, `compose`,
  `gradle`, `plain` (auto-detected or `--type=`).

### Validated

- `go vet ./...` exit 0
- `go test -count=1 ./...` all PASS
- `go build -o kdoctor.exe ./cmd/kdoctor` BUILD_OK
- End-to-end smoke test against `examples/bad-project` (Health Score: 82
  deterministic, 6 required findings matched).

### Notes

- Detekt SARIF 2.1.0 input validation in `sarif.Parse`.
- Detekt `--config` REPLACE-aware seeded config (`examples/bad-project/detekt.yml`).
- File path normalization handles Unix, Windows, mixed-slashes, and UNC
  via `internal/core/pathutil`.
- Tagged as `v1.0.0` (annotated).

---

## Future (post-v1.0.0)

See `docs/HANDOVER.md` §13 Round-3 release prep backlog (CI workflows,
README public rewrite, Makefile + Dockerfile, `kdoctor init` bootstrapper,
MobiAI Graph integration end-to-end).
