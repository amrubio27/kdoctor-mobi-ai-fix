# 🩺 kdoctor — Code Quality & Health Auditor for Android / KMP / Compose

> **Addon PoC para `mobiAi`**: Herramienta de auditoría estática de código de alto rendimiento que evalúa la salud de proyectos Kotlin/KMP/CMP (0-100 Health Score), detecta antipatrones de arquitectura, seguridad y Compose, y permite autocorregir hallazgos mediante IA.

---

## 🚀 Características Clave

- 📊 **Health Score (0-100)**: Algoritmo determinista de puntuación: `100 - (errors×5) - (warnings×2) - (info×0.5)`.
- 🔍 **78 Reglas de Calidad Catalogadas**:
  - 11 Reglas V1 Live (seguridad, corrutinas, fugas de memoria, arquitectura God Class).
  - 14 Mapeos directos de Detekt SARIF 2.1.0.
  - Reglas nativas en Go (sin JVM overhead para escaneo regex de PII, WebViews inseguros, Dispatchers hardcoded y llaves de Compose).
- 🛠️ **Integración MCP Natively Built-in (`kdoctor-mcp`)**: Servidor JSON-RPC 2.0 sobre `stdio` para consumo directo por Cursor, Claude Code y agencias MobiAI.
- 🤖 **AI Quality Fixer (`kdoctor fix --ai`)**: Construye prompts enriquecidos con contexto relativo de código (±10 líneas) e incluye **Patchguard** (lexer Kotlin que previene syntax breaks y realiza rollback automático).
- 📑 **Reportes Multi-formato**: Consola TUI interactiva, Markdown (`kdoctor scan --md`), JSON Schema v3 (`--json`) y SARIF 2.1.0 (`--sarif`) para GitHub Code Scanning.
- 🌐 **Integración MobiAI Graph**: Exportación directa de telemetría de hallazgos vía API REST o feed local `.mobiai/graph/findings.jsonl`.
- ⚙️ **Configuración Flexible (`kdoctor.config.yaml`)**: Anulación de severidades por regla o por cluster, exclusiones personalizadas y baselines XML (`--baseline`).

---

## 💻 Requisitos Previos

- **Go 1.22+** (para compilación).
- **JDK 17+** (necesario para ejecutar el motor `detekt` subyacente).
- Binario `detekt-cli` (opcionalmente gestionado o especificado mediante `--detekt-bin`).

---

## 📦 Instalación

### Opción A: Compilación desde código fuente
```bash
git clone https://github.com/amrubio27/kdoctor-mobi-ai-fix.git
cd kdoctor-mobi-ai-fix
make build
# Genera kdoctor.exe y kdoctor-mcp.exe
```

### Opción B: Docker
```bash
docker build -t kdoctor .
docker run --rm -v $(pwd):/workspace kdoctor scan --project-dir=/workspace
```

---

## 🏁 Guía de Uso Rápido (Quickstart)

### 1. Verificar Entorno
```bash
./kdoctor doctor
```

### 2. Inicializar Proyecto
Genera archivos de configuración adaptados al stack (`android`, `kmp`, `cmp`, `compose`):
```bash
./kdoctor init --type=kmp --with-skills
```

### 3. Escanear un Proyecto
```bash
./kdoctor scan --project-dir=/ruta/a/tu/proyecto
```

### 4. Generar Reporte Markdown o Resumen Ejecutivo
```bash
# Reporte completo en Markdown
./kdoctor scan --md --project-dir=/ruta/a/tu/proyecto

# Resumen ejecutivo en consola (Score + Top Clusters)
./kdoctor scan --summary --project-dir=/ruta/a/tu/proyecto
```

### 5. Reparación Asistida por IA
```bash
# Sugerir arreglos en fixes.md
./kdoctor fix --ai

# Aplicar arreglos automáticamente con validación Patchguard
./kdoctor fix --ai --mode=auto
```

---

## 🔌 Configuración para IDEs y Agentes IA (MCP Server)

Para integrar `kdoctor` en Cursor, Claude Code o MobiAI CLI, añade la siguiente entrada a la configuración MCP (`mcpServers`):

```json
{
  "mcpServers": {
    "kdoctor": {
      "command": "C:/ruta/a/kdoctor-mcp.exe",
      "env": {
        "KDOCTOR_BIN": "C:/ruta/a/kdoctor.exe"
      }
    }
  }
}
```

### Herramientas MCP Disponibles

| Tool MCP | Descripción |
|---|---|
| `kdoctor_scan` | Ejecuta análisis estático y devuelve score + hallazgos JSON |
| `kdoctor_rules` | Lista el catálogo de reglas y sus taxonomías |
| `kdoctor_init` | Configura archivos `kdoctor.config.yaml` y `detekt.yml` |
| `kdoctor_doctor` | Valida dependencias del sistema |
| `kdoctor_fix_suggest` | Genera prompt de reparación con contexto de código |

---

## ⚙️ Configuración del Proyecto (`kdoctor.config.yaml`)

Ejemplo de personalización de reglas por equipo:

```yaml
project:
  type: kmp
  excludes:
    - "**/build/**"
    - "**/.gradle/**"

overrides:
  # Desactivar cluster completo
  formatting: OFF
  
  # Cambiar severidad de cluster
  security: warning
  
  # Override por regla específica (precede sobre cluster)
  sec-log-pii: error
```

---

## 🔗 Integración CI/CD y MobiAI Graph

### En GitHub Actions
```yaml
- name: Run kdoctor Scan
  run: ./kdoctor scan --sarif --out=results.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

### Conexión con MobiAI Graph Backend
```bash
./kdoctor scan --mobiai-url="https://api.mobiai.dev" --mobiai-token="$MOBIAI_TOKEN"
```

---

## 📄 Licencia
Este proyecto está bajo la Licencia MIT.
