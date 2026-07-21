---
name: kdoctor
description: Ejecuta `kdoctor scan` en proyectos Android/KMP/CMP para auditar la salud del código. Encuentra antipatrones en Compose, coroutines, lifecycle y arquitectura; produce un Health Score 0–100 y findings accionables. Usar cuando el usuario pida auditar, revisar o aplicar fixes automáticos a código Android/Kotlin. Cuando trabajes en el propio repositorio de kdoctor, lee primero `HONEY.md`.
when_to_use: Cuando hay un proyecto Android/KMP/CMP y el usuario quiere diagnóstico, score, o auto-fix con IA. Cuando un agente vaya a tocar el código de kdoctor, también debe leer `HONEY.md`.
---

# kdoctor — Android/KMP/CMP Doctor

## Cuándo invocar

- "audita mi proyecto Android", "pásale el doctor", "¿qué tal está mi Health Score?"
- "encuentra antipatrones de Compose en mi app"
- "arregla los issues automáticamente con IA"
- "conecta kdoctor como MCP" o "usa kdoctor desde Cursor/Claude/Cline"

## Comandos principales

### 1. `kdoctor scan`
Ejecuta en la raíz del proyecto. Devuelve:
- Health Score (0–100).
- Lista de findings agrupados por cluster (compose-performance, coroutines, lifecycle, architecture, accessibility, testing, security, kmp, dead-code).
- Modo `--json` para CI; modo `--sarif` para GitHub Code Scanning.
- Modo `--diff main` para auditar solo cambios nuevos respecto a una rama.

### 2. `kdoctor fix --ai`
Tres modos:
- `--mode suggest` (default): genera `fixes.md`, NO toca código.
- `--mode interactive`: pregunta por cada fix.
- `--mode auto`: aplica, valida con patch guard, deja `git diff` listo.

El LLM provider se detecta automáticamente (`provider: auto`). MobiAI, Claude Code, Cursor, Gemini CLI, Codex funcionan out-of-the-box.

### 3. `kdoctor-mcp`
Servidor MCP stdio que expone herramientas a agentes de IA. Herramientas:
- `kdoctor_scan` — escanear un proyecto y devolver reporte JSON/SARIF.
- `kdoctor_rules` — listar el catálogo de reglas.
- `kdoctor_init` — generar `kdoctor.config.yaml` y `detekt.yml`.
- `kdoctor_doctor` — diagnosticar dependencias del entorno.
- `kdoctor_fix_suggest` — generar sugerencias de fix sin aplicarlas.

Arrancar:
```bash
go run ./cmd/kdoctor-mcp
```

Configurar en Cursor / Claude / Cline con `KDOCTOR_BIN` apuntando al binario.

### 4. `mobiai doctor --code`
Una vez instalado `mobiai skills install kdoctor`, este subcomando corre `kdoctor scan` y, si el usuario acepta, lanza `--fix --ai`.

## Flujo típico (ejemplo end-to-end)

1. Usuario: "audita el módulo app del proyecto".
2. Invocar `mobiai graph context "android audit module app"` primero — MobiAI Graph da la lista de archivos relevantes.
3. Sobre esos archivos: `kdoctor scan --diff main --json`.
4. Parsear JSON: extraer `findings[]` ordenados por severidad.
5. Si `healthScore` baja: proponer al usuario `kdoctor fix --ai --mode interactive`.
6. Tras cada fix: re-scan, mostrar delta de score.

## Guardrails Honey

Antes de tocar el código del propio repositorio `kdoctor`, leer `HONEY.md`. Reglas clave:
- Planificar antes de codear.
- Revisar el propio cambio antes de declararlo listo.
- No eliminar tests sin reemplazarlos.
- No cambiar APIs públicas sin actualizar todos los callers.
- Validar con `go vet ./...`, `go test ./...`, `go build ./...` y `gofmt -l .`.
