# adkd · Android / KMP / CMP Doctor

CLI estilo `react-doctor` para proyectos Android, Kotlin Multiplatform y Compose Multiplatform. Asigna un **Health Score 0–100** sobre un catálogo de reglas de calidad y permite aplicar auto-fix con IA (Claude Code, Cursor, Gemini, MobiAI).

> Estado actual: **Fase 1 — PoC Inspector**. Ver [`docs/superpowers/plans/2026-07-19-adkd-implementation-plan.md`](docs/superpowers/plans/2026-07-19-adkd-implementation-plan.md) para el plan completo y [`docs/v2/adkd-proposal-v2.md`](docs/v2/adkd-proposal-v2.md) para el spec V2 revisado.

## Quickstart

```bash
# 1. Requisitos: Go 1.22+, Detekt CLI (o gradlew con el plugin), un LLM CLI (opcional en Fase 1)
go install github.com/adkd/adkd/cmd/adkd@latest  # o `go build -o adkd ./cmd/adkd`

# 2. Inicializa en un proyecto Android/KMP
cd tu-proyecto
adkd init   # crea adkd.config.yaml

# 3. Escanea
adkd scan                    # salida rich console + Health Score
adkd scan --json             # salida JSON schema v3 (CI)
adkd scan --sarif            # SARIF 2.1 (GitHub Code Scanning)
adkd scan --diff main        # solo findings nuevos vs main
```

## Comandos principales

| Comando | Qué hace |
|---|---|
| `adkd scan` | Escanea el proyecto, calcula Health Score |
| `adkd scan --json` | Igual, output JSON schema v3 |
| `adkd scan --sarif` | Output SARIF 2.1.0 |
| `adkd scan --diff <ref>` | Solo findings introducidos desde `<ref>` (git) |
| `adkd fix --ai` | Aplica fixes con LLM (Fase 3) |
| `adkd init` | Crea `adkd.config.yaml` |
| `adkd rules` | Lista todas las reglas del catálogo |
| `adkd hook install` | Instala pre-push hook con quality gate |

## Inspirado por

- [`react-doctor`](https://github.com/millionco/react-doctor) — el modelo a seguir.
- [`MobiAI`](https://mobiai.dev/) — aliado de distribución (Aris Guimera).
- [`Detekt`](https://detekt.dev/), [Android Lint](https://developer.android.com/studio/write/lint), [KtLint](https://ktlint.github.io/).

## Licencia

MIT — ver [`LICENSE`](LICENSE).
