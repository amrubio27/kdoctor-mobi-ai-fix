# 🩺 kdoctor — Code Quality & Health Auditor for Android / KMP / Compose

> **Addon PoC para `mobiAi`**: Herramienta de auditoría estática de código de alto rendimiento que evalúa la salud de proyectos Kotlin/KMP/CMP (0-100 Health Score), detecta antipatrones de arquitectura, seguridad y Compose, y permite autocorregir hallazgos mediante IA.

---

## 🚀 Características Clave

- 📊 **Health Score Continuo (0-100) y Calibrado por KLOC**: Algoritmo continuo sin saltos discontinuos, ponderado por categorías de cluster (Seguridad x2.0, Arquitectura x1.5, Corrutinas x1.25, UI x0.75). Mantiene los fallos críticos protegidos de la dilución por KLOC y aplica rendimientos decrecientes a avisos repetidos con tope de penalización a hallazgos `Info`.
- ⚡ **Auto-configuración en Memoria**: Exención automática para funciones `@Composable` en proyectos recién clonados sin requerir configuración manual previa de `detekt.yml`.
- 🔍 **116 Reglas Catalogadas, 64 activas** (20 nativas en Go + 44 vía detekt). Ejecuta `kdoctor doctor` para ver cuántas puede evaluar tu entorno concreto:
  - Reglas de Clean Architecture y SOLID (Separación Data/Domain/Presentation, contratos ViewModel/UseCase, prevención de fuga de lógica, mappers `DataModel` → `DomainModel` → `UiModel`, patrones MVI y UDF).
  - Reglas avanzadas de Jetpack Compose (prevención de recomposiciones, claves en listas `key`, optimización `graphicsLayer`, modularización de Composables grandes).
  - Reglas de Corrutinas & Flow (manejadores de excepciones, inyección de `Dispatchers`, operadores de Flow).
  - Testeabilidad e Inyección de Dependencias (detección de instanciaciones directas).
- 📦 **Instalación en 1 Línea (Zero Build)**: Scripts de instalación directa para Windows, macOS y Linux sin necesidad de compilar.
- 🔄 **Reglas Modulares Offline-First**: Catálogo embebido en el ejecutable, con caché local persistente (`~/.kdoctor/rules/metadata.json`) y actualización remota con un clic (`kdoctor rules update`).
- 🌐 **Reporte Web HTML Interactivo (`--html`)**: Genera un informe autocontenido y offline (`kdoctor-report.html`) en modo oscuro, con el Health Score, filtros combinables por severidad y por cluster, y sugerencias de remediación (`FixHint`).
- 📑 **Reportes Multi-formato**: Consola con color, Markdown (`--md`), HTML interactivo (`--html`), JSON Schema v3 (`--json`) y SARIF 2.1.0 (`--sarif`).
- 🛠️ **Integración MCP Natively Built-in (`kdoctor-mcp`)**: Servidor JSON-RPC 2.0 sobre `stdio` para consumo directo por Cursor, Claude Code y agencias MobiAI.
- 🤖 **Plan de Remediación para Agentes (`kdoctor fix`)**: Emite cada hallazgo con su ventana de código (±10 líneas), el `fixHint` de la regla y el rango exacto de líneas a reemplazar, en JSON o Markdown. kdoctor **no llama a ningún modelo y nunca edita tus ficheros**: el agente que lo invoca ya tiene uno. `kdoctor fix --validate` pasa **Patchguard** (lexer Kotlin) sobre lo que el agente escribió.

---

## 📋 Requisitos

Solo **un JDK 11-21**. kdoctor descarga detekt y el plugin de reglas Compose
automáticamente en el primer escaneo (a `~/.kdoctor/tools/`, verificando el
checksum) y los reutiliza después. Si no encuentra un JDK compatible, el scan
sigue funcionando con las 20 reglas nativas y te dice qué falta.

`kdoctor doctor` te dice exactamente en qué estado está tu entorno.

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
# Listar el catálogo completo y su fuente de carga
kdoctor rules list

# Sincronizar las últimas reglas lanzadas en GitHub
kdoctor rules update
```

### 6. Plan de Remediación
```bash
# Plan legible en fixes.md (dentro del proyecto)
kdoctor fix

# Plan en JSON, para que lo consuma tu agente o el servidor MCP
kdoctor fix --json

# Tras aplicar los cambios, comprobar que el Kotlin sigue balanceado
kdoctor fix --validate src/main/kotlin/MiFichero.kt
```

> `--ai` y `--mode` se siguen aceptando con un aviso de deprecación, para no romper scripts existentes.

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
projectType: cmp

excludes:
  - "**/build/**"
  - "**/.gradle/**"

# Severidad por cluster o por regla concreta: error | warning | info | off
rules:
  formatting: off
  security: warning
  coroutine-dispatchers-hardcoded: info

# Quality gate: si el Health Score baja de aquí, exit code != 0.
# Un `--fail-below` explícito en la línea de comandos tiene prioridad.
score:
  failBelow: 80
```

Genera el fichero con `kdoctor init`. Los campos son los de
[`kdoctor.config.example.yaml`](kdoctor.config.example.yaml).

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
