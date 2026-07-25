# 🩺 kdoctor — Code Quality & Health Auditor for Android / KMP / Compose

> **Addon PoC para `mobiAi`**: Herramienta de auditoría estática de código de alto rendimiento que evalúa la salud de proyectos Kotlin/KMP/CMP (0-100 Health Score), detecta antipatrones de arquitectura, seguridad y Compose, y permite autocorregir hallazgos mediante IA.

---

## 🚀 Características Clave

- 📊 **Health Score (0-100) Proporcional por KLOC**: Algoritmo ajustado por densidad de defectos por miles de líneas de código Kotlin (KLOC), garantizando evaluaciones justas tanto en micro-proyectos como en grandes monorepositorios.
- ⚡ **Auto-configuración en Memoria**: Exención automática para funciones `@Composable` en proyectos recién clonados sin requerir configuración manual previa de `detekt.yml`.
- 🔍 **88 Reglas de Calidad Catalogadas**:
  - Reglas avanzadas de Jetpack Compose (recomposiciones, claves en listas, DerivedState, Compose Rules).
  - Reglas de Corrutinas & Flow (manejadores de excepciones, testeabilidad de Dispatchers, operadores de Flow).
  - Reglas de Clean Architecture (encapsulación de ViewModels, métodos de UseCases) y KMP.
- 📦 **Instalación en 1 Línea (Zero Build)**: Scripts de instalación directa para Windows, macOS y Linux sin necesidad de compilar.
- 🔄 **Reglas Modulares Offline-First**: Catálogo embebido en el ejecutable, con caché local persistente (`~/.kdoctor/rules/metadata.json`) y actualización remota con un clic (`kdoctor rules update`).
- 🌐 **Reporte Web HTML Interactivo (`--html`)**: Genera un dashboard web autocontenido (`kdoctor-report.html`) en Dark Mode con medidor de score, filtros instantáneos por severidad/cluster y sugerencias de remediación (`FixHint`).
- 📑 **Reportes Multi-formato**: Consola TUI interactiva con avisos explicativos, Markdown (`--md`), HTML interactivo (`--html`), JSON Schema v3 (`--json`) y SARIF 2.1.0 (`--sarif`).
- 🛠️ **Integración MCP Natively Built-in (`kdoctor-mcp`)**: Servidor JSON-RPC 2.0 sobre `stdio` para consumo directo por Cursor, Claude Code y agencias MobiAI.
- 🤖 **AI Quality Fixer (`kdoctor fix --ai`)**: Construye prompts enriquecidos con contexto relativo de código (±10 líneas) e incluye **Patchguard** (lexer Kotlin que previene syntax breaks y realiza rollback automático).

---

## 💻 Instalación Rápida (One-Liner Installers)

Instala `kdoctor` directamente en tu sistema sin necesidad de instalar Go ni compilar:

### 🔹 Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.ps1 | iex
```

### 🔹 macOS / Linux (Bash / Zsh)
```bash
curl -fsSL https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.sh | sh
```

---

### Opción Alternativa: Compilación desde código fuente
```bash
git clone https://github.com/amrubio27/kdoctor-mobi-ai-fix.git
cd kdoctor-mobi-ai-fix
go build -o kdoctor.exe ./cmd/kdoctor
```

---

## 🏁 Guía de Uso Rápido (Quickstart)

### 1. Inicializar Proyecto
Genera la configuración y exenciones adaptadas al stack (`android`, `kmp`, `cmp`, `compose`):
```bash
kdoctor init --type=cmp
```

### 2. Escanear un Proyecto (Consola)
Si estás dentro del directorio del proyecto, simplemente ejecuta:
```bash
kdoctor scan
```
*(Opcionalmente puedes pasar la ruta como argumento: `kdoctor scan /ruta/a/tu/proyecto` o usar `--project-dir=/ruta`).*

### 3. Generar Reporte Web HTML Interactivo
```bash
kdoctor scan --html
# Genera y abre kdoctor-report.html en el directorio actual
```

### 4. Generar Reporte Markdown o JSON
```bash
# Reporte completo en Markdown
kdoctor scan --md

# Reporte JSON estructurado para agentes de IA
kdoctor scan --json
```

### 5. Inspeccionar o Actualizar Catálogo de Reglas
```bash
# Listar las 88 reglas y su fuente de carga
kdoctor rules list

# Sincronizar las últimas reglas lanzadas en GitHub
kdoctor rules update
```

### 6. Reparación Asistida por IA
```bash
# Sugerir arreglos en fixes.md
kdoctor fix --ai

# Aplicar arreglos automáticamente con validación Patchguard
kdoctor fix --ai --mode=auto
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

---

## ⚙️ Configuración del Proyecto (`kdoctor.config.yaml`)

Ejemplo de personalización de reglas por equipo:

```yaml
project:
  type: cmp
  excludes:
    - "**/build/**"
    - "**/.gradle/**"

overrides:
  # Desactivar cluster completo
  formatting: OFF
  
  # Cambiar severidad de cluster
  security: warning
  
  # Override por regla específica
  coroutine-dispatchers-hardcoded: info
```

---

## 🔗 Integración CI/CD y GitHub Actions

```yaml
- name: Run kdoctor Scan
  run: kdoctor scan --sarif --out=results.sarif

- name: Upload SARIF to GitHub Code Scanning
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

---

## 📄 Licencia
Este proyecto está bajo la Licencia MIT.
