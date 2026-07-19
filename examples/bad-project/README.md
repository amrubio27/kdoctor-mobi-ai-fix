# `examples/bad-project/` — kdoctor smoke-test fixture

Este directorio es el **fixture canónico de antipatrones** que el smoke test
(`scripts/evalprojects/main_test.go::TestEvalBadProjectFixture`) usa para
verificar que kdoctor detecta los hallazgos requeridos.

## Qué vive aquí

| Archivo | Rol |
|---|---|
| `src/main/kotlin/bad/BadCode.kt` | Fuente con 9 antipatrones sembrados a propósito (no los corrijas — son los datos de prueba del smoke test). |
| `detekt.yml` | Config mínima para detekt 1.23.x. Activa *solo* las 2 reglas que el smoke test verifica (`GlobalCoroutineUsage` y `UnusedImports`). El resto de reglas default de detekt están *desactivadas* por el gotcha `--config REPLACE`. |
| `kdoctor.config.yaml` | Demo vivo de la feature **Tier 3#7** (overrides por equipo). Ver `## Overrides activos` abajo. |
| `report.json` | Snapshot regenerado con `kdoctor scan --json --out examples/bad-project/report.json`. Sirve como referencia del output esperado (schema v3). |

## Cómo reproducir el smoke test manualmente

```bash
# (1) Build kdoctor.exe (desde la raíz del repo)
go build -o kdoctor.exe ./cmd/kdoctor

# (2) Correr scan contra el fixture y emitir a report.json
./kdoctor.exe scan --json --type=android \
  --prefer-standalone --detekt-bin=D:/tools/detekt.cmd \
  --project-dir=examples/bad-project \
  --out=examples/bad-project/report.json

# (3) Verificar findings esperados
grep -o '"id": "[^"]*"' examples/bad-project/report.json | sort -u
# Debe listar los 6 IDs:
#   compose-missing-key, coroutine-dispatchers-hardcoded, coroutine-global-scope,
#   dead-unused-import, sec-log-pii, sec-webview-javascript-enabled

# (4) Idem via test estándar
go test -v -count=1 -run 'TestEval' ./scripts/evalprojects/...
```

## Output esperado (con `examples/bad-project/kdoctor.config.yaml` aplicado)

| Métrica | Valor | Notas |
|---|---|---|
| Schema version | `3` | — |
| Health Score | `82` | Varía si alguien edita los overrides. |
| Findings totales | `6` | Constante (override warning no dropa). |
| Errors | `3` | `coroutine-global-scope`, `compose-missing-key`, `sec-log-pii`. |
| Warnings | `1` | `sec-webview-javascript-enabled` (downcast cluster). |
| Info | `2` | `dead-unused-import`, `coroutine-dispatchers-hardcoded`. |

### Por qué el score es 82 y no 79 ni 85

Cálculo V1 (Health Score = 100 − err×5 − warn×2 − info×0.5):

```
100 - (3 errores × 5) - (1 warning × 2) - (2 infos × 0.5)
= 100 - 15 - 2 - 1
= 82
```

El score NO es 79 (baseline sin overrides) porque hay un cluster-level override
activo. NO es 85 porque **rule-level override gana sobre cluster-level override**
(`ApplyOverrides` mira primero `overrides[f.ID]` antes que `overrides[f.Cluster]`).
En este fixture:

- `sec-log-pii` tiene **ambas** overrides en juego:
  - Cluster-level: `security: warning` (downcast a warning).
  - Rule-level: `sec-log-pii: error` (mantiene error).
  - **Gana rule-level** → queda en error → aporta 5 puntos al score.
- `sec-webview-javascript-enabled` solo tiene cluster-level:
  - Cluster-level: `security: warning` → baja a warning → aporta 2 puntos.

## Overrides activos en `kdoctor.config.yaml`

Demo vivo de los tres code paths de `ApplyOverrides`:

```yaml
excludes:
  - "**/Legacy.kt"     # Path no-existente → regression-guard de la rama excludes.
                        # (Rama 1 del mapping.go ApplyOverrides)

rules:
  security: warning     # Cluster-level DOWNCAST. Aplica a TODA regla con
                        # cluster=security. NO dropa, sólo baja de error→warning.
                        # (Rama 2 del mapping.go ApplyOverrides)
                        # Impacto: sec-webview-javascript-enabled (única sin rule-override).

  sec-log-pii: error    # Rule-level OVERRIDE. Gana sobre cluster-level
                        # (regla precedence: rule antes que cluster).
                        # Resultado: sec-log-pii mantiene "error" pese al
                        # cluster-level `security: warning`. Demuestra
                        # explícitamente la precedencia.
                        # (Rama 3 del mapping.go ApplyOverrides)
```

### Por qué `compose-performance` NO está override aquí

`compose-missing-key` (cluster `compose-performance`) es required en
`examples/scoring-fixtures/bad.json::mustIncludeFinding`. Si alguien añadiera
`compose-performance: off` a este config, el smoke test rompe inmediatamente
porque `ApplyOverrides` filtra el finding. Esto es deliberado: la cobertura
de cluster-level override la ejerce `security: warning` (downcast innocuous)
en lugar de silenciar `compose-performance`.

## ⚠️ Warnings esperados en stderr

Detekt 1.23.x en modo standalone (sin classpath / type-resolution) emite
**warnings de UnresolvedReference** a stderr para los bare references de
`BadCode.kt`:

- `configureWebView(settings: Any)` — usa tipo built-in `Any` para evitar
  definir un stub falso. Detekt acepta el parse y SARIF se genera OK.
- `Dispatchers.IO`, `items(...)`, `Log.d(...)` — referencias sin declarar
  para que se disparen los detectores regex nativos de
  `internal/core/rules/rules.go` sin necesidad de stubs.

**Esto es por diseño**: los detectores nativos son regex puramente textuales
sobre el source. No requieren resolución de tipos. NO indican bugs en
kdoctor — solo son warnings cosméticos que puedes ignorar.

## Forward-compat note

Si en el futuro migras a **detekt 2.x con `--classpath` o `--all-rules`**,
detekt activará resolución de tipos. En ese escenario:

1. `Any.javaScriptEnabled = true` será un error de tipo fatal (no warning).
2. Los bare references a `Dispatchers`, `Log`, `items` también fallarán.

Soluciones forward-compat:

- **Opción A**: crear `examples/bad-project/src/main/kotlin/bad/Stubs.kt`
  con las clases dummy de tipo proper (`class WebSettings { val javaScriptEnabled = ... }`),
  aisladas claramente como scaffolding de tests. El smoke test sigue igual.
- **Opción B**: usar solo imports reales Android SDK (`import android.util.Log`,
  `import kotlinx.coroutines.Dispatchers`, etc.) — necesita que detekt
  tenga el classpath del Android SDK disponible.

Para detekt 1.23.x (target actual del proyecto), las referencias intencionalmente
unresolved de `BadCode.kt` siguen siendo el approach más limpio.

## Convenciones del fixture

- **No "arregles" los antipatrones** — son los datos del test.
- Si añades nueva regla al catálogo `scripts/genschema/main.go` y la
  activas como nativa (`Status: live`), añade su caso a `BadCode.kt`
  usando el patrón `<regex>` esperado. Ver `internal/core/rules/rules.go`
  para los patterns.
- Si modificas `BadCode.kt`, valida que sigue produciendo los 6 findings
  esperados vía `kdoctor scan --json`.
