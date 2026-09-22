# kdoctor + MobiAI

kdoctor nació como complemento de [MobiAI](https://github.com/ArisGuimera/MobiAI-Core). Este documento separa **lo que funciona hoy** de **lo que es una propuesta** todavía no acordada, porque mezclarlo hace perder el tiempo a quien lo lea.

---

## Estado: qué funciona hoy

### El encaje conceptual

MobiAI distribuye *skills de guía*: le dicen al agente **cómo** trabajar en un proyecto móvil. kdoctor aporta *medición*: un Health Score reproducible y findings con fichero, línea y columna. Son complementarios — MobiAI acota el problema, kdoctor lo cuantifica.

### El flujo que funciona

```bash
# 1. MobiAI Graph acota el radio de impacto
mobiai graph context "audit module app"

# 2. kdoctor mide, sólo sobre lo que cambió
kdoctor scan --diff main --json

# 3. El agente lee findings[] y arregla
# 4. Re-scan y mostrar el delta de score
```

Los tres comandos existen y se pueden encadenar hoy.

### Volcado al grafo de MobiAI

```bash
kdoctor scan --mobiai
```

Escribe `.mobiai/graph/findings.jsonl` junto al `index.json` de MobiAI Graph, una línea por finding con el `types.Finding` completo (cluster, severidad, `fixHint` incluidos). Combina con cualquier formato de salida:

```bash
kdoctor scan --mobiai --json
```

> Hasta la ronda de arreglos de 2026-09, esta combinación concreta **no escribía nada**: el `switch` de formato de salida hacía `return` antes de llegar al bloque de volcado, así que sólo funcionaba en modo consola. Si estás leyendo una versión anterior del binario, actualízalo.

Con `--mobiai-url` (o `KDOCTOR_MOBIAI_URL`) además hace `POST` de los findings aplanados a un endpoint `/graph/findings`. Ese endpoint es **parte de la propuesta**, no algo que MobiAI sirva hoy.

### Instalar la skill sin MobiAI

`SKILL.md` en la raíz del repo sigue el estándar agentskills.io, así que es autocontenido. Se puede copiar a mano al host que sea:

```bash
# Claude Code
mkdir -p ~/.claude/skills/kdoctor && cp SKILL.md ~/.claude/skills/kdoctor/

# Cursor, Gemini CLI, Copilot: misma idea bajo ~/.cursor, ~/.gemini, ~/.copilot
```

La skill hace *pre-flight* del binario `kdoctor` y, si no está, le da al usuario el one-liner de instalación sin ejecutarlo. Así funciona igual con MobiAI o sin él; MobiAI sólo aporta la distribución multi-host.

---

## Propuesta: qué haría falta acordar

Nada de esta sección existe todavía. Está aquí como petición concreta a MobiAI, no como documentación.

### 1. kdoctor en el catálogo de MobiAI

Hoy MobiAI instala **packs** desde un catálogo embebido en su binario:

```bash
mobiai skills add android      # packs: android, core, flutter, ios, kmp, mobile, react-native
mobiai skills list
```

Un pack despliega carpetas `<nombre>/SKILL.md` en los hosts detectados (`~/.claude/skills/`, `~/.cursor/`, `~/.gemini/`, `~/.copilot/`) y registra el estado en `~/.mobiai/state/installed.json`.

Para que kdoctor se distribuya así haría falta que su `SKILL.md` entre en el catálogo, probablemente como `mobiai-kdoctor` siguiendo la convención de nombres.

**Detalle a resolver:** MobiAI distribuye markdown, no binarios. Ninguna skill del catálogo instala un ejecutable. kdoctor necesita su binario, así que o bien la skill se limita a instruir al usuario (lo que hace hoy en su pre-flight), o bien MobiAI incorpora alguna noción de dependencia de herramienta.

### 2. `mobiai doctor --code`

Un envoltorio que ejecute `kdoctor scan --mobiai --json`, evalúe el score y ofrezca remediación. Requiere acuerdo sobre el contrato de salida.

### 3. Endpoint `/graph/findings`

kdoctor ya sabe hacer `POST` de sus findings (`--mobiai-url`, `--mobiai-token`). Falta el lado servidor.

---

## Nota histórica

Versiones anteriores de este documento describían `mobiai skills install kdoctor` y `mobiai doctor --code` como si ya existieran. Ninguno de los dos existe: el comando real es `mobiai skills add <pack>`, y `mobiai doctor` son diagnósticos del propio CLI, sus hosts y su catálogo. `plugin.json` describe además un formato de plugin con `commands[]` y `hooks[]` que MobiAI no consume.
