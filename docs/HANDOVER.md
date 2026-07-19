# kdoctor — Handover document (continuidad entre sesiones IA)

> **Objetivo**: que cualquier IA (o persona) que llegue a este repo pueda leer este único documento y saber **exactamente** qué es kdoctor, qué está hecho, qué está pendiente, cómo validarlo, y cómo continuar.

> **Estado al cierre**: 11 commits en `main`, Phase 1 + Phase 1.5 cerradas, Tier 1 Item 1 en curso.

---

## 1. TL;DR (1 párrafo)

**kdoctor** es un CLI en Go para auditar calidad de proyectos Android / KMP / CMP. Inspirado en `react-doctor`. Reutiliza el binario `detekt` como motor de análisis estático, parsea su salida SARIF 2.1, mapea reglas Detekt a reglas kdoctor (78 reglas: 11 V1 live + 53 V1 planned + 14 default-detekt mappings Phase 1.5), calcula un **Health Score 0-100** y emite reportes en consola/JSON/SARIF. Naming final fijado en `kdoctor` (anteriormente `adkd`).

## 2. Repo state (11 commits, 57 archivos)

```
ee191e0 feat(rules): Phase 1.5 — 78 reglas catalog + FixHint + consistency tests
217adc7 feat(scan): end-to-end validation against RickMortyApp — Fase 1 closed
b20899f chore(license): rename adkd to kdoctor in copyright holder
9d3c1ee refactor(rebrand): rename adkd → kdoctor
b119be4 refactor(detektrunner): limpiar vars zombies y consolidar test helpers
642b337 feat(detektrunner): validar formato SARIF 2.1.0 antes de aceptar .sarif
140b8f8 fix: validate Fase 1 canvas with real Go 1.26.5
7389bf7 feat: scaffold adkd Fase 1 PoC Inspector
```

- **Branch**: `main`, working tree clean.
- **Working directory local**: `C:\Users\Miguel\Desktop\doctor mobi ai fix`.
- **Project local Maven module path** (importable): `github.com/adkd/adkd`. **Pendiente GitHub handle del usuario — reversible.**

## 3. Tech stack

- **Go 1.22+** (testado con `go1.26.5` en `C:\Program Files\Go\bin\go.exe`)
- **`github.com/spf13/cobra`** para la CLI
- **`detekt-cli` 1.23.8** como motor de análisis (en `D:\tools\detekt-cli.jar` + shim `D:\tools\detekt.cmd`)
- **JDK 25.0.3 (Temurin)** en `C:\Program Files\Eclipse Adoptium\jdk-25.0.3.9-hotspot\bin` (NO en PATH de usuario; lo exporto manualmente en cada shell)
- **Sin Node, sin cGo, sin dependencias nativas**. Compilación reproducible desde cero con `go mod tidy && go build -o kdoctor.exe ./cmd/kdoctor`.

## 4. Top-level layout

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
│   │   └── scan.go                     # comando principal (validate contre RickMortyApp)
│   ├── core/
│   │   ├── types/                      # Finding, Rule, HealthScore, Summary, Report
│   │   ├── sarif/                      # parser SARIF 2.1.0 estricto
│   │   ├── detektrunner/               # subprocess manager (standalone + gradlew modes)
│   │   ├── rulemap/                    # Detekt ID → kdoctor rule
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
│       └── main.go (CLI)               # Reads examples/scoring-fixtures/*.json, runs kdoctor scan, validates
├── examples/
│   ├── bad-project/                    # SEEDED antipatterns for smoke test (Phase B in flight)
│   ├── good-project/                   # SEEDED clean code
│   └── scoring-fixtures/
│       ├── bad.json                    # expects 40-80 score, must include coroutine-global-scope + sec-hardcoded-secret + dead-unused-import
│       └── good.json                   # expects 95-100 score, no required findings
├── docs/
│   ├── HANDOVER.md                     # ← YOU ARE HERE
│   ├── superpowers/plans/
│   │   └── 2026-07-19-kdoctor-implementation-plan.md   # MASTER plan (Phase 1-5 + Tier 1/2/3 split)
│   ├── v2/
│   │   └── kdoctor-proposal-v2.md      # extracted del PDF MobiAi Cli PoC Addon.pdf
│   └── website/                        # landing page existing
└── .gitignore
```

## 5. Architecture (Phase 1+1.5)

```
┌─────────────────┐     spawn subprocess     ┌─────────────────┐
│ kdoctor scan   │ ──────────────────────── │ detekt-cli      │
│ ─────────────   │                          │ ──────────────── │
│ 1. resolveRules │   reads rules/metadata.json via scripts/genschema
│ 2. Detect mode  ├─ standalone (default Fase 1.5) OR ./gradlew (scaffold)
│ 3. RunDetekt    │ writes SARIF to os.TempDir()/kdoctor-detekt.sarif
│    ├─ RunStandalone: java -jar detekt-cli.jar --input --report sarif: ...
│    │  - --max-issues 99999 (forces exit 0 even with findings)
│    │  - --excludes '**/build/**,**/.gradle/**,**/kspCaches/**'
│    │  - safety net: if cmd.Run fails but SARIF OK, accept it
│    └─ RunGradlew: ./gradlew detekt --init-script <WriteInitScript output>
│ 4. sarif.Parse(file)  → []types.Finding
│ 5. rulemap.BuildIndex(rules)
│ 6. rulemap.Map(findings) → enriches with kdoctor IDs/clusters/severities
│ 7. grader.Score(findings) → int 0-100 + Summary
│ 8. Reporter.RenderReport(...) → console / JSON / SARIF
└─────────────────┘
```

**Adapted from V2 proposal**: bypassing Node.js, using Go + detekt SARIF, single-pass quality-focused prompts for future fix phase.

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
```

