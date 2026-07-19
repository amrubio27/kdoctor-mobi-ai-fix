# `examples/scoring-fixtures/` — kdoctor smoke-test fixtures

Este directorio contiene los **JSON fixtures** que `scripts/evalprojects/main_test.go`
corre contra `kdoctor scan --json` para validar que el CLI detecta los hallazgos
esperados con el score correcto. Cada fixture = un proyecto bajo `examples/`
(`bad-project/`, `good-project/`) + un expectations.json que define:

- `projectPath`: ruta al proyecto fixture.
- `minScoreExpected` / `maxScoreExpected`: banda aceptable para el Health Score.
- `mustIncludeFinding`: lista de rule-IDs (kdoctor) que DEBEN aparecer en el
  reporte. El test falla si alguno falta (independiente del score).

## Fixtures y sus bandas

### `bad.json` — `[75, 90]` ⭐ (revisado round-2 polish)

Fixture de antipatrones (`examples/bad-project/`). El score actual con los
overrides inofensivos activos en `examples/bad-project/kdoctor.config.yaml`
es **82 determinísticamente** (errors=3, warnings=1, info=2, total=6).

**Decisión de banda [75, 90] — round-2 polish**:

| Opción | Banda | Trade-off |
|---|---|---|
| (a) restaurar `[80, 95]` (round-1 original) | más estricta | requeriría eliminar `security: warning` cluster-level del config → perder cobertura viva de Tier 3#7 |
| **(b) `[75, 90]` ← actual** | ±8 headroom | preserva cobertura de overrides; banda asimétrica (centro 82.5, 7 abajo / 8 arriba) |

Se eligió **(b)** porque el smoke test sirve como regression-guard vivo de la
feature Tier 3#7. Quitar el cluster-level override para restaurar (a) perdería
esa cobertura. La asimetría ↓7/↑8 se acepta porque el camino natural de drift
del score es hacia abajo (más findings, no menos).

#### Justificación del headroom ±8

Detekt 1.23.x standalone puede emitir pequeñas variaciones de stderr entre
corridas (errors de cache warm-up, orden de parser, etc.). ±8 puntos absorbe
esa varianza sin cazar falsos negativos en CI. Si se quiere ser más estricto,
bajar el lower bound a 78 o 80 acepta menor tolerancia — usar `--diff main`
en CI para reducir el scope efectivo.

#### `mustIncludeFinding` (6 IDs)

```
coroutine-global-scope          (detekt: GlobalCoroutineUsage, native)
dead-unused-import              (detekt: UnusedImports)
compose-missing-key             (native regex Tier 1#2)
sec-log-pii                     (native regex Tier 1#2)
sec-webview-javascript-enabled  (native regex Tier 1#2)
coroutine-dispatchers-hardcoded (native regex Tier 1#2)
```

**Por qué 6 y no más:** los 3 primeros vienen del detekt rules activos en
`examples/bad-project/detekt.yml`. Los 3 últimos vienen de los detectores regex
nativos de `internal/core/rules/rules.go`. Si añades una nueva regla nativa al
catálogo (`Status: live`), añade su caso a `BadCode.kt` y su ID aquí.

### `good.json` — `[95, 100]`

Fixture limpio (`examples/good-project/`). Sólo contiene un `Greeter.kt`
trivial sin antipatrones. El score actual es 100/100, banda `[95, 100]`
acepta drift menor (e.g. un info de `formatting-newline-at-eof`).

`mustIncludeFinding: []` significa que el test no exige presencia de ningún
rule-ID específico, sólo que el score caiga en banda.

## Schema invariants

Ambos fixtures respetan:

- `minScoreExpected ≤ maxScoreExpected` — un typo invirtiendo el orden
  resulta en banda vacía.
- `projectPath` apunta a un directorio existente bajo `examples/` con
  un `.kt` válido parseable por detekt.
- `mustIncludeFinding` (si no vacío) contiene rule-IDs que existen en
  `rules/metadata.json` con `Status: "live"` o `Status: "planned"`.

Estas invariantes NO se validan hoy via unit test. La validación es
integration via `evalprojects`, lo cual significa: si rompes una invariante,
el test falla con un mensaje críptico (e.g. "file not found" si el path
typo, o "no findings" si los IDs están stale). Considerar añadir un
`TestFixtureSchema` barato en `evalprojects/main_test.go` que valide las
invariantes antes de invocar kdoctor.

## Cómo añadir un nuevo fixture

1. Crear el proyecto bajo `examples/<nombre>/`.
2. Sembrar antipatrones (si fixture bad) o dejarlos limpios (si fixture good).
3. Si proyecto Android real: añadir `kdoctor.config.yaml` con la feature
   Tier 3#7 exercitada (cluster-level override + rule-level override +
   excludes pattern).
4. Definir el `*.json` en este directorio con `projectPath`, banda de score,
   y `mustIncludeFinding` según lo esperado.
5. Validar localmente:

```bash
./kdoctor.exe scan --json --type=android \
  --prefer-standalone --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=examples/<nombre>

go test -v -count=1 -run 'TestEval' ./scripts/evalprojects/...
```

## Historial de bandas

| Round | bad.json banda | good.json banda | Razón |
|---|---|---|---|
| Round-1 (original) | `[60, 85]` | `[95, 100]` | baseline sin overrides activos |
| Round-2 polish | `[75, 90]` ⭐ actual | `[95, 100]` | preservar cobertura Tier 3#7 cluster override |