SchemaVersion=3 alinea con react-doctor para que tools downstream lo entiendan.

## 7. Validación rápida (copiar-pegar)

```bash
# Setup PATH (en CADA nueva terminal; el JDK no está en PATH permanente)
export PATH="/c/Program Files/Go/bin:/c/Program Files/Eclipse Adoptium/jdk-25.0.3.9-hotspot/bin:/d/tools:$PATH"
cd "C:\\Users\\Miguel\\Desktop\\doctor mobi ai fix"

# 1. Validar Go vet + tests + build
go vet ./...                                  # exit 0
go test ./...                                 # ~35 tests, exit 0 (cached)
go build -o kdoctor.exe ./cmd/kdoctor         # exit 0, binary ~7MB

# 2. Validar que el catálogo no está drift
go run ./scripts/genschema -out rules/metadata.json
# → stderr: "Generated 78 rules to rules/metadata.json"

# 3. Smoke test contra RickMortyApp (proyecto real del usuario)
./kdoctor.exe scan --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp
# → exit 0, Health Score 0/100, 103 findings clustered across naming/formatting/complexity/etc.

# 4. sub-comandos CLI
./kdoctor.exe --version           # "kdoctor version 0.1.0"
./kdoctor.exe --help              # usage completo
./kdoctor.exe rules               # 64-78 reglas listadas
./kdoctor.exe doctor              # checks de deps (Go, Java, Detekt, etc.)
./kdoctor.exe scan --help         # help completo del scan
```

## 8. Tier roadmap (status)

| Tier | Item | Estado | Próximo paso |
|---|---|---|---|
| **Tier 1 #1** | examples/bad-project + smoke test in `go test` | 🚧 En curso | Populate `examples/bad-project/src/main/kotlin/bad/*.kt` |
| **Tier 1 #2** | Go-based regex detectors en `internal/core/rules/` | ⏳ Pendiente | Implementar compose-missing-key, sec-log-pii, compose-remember-missing como Go regex |
| **Tier 1 #3** | `--diff main` / baseline.xml | ⏳ Pendiente | git merge-base + diff SARIF output |
| **Tier 2 #4** | `kdoctor-gradle-plugin` | ⏳ Pendiente | Subproyecto Gradle con task `kdoctorScan` no-invasivo |
| **Tier 2 #5** | `kdoctor fix --ai` Quality-Focused | ⏳ Pendiente | LLM CLI adapter (Claude Code primero) + dry-run obligatorio |
| **Tier 3 #6** | HTML dashboard | ⏳ Pendiente | vite + tailwind sobre schematic 3 JSON |
| **Tier 3 #7** | `kdoctor.config.yaml` overrides por equipo | ⏳ Pendiente | Cluster-aware severity + custom excludes |

**Orden de ejecución fijado por el usuario**: abrir Tier 1 #1 primero, cerrarlo, luego Tier 1 #2, etc, hasta Tier 3 #7.

## 9. Gotchas críticos (leer antes de tocar nada)

1. **NO commitear nada en `D:\Programacion\RickMortyApp`** — el usuario lo dijo explícitamente. Es su proyecto de prueba; cualquier cambio ahí debe revertirse o consultarse primero. kdoctor es READ-ONLY sobre el proyecto del usuario.

2. **`go.mod` tiene `module github.com/adkd/adkd`** — placeholder. Cuando el usuario defina su handle de GitHub, hay que cambiar esa línea + `go mod tidy`. Reversible sin breakage (los import paths internos son relativos al module path).

3. **Detekt setup del usuario está en `D:\tools\`** — `detekt-cli.jar` + `detekt.cmd` shim. Fuera de PATH de usuario (probablemente). kdoctor acepta `--detekt-bin=<ruta_explícita>` para no depender del PATH. Usar ese flag en lugar de setup global de PATH, siempre que sea posible.

4. **JDK NO está en PATH permanente**. Hay que hacer `export PATH=...:$PATH` cada terminal. La ruta canónica es `C:\Program Files\Eclipse Adoptium\jdk-25.0.3.9-hotspot\bin\java.exe`.

5. **CRLF vs LF en Windows**. Git auto-convierte CRLF↔LF pero genera warnings. Después de CUALQUIER edit a `.go`, correr `gofmt -w <archivo>` para normalizar y evitar drift en `git diff`.

6. **`os.TempDir()` en Windows = `C:\Users\<user>\AppData\Local\Temp\`**. SARIF lives ahí. Sarifs stale pueden MASKAR errores reales. **Safety net actual**: cada run hace `os.Remove(sarifPath)` ANTES de invocar detekt → solo acepta SARIF fresco.

7. **regexp en scan es el `filepath.WalkDir`** en `internal/core/detektrunner/find.go`. Skip dirs: `.git`, `node_modules`, `.gradle`, `.idea`, `build`, `kspCaches`. El `--excludes` de detekt hace el mismo trabajo pero más rápido.

8. **V2 del plan explica la diferencia con V1** (`docs/v2/kdoctor-proposal-v2.md`): Fase 5 propone `FirAdditionalCheckersExtension` para K2 compiler checkers nativos. Esa rama queda en R&D; no se implementa hasta Fase 5 futura.

9. **Drift guardrail**: `scripts/genschema/main_test.go::TestCatalogConvergence` falla si `rules/metadata.json` no coincide con el slice `CatalogRules` en código. **IMPORTANTE**: editar SIEMPRE primero `scripts/genschema/main.go` (no el JSON). Después correr `go run ./scripts/genschema` para regenerar JSON y commitear ambos.

10. **`TestClusterTaxonomyIsConsistent` y `TestValidPrefixesCoverage`** son meta-tests que prohíben el bug "prefijo que no es del cluster" (`style-*` cuando el cluster es `complexity`). Si añades nueva regla a `CatalogRules`, asegúrate de añadir su cluster a `validPrefixes` map en el test.

## 10. On-ramp para la siguiente sesión IA

Si acabas de aterrizar en este repo sin contexto:

```
1. Lee ESTE archivo entero (docs/HANDOVER.md).
2. Lee docs/superpowers/plans/2026-07-19-kdoctor-implementation-plan.md (master plan, ~70KB).
3. Lee docs/v2/kdoctor-proposal-v2.md (contexto estratégico extraído del PDF).
4. Mira el log: git log --oneline -15
5. Mira el catálogo: cat rules/metadata.json | jq 'length'  → 78
6. Valida baseline (5 minutos):
   export PATH="/c/Program Files/Go/bin:/c/Program Files/Eclipse Adoptium/jdk-25.0.3.9-hotspot/bin:/d/tools:$PATH"
   cd "C:\\Users\\Miguel\\Desktop\\doctor mobi ai fix"
   go vet ./... && go test ./... && go build -o kdoctor.exe ./cmd/kdoctor
7. Identifica qué tier está abierta. Sección 8 arriba lista el estado.
8. Confirma con el usuario qué item abrir antes de empezar trabajo sustantivo.
```

## 11. Helpers de invocación frecuente

```bash
# Smoke test contra RickMortyApp (proyecto real del usuario)
./kdoctor.exe scan --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp

# Scan en JSON schema v3 (para CI o MobiAI Graph)
./kdoctor.exe scan --json --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp

# Scan en SARIF 2.1.0 (para GitHub Code Scanning)
./kdoctor.exe scan --sarif --type=kmp --prefer-standalone \
  --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=D:/Programacion/RickMortyApp \
  --out=detekt.sarif

# Regen del catálogo (tras editar scripts/genschema/main.go)
go run ./scripts/genschema -out rules/metadata.json

# Formatear todo
gofmt -w .

# Doctor
./kdoctor.exe doctor

# Rules (listado)
./kdoctor.exe rules | tail -3
```

## 12. Convenciones de código

- **Naming IDs**: V1 usa prefijo = namespace (`compose-*`, `coroutine-*`, `arch-*`, `sec-*`, `mem-*`, `dead-*`, `a11y-*`, `test-*`, `kmp-*`, `lifecycle-*`). Phase 1.5+ (5.11) usa cluster como prefijo (`complexity-*`, `error-handling-*`, `magic-numbers-*`, `naming-*`, `formatting-*`). Tests garantizan esta convención.
- **Severity**: `error | warning | info`.
- **FixHint**: string accionable en imperativo ("Replace with..."),omitempty si vacío.
- **Commit messages**: conventional commits (`feat(scope): summary`, `fix(scope): summary`, `refactor(scope): summary`).
- **Spanish comments / English code** (mixed). Pragmatic. kdoctor code identifiers en inglés.

---

**Última actualización**: cierre de Phase 1.5 (commit `ee191e0`), antes de empezar Tier 1 items.

**Próxima sesión**: continuar desde Tier 1 #1 (examples/bad-project populated + evalprojects smoke test).
