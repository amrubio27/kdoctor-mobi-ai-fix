# 🩺 android-kmp-doctor-ai-fix

> Un CLI estilo `react-doctor` para Android, Kotlin Multiplatform y Compose Multiplatform, con auto-fix potenciado por IA e integrado como skill de **MobiAI**.

**TL;DR** — Escaneas tu código, te da un **Health Score 0-100**, encuentra antipatrones en Compose, coroutines, lifecycle, accesibilidad, arquitectura… y un agente IA (Claude Code / Cursor / Gemini / Copilot) **lo arregla automáticamente** desde tu terminal, dejando un `git diff` listo para revisar.

```
$ adkd scan
⚠ Health Score: 47/100
$ adkd fix --ai
✓ 14 issues fixed in 7 files
$ adkd scan
✅ Health Score: 92/100
```

---

## 📑 Tabla de contenidos

1. [Motivación](#1-motivación-qué-falta-en-el-ecosistema-androidkmp)
2. [Inspiración: qué hace `react-doctor`](#2-inspiración-qué-hace-react-doctor)
3. [El aliado: MobiAI de AristiDevs](#3-el-aliado-mobiai-de-aristidevs)
4. [Propuesta de valor del PoC](#4-propuesta-de-valor-del-poc)
5. [Las 60+ reglas agrupadas](#5-las-60-reglas-agrupadas)
6. [Arquitectura técnica](#6-arquitectura-técnica)
7. [El flujo end-to-end con IA](#7-el-flujo-end-to-end-con-ia)
8. [Health Score: cómo se calcula](#8-health-score-cómo-se-calcula)
9. [Modos del AI Fixer](#9-modos-del-ai-fixer)
10. [Integración profunda con MobiAI](#10-integración-profunda-con-mobiai)
11. [Roadmap por fases](#11-roadmap-por-fases)
12. [Quickstart para developers](#12-quickstart-para-developers)
13. [Cómo contribuir reglas](#13-cómo-contribuir-reglas)
14. [Referencias y links](#14-referencias-y-links)

---

## 1. Motivación: qué falta en el ecosistema Android/KMP

Hoy por hoy, un dev Android tiene a su disposición:

| Herramienta | Qué hace | Le falta |
|---|---|---|
| **`./gradlew lint`** | Reglas de Android Lint | No da score, no agrupa, no arregla con IA |
| **Detekt** | Static analysis en Kotlin | Reporta pero no prioriza, no tiene bridge con IA |
| **KtLint** | Style guide | Solo estilo, no semántica |
| **Spotless** | Format check | Solo formato |
| **LeakCanary** | Memory leaks en runtime | No es estático, solo debug builds |
| **Baseline Profile** | Performance | Requiere Macrobenchmark, no analiza código |
| **Now in Android (sample)** | Buenas prácticas | Es un sample, no es herramienta |

**Nadie une todo bajo un mismo Health Score que la IA pueda leer y arreglar.**

Concretamente:

- ❌ No existe un **"doctor" para Android** al estilo `react-doctor` que haga AST analysis + dead-code detection + reporte con score 0-100.
- ❌ No existe una **integración limpia** entre esos lints y un LLM que aplique los fixes sin tocar a mano.
- ❌ No hay un **sistema extensible de rule-packs** que la comunidad pueda compartir (como las reglas de ESLint).
- ❌ No hay un **pre-commit / pre-push hook** que diga "tu código va a bajar el proyecto de 92 a 64, ¿lo bloqueamos?".

Eso es exactamente lo que vamos a construir.

---

## 2. Inspiración: qué hace `react-doctor`

[`react-doctor`](https://github.com/millionco/react-doctor) es un CLI open-source publicado por Million, licencia **MIT**, distribuido como `npx react-doctor@latest`.

### 2.1 Qué detecta (60+ reglas)

- **Hooks mal usados** — `useEffect` con deps incorrectas, missing deps, deps innecesarias.
- **Renderizados innecesarios** — componentes sin `memo`, props nuevas en cada render, context overkill.
- **Arquitectura** — prop drilling, componentes gigantes (>250 líneas), barrel exports caóticos.
- **Performance** — re-renders en cascada, key instable, listas sin keys.
- **Accesibilidad (a11y)** — `alt` faltante, `aria-*`, contraste mediante DOM.
- **Seguridad** — `dangerouslySetInnerHTML`, secrets hardcoded.
- **Dead code** — archivos, exports, tipos que nadie importa.

### 2.2 Cómo funciona técnicamente

- **Análisis estático puro** sobre el **AST** del código fuente. No usa LLM para escanear (sería lento y caro), solo para arreglar.
- Dos pasadas paralelas:
  1. **Lint analysis** — reglas detector de patrones en AST.
  2. **Dead-code detection** — grafo de dependencias para encontrar orphan exports y archivos no usados.
- **Motor propio** en TypeScript sobre Node.js. NO es un wrapper de ESLint ni de Babel: trae su propio walker de AST.

### 2.3 Configurable y AI-friendly

- Archivo `doctor.config.ts` en la raíz para activar/desactivar reglas.
- Salida estructurada (SARIF-like + JSON) lista para que Claude Code, Cursor, Windsurf la lean y arreglen.

### 2.4 El diferenciador: Health Score

Genera un **Health Score 0-100**:

```
Health Score: 47 / 100  ⚠️
✗ 23 issues broken down across 12 files.
✗ 8 dead exports, 3 dead files, 2 dead types.
✗ 2 critical perf issues (re-renders, missing memo).
```

Bloquea deploys basados en score mínimo. Es el "linter conversacional" que faltaba en React.

---

## 3. El aliado: MobiAI de AristiDevs

[`MobiAI`](https://github.com/ArisGuimera/MobiAI-Core) y [mobiai.dev](https://mobiai.dev/) (de **Aris Guimera** / aristidevs) es **la mejor distribución práctica** que existe hoy para integrar IA en desarrollo mobile.

### 3.1 Qué es realmente

**MobiAI no es un LLM por sí mismo.** Es un **orquestador meta-CLI** que mejora asistentes IA existentes:

- Claude Code
- Cursor
- GitHub Copilot CLI
- Gemini CLI

Inyectándoles **contexto mobile** (skills específicas de Android, KMP, Compose, Gradle, testing, etc.).

### 3.2 Distribución

Binario standalone vía script:

```bash
curl -fsSL https://mobiai.dev/install.sh | sh
```

**Adecuado para que `adkd` (android doctor) sea un binario standalone y se enchufe como skill.**

### 3.3 Qué tiene hoy y qué le falta

| Función | MobiAI hoy | `adkd` aporta |
|---|---|---|
| Diagnóstico de entorno (IA hosts, skills, perms) | ✅ `mobiai doctor` | — |
| **Análisis estático del código fuente** | ⚠️ Solo índice semántico (`Graph`) | ✅ Motor de reglas AST profundo |
| **Health Score 0-100** | ❌ | ✅ |
| **60+ reglas de calidad (perf, a11y, antipatrones)** | ❌ | ✅ |
| **Dead code detection** | ❌ | ✅ |
| Reparto de findings a LLM para fix | ✅ (es su core) | ✅ (como deliverable) |
| Skills instalables | ✅ tiene `install` | ✅ adkd se instala como skill |

**MobiAI ya tiene la *infraestructura de entrega*** (instalar skills, invocar IA para arreglar), pero **no tiene un motor de reglas estáticas** comparable a las 60 reglas de react-doctor.

**Juntando ambos** → la pareja perfecta:

- `adkd` = el **detective** (escanea, encuentra, reporta con score).
- MobiAI skills + CLI = el **mecánico** (lee el reporte, llama al LLM adecuado, aplica fixes, revisa).

---

## 4. Propuesta de valor del PoC

### 4.1 Tagline

> **"Sube tu Health Score de Android/KMP sin tocar el código. La IA lo arregla por ti."**

### 4.2 Usuario objetivo

- Dev Android / KMP / CMP que ya usa un IDE con IA (Claude Code, Cursor).
- Equipos que quieren imponer **quality gates** medibles en CI.
- Refactor de legacy Spaghetti Compose UI.
- Estudiantes y mentores (AristiDevs tiene una comunidad enorme) que necesitan una herramienta "enseñable" para interiorizar buenas prácticas.

### 4.3 Qué hace en una sola línea

```
AST → sub-analyzers → rule engine → score + findings →
→ [MobiAI/Claude Code/Cursor/Gemini CLI] →
→ git diff con fixes aplicados → humanos revisan
```

### 4.4 Comandos principales del PoC

Todas vivien bajo un binario `adkd` (android doctor). Pensado para ser `npx`, `brew`, o standalone.

| Comando | Qué hace |
|---|---|
| `adkd scan` | Escanea todo el repo, emite Health Score + findings list |
| `adkd scan --json` | Igual pero en JSON para CI |
| `adkd scan --sarif` | SARIF 2.1 para GitHub Code Scanning |
| `adkd scan --rule <id>` | Solo una regla (debug) |
| `adkd fix --ai` | Lanza el fixer IA con el último scan |
| `adkd fix --ai --dry-run` | Solo muestra el diff propuesto, no toca nada |
| `adkd fix --ai --interactive` | Pregunta antes de cada fix |
| `adkd explain <finding-id>` | Por qué importa esa regla, ejemplos, links |
| `adkd rules` | Lista todas las reglas disponibles y su status |
| `adkd init` | Crea `adkd.config.ts` con defaults sensatos |
| `adkd hook install` | Instala un pre-commit / pre-push hook |
| `adkd doctor` | Diagnóstico del propio `adkd` (env, versiones, IA hosts) |

---

## 5. Las 60+ reglas agrupadas

Agrupadas por categoría. **Total objetivo para la v1**: 60 reglas (mismo número que react-doctor). Para el **PoC MVP** (semana 1-2), target 12 reglas críticas.

### 5.1 🧠 Compose Performance (12 reglas)

| ID | Regla | Severity |
|---|---|---|
| `compose-missing-key` | Falta `key` en `items()` con listas grandes | error |
| `compose-unstable-params` | Lambda/Collection como param sin `remember`, causa recomposición | error |
| `compose-derived-state-missing` | `derivedStateOf` debería usarse para state derivado | warning |
| `compose-lambda-recomposition` | Inline lambdas no estables en items (perf hit) | warning |
| `compose-heavy-composable` | Composable >250 líneas, dividir | info |
| `compose-remember-missing` | Estado olvidado sin `remember` | error |
| `compose-state-hoisting` | State debería hoisting fuera del composable | warning |
| `compose-modifier-frequent-changes` | Modifier asignado dentro de recomposición | warning |
| `compose-graphics-layer` | `Modifier.graphicsLayer` debería `remember { ... }` | warning |
| `compose-list-animated` | `LazyColumn` con >100 items sin `key` ni `contentType` | warning |
| `compose-side-effect-in-compose` | `LaunchedEffect` con keys vacíos en cuerpo recomponible | error |
| `compose-runtime-import-bleeding` | Compose Runtime importado fuera de `@Composable` | error |

### 5.2 🔄 Coroutines & Async (8 reglas)

| ID | Regla | Severity |
|---|---|---|
| `coroutine-viewmodel-scope` | `viewModelScope` usado fuera de ViewModel | error |
| `coroutine-global-scope` | Uso de `GlobalScope` en producción | error |
| `coroutine-dispatchers-hardcoded` | `Dispatchers.IO`/`Main` hardcoded, debería inyectarse | info |
| `coroutine-supervisor-missing` | Job hijo sin `SupervisorJob` en scope crítico | warning |
| `coroutine-unstructured-concurrency` | `launch` no estructurado en `suspend fun` | warning |
| `coroutine-cancellation-leak` | `runCatching {}` swallow `CancellationException` | error |
| `coroutine-flow-buffer-missing` | `flow.collect` sin buffer en cadenas largas | warning |
| `coroutine-sharedflow-replay` | `MutableSharedFlow` con replay en cache crítico | info |

### 5.3 🔁 Lifecycle & Activity leaks (6 reglas)

| ID | Regla | Severity |
|---|---|---|
| `lifecycle-context-leak` | ViewModel/Composable guardando referencia a Activity/Context | error |
| `lifecycle-collect-as-state-missing` | `Flow.collectAsState()` sin `Lifecycle.repeatOnLifecycle` | error |
| `lifecycle-collect-lifecycle-aware` | `LaunchedEffect` sin `LifecycleState` apropiado | warning |
| `lifecycle-ondestroy-listener` | Listener no removido en `onDestroy`/`DisposableEffect` | warning |
| `lifecycle-job-not-cancelled` | `Job` no cancelado en `onCleared` | error |
| `lifecycle-config-change-survival` | Estado no persistente en `onSaveInstanceState` | info |

### 5.4 🧩 Memory & Resources (5 reglas)

| ID | Regla | Severity |
|---|---|---|
| `mem-bitmap-no-pool` | Bitmap grande sin `BitmapFactory.Options.inBitmap` | warning |
| `mem-context-receiver-leak` | Context en clase con lifecycle mayor al context | error |
| `mem-static-context` | Static field contendo Context | error |
| `mem-handler-leak` | `Handler` con lambda no débil | warning |
| `mem-coroutine-job-leak` | Job no cancelado, mantiene ViewModel activo | error |

### 5.5 🏗️ Architecture & Clean code (10 reglas)

| ID | Regla | Severity |
|---|---|---|
| `arch-god-class` | Clase >500 líneas o >20 métodos | warning |
| `arch-circular-dep` | Dependencia circular entre paquetes | error |
| `arch-feature-module-public-api-bleed` | API interna de feature expuesta hacia fuera | warning |
| `arch-public-api-mutable-state` | `var` en API pública | error |
| `arch-data-class-with-logic` | `data class` con métodos no triviales | warning |
| `arch-named-arg-required` | Función pública con muchos params sin named args | info |
| `arch-utility-function-should-be-extension` | Function top-level debería ser extension | info |
| `arch-internal-in-public-api` | Tipo `internal` usado en API `public` | error |
| `arch-package-cycles-kmp` | Ciclos entre módulos KMP (`commonMain` ↔ `androidMain`) | error |
| `arch-presentation-depends-on-data` | Capa `presentation` importando `Retrofit` directamente | error |

### 5.6 ♿ Accessibility (a11y) (5 reglas)

| ID | Regla | Severity |
|---|---|---|
| `a11y-content-description` | `Icon`/`Image` sin `contentDescription` | error |
| `a11y-click-target-size` | Elemento clickable <48dp | warning |
| `a11y-merged-clickable` | `Modifier.clickable` sobre `Modifier.semantics` merge | info |
| `a11y-talkback-label-missing` | Componente interactivo sin `onClickLabel` | warning |
| `a11y-color-contrast-note` | Color similar a background en producción | info |

### 5.7 🧪 Testing (5 reglas)

| ID | Regla | Severity |
|---|---|---|
| `test-public-api-without-test` | Función pública sin test correspondiente | warning |
| `test-flaky-test-marker` | `@FlakyTest` ignorado en CI | info |
| `test-hilt-rule-missing` | `@HiltAndroidTest` sin `HiltAndroidRule` | error |
| `test-runblocking-in-test` | `runBlocking` en test unit, debería `runTest` | warning |
| `test-compose-test-rule-missing` | Test de Composable sin `createComposeRule()` | warning |

### 5.8 🔒 Security (5 reglas)

| ID | Regla | Severity |
|---|---|---|
| `sec-hardcoded-secret` | API key, token o URL sospechosa en código | error |
| `sec-log-pii` | Log de email, phone, password | error |
| `sec-webview-javascript-enabled` | `webView.settings.javaScriptEnabled = true` sin sanitizar | error |
| `sec-deeplink-no-validation` | Deeplink sin validación de origen/host | warning |
| `sec-fragment-injection` | `Fragment` cargado desde intent sin sanitizar | error |

### 5.9 🌐 KMP / CMP specific (4 reglas)

| ID | Regla | Severity |
|---|---|---|
| `kmp-platform-api-leak` | `androidx.*` o `UIKit.*` importado en `commonMain` | error |
| `kmp-expect-actual-violation` | `expect fun` sin `actual fun` definido | error |
| `kmp-coroutines-supervisor-in-common` | `CoroutineExceptionHandler` faltante en `commonMain` | warning |
| `kmp-compose-multiplatform-stable-required` | Compose Multiplatform: surface debe ser `@Stable` | warning |

### 5.10 ⚰️ Dead code (4 reglas)

| ID | Regla | Severity |
|---|---|---|
| `dead-unused-import` | Import nunca usado | info |
| `dead-unused-private-fun` | Función `private` sin uso | info |
| `dead-unused-parameter` | Parámetro ignorado en override (no `_` prefix) | warning |
| `dead-white-label-export` | Top-level export sin import en el proyecto | info |

Total: **12 + 8 + 6 + 5 + 10 + 5 + 5 + 5 + 4 + 4 = 64 reglas**. (Más que las 60 de react-doctor. 😉)

---

## 6. Arquitectura técnica

### 6.1 Vista general

```
┌─────────────────────────────────────────────────────────────┐
│                          adkd CLI                            │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ Detekt   │  │Android   │  │ KtLint   │  │ KSP/Custom│     │
│  │  bridge  │  │  Lint    │  │ bridge   │  │ rules     │     │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
│       └─────────────┴────────────┴─────────────┘             │
│                       │  SARIF / JSON                         │
│                       ▼                                       │
│            ┌─────────────────────────┐                       │
│            │   Rule Engine (core)    │                       │
│            │   - AST merge           │                       │
│            │   - dedup findings      │                       │
│            │   - weight by severity  │                       │
│            │   - calculate score     │                       │
│            └────────────┬────────────┘                       │
│                         ▼                                     │
│              ┌──────────────────────┐                        │
│              │   Reporter Layer     │                        │
│              │   - Console (rich)   │                        │
│              │   - JSON             │                        │
│              │   - SARIF 2.1        │                        │
│              │   - HTML dashboard   │                        │
│              └─────────┬────────────┘                        │
│                        ▼                                     │
│         ┌──────────────────────────────┐                     │
│         │   AI Fixer Bridge            │                     │
│         │   - CLI resolver (MobiAI /  │                     │
│         │     Claude / Cursor / etc.) │                     │
│         │   - Prompt builder          │                     │
│         │   - Diff applier            │                     │
│         │   - Dry-run / interactive   │                     │
│         └──────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 Stack propuesto

| Capa | Tecnología | Por qué |
|---|---|---|
| **CLI core** | **Node.js + TypeScript** | Ecosistema maduro, mismo lenguaje que react-doctor, fácil publicar en npm |
| **AST Kotlin parser** | [`kotlinc`](https://github.com/JetBrains/kotlin) embedded / [`kotlin-parser`](https://www.npmjs.com/package/kotlin-parser) fork tipo [jetbrains/kotlin](https://github.com/JetBrains/kotlin) | Parsear `.kt` files a AST sin JVM |
| **Analyzers externos** | Detekt (XML/JSON output), Android Lint (XML), KtLint (JSON) | Reusar en vez de reinventar |
| **Custom rules (KSP-friendly)** | Kotlin Symbol Processing API | Para escribir reglas con type info |
| **Linter UI (rich console)** | [`@clack/prompts`](https://github.com/natemoo-re/clack) + [`chalk`](https://github.com/chalk/chalk) | Terminales bonitas |
| **TUI interactiva** | [`ink`](https://github.com/vadimdemedes/ink) (React para terminal) | Para `adkd doctor` interactivo |
| **AI Bridge** | Spawn de `claude`, `cursor-agent`, `gemini` CLI + prompt file | Agnóstico al LLM |
| **Reporte HTML** | Plantilla Vite + static export | Sin servidor, GitHub Pages friendly |
| **Testing del propio CLI** | Vitest + fixtures de proyectos Android reales | Cobertura E2E |

### 6.3 Estructura del monorepo

```
android-kmp-doctor-ai-fix/
├── packages/
│   ├── core/                # Rule engine, AST merge, score
│   ├── adapter-detekt/      # Convierte output de Detekt → findings
│   ├── adapter-androidlint/ # Convierte output de Android Lint → findings
│   ├── adapter-ktlint/      # Convierte output de KtLint → findings
│   ├── rules-compose/       # Reglas custom Compose
│   ├── rules-coroutines/    # Reglas custom coroutines
│   ├── rules-arch/          # Reglas arquitectura
│   ├── rules-a11y/          # Reglas accesibilidad
│   ├── rules-kmp/           # Reglas KMP-specific
│   ├── reporters/
│   │   ├── console/         # Rich console
│   │   ├── json/
│   │   ├── sarif/
│   │   └── html/            # Static dashboard
│   └── ai-fixer/            # Bridge con LLMs
├── apps/
│   └── cli/                 # Binario `adkd`
├── examples/
│   ├── bad-project/         # Proyecto Android deliberadamente roto
│   └── good-project/        # Mismo proyecto arreglado por la IA
└── docs/
    ├── website/             # Landing page (artefacto de diseño)
    └── rules/               # Spec de cada regla
```

### 6.4 Configuración del usuario — `adkd.config.ts`

```ts
import { defineConfig } from 'adkd'

export default defineConfig({
  projectType: 'android', // 'kmp' | 'cmp'
  paths: {
    kotlin: ['app/src/main/**/*.kt'],
    compose: ['**/*Composable*.kt'],
  },
  rules: {
    // Activar todas las del namespace compose-performance
    'compose-performance/*': 'error',

    // Desactivar una específica
    'arch-package-cycles-kmp': 'off',

    // Custom override
    'compose-heavy-composable': ['warn', { maxLines: 200 }],
  },
  score: {
    failBelow: 80, // bloquea CI por debajo de 80
  },
  aiFixer: {
    provider: 'auto', // 'mobiai' | 'claude' | 'cursor' | 'gemini'
    mode: 'interactive', // 'auto' | 'interactive' | 'pr'
  },
})
```

---

## 7. El flujo end-to-end con IA

Este es el "wow moment" del PoC. El flujo exacto que verás en la demo de la landing:

```
1. adkd scan
   ↓
2. Encuentra 18 issues repartidos en 7 archivos
   ↓
3. Reporta score 47/100 con severity breakdown
   ↓
4. adkd fix --ai
   ↓
5. adkd genera un prompt estructurado con los findings
   ↓
6. adkd lanza el provider IA elegido:
   - MobIAI skill: invoca el agente configurado
   - Claude Code: lo abre con el prompt cargado
   - Cursor: detecta contexto y aplica
   ↓
7. La IA lee cada finding, propone fix, lo aplica en el código
   ↓
8. adkd recibe los nuevos archivos, calcula diff
   ↓
9. adkd scan (segunda vuelta)
   ↓
10. Score 92/100
   ↓
11. git diff con los fixes, humano revisa y commitea
```

### 7.1 El prompt que `adkd` le pasa a la IA

```text
Eres un senior Android/Kotlin/Compose engineer.
Tu única tarea: aplicar los fixes descritos abajo, sin modificar nada más.

PROYECTO: com.amrubio27.cursotestingandroid
ARCHIVOS A TOCAR: 7
FINDINGS:

[FINDING-001] compose/remember-missing | ERROR
  Archivo: app/src/main/.../CartViewModel.kt:42
  Regla:    "Composable state should be remembered"
  Fix sugerido: Envolver `var expanded = ...` con `var expanded by remember { mutableStateOf(false) }`
  Import a añadir: androidx.compose.runtime.*

[FINDING-002] coroutine/unstructured-concurrency | WARNING
  Archivo: app/src/main/.../ProductDetailViewModel.kt:88
  Regla:    "launch dentro de suspend fun sin scope"
  ...

  ...

INSTRUCCIONES:
1. Aplica solo los fixes descritos.
2. Mantén imports ordenados.
3. No re-formatees código no relacionado.
4. Devuelve un resumen final con los archivos tocados y por qué.
```

### 7.2 Por qué dejamos que la IA modifique y no `adkd` directamente

- Las **reglas difíciles** (a11y, architecture, naming) requieren **juicio**, no patrón.
- La IA ya sabe Kotlin/Compose. `adkd` no necesita re-implementar el conocimiento.
- Si la IA falla, `adkd` puede re-lanzar con retries. Si `adkd` fallara, tendría que re-ejecutar todo el pipeline.
- **Cero acoplamiento**: si mañana sale un LLM nuevo, solo cambia el bridge.

---

## 8. Health Score: cómo se calcula

Fórmula inicial v1:

```
score = 100
  - (errors * 5)
  - (warnings * 2)
  - (info * 0.5)
  - (dead_files * 3)
  - (unused_exports * 1)
  - (a11y_error * 4)         // a11y pesa más
  - (a11y_warning * 2)

floor(score, 0)
```

Con cap mínimo en 0 y máximo en 100.

### 8.1 Categorías del score

| Rango | Color | Categoría |
|---|---|---|
| 90-100 | 🟢 verde | Excelente |
| 75-89 | 🔵 cyan | Bueno |
| 50-74 | 🟡 amarillo | Necesita atención |
| 25-49 | 🟠 naranja | Crítico |
| 0-24 | 🔴 rojo | Crashed |

### 8.2 Por qué este score, no "failing tests"

- Es **forward-looking**: detecta deuda técnica antes de que rompa.
- Es **comparable entre proyectos**: dos proyectos se pueden rankear.
- Es **gamificable**: subir de 47 a 92 en un commit da feedback inmediato.
- Es **bloqueable en CI**: `adkd scan --fail-below 80` es un quality gate.

---

## 9. Modos del AI Fixer

Tres modos, de menos a más invasivo:

### 9.1 `suggest` (default seguro)

`adkd fix --ai` → la IA **solo sugiere**. Genera un `fixes.md` con diffs propuestos + explicación. **No toca nada.**

### 9.2 `apply --interactive`

`adkd fix --ai --interactive` → por cada fix:

```
[FIX 1/12] compose/remember-missing in CartViewModel.kt:42
  Aplicar? [Y/n/edit/skip]
```

Si el dev dice `edit`, puede ajustar antes de aplicar.

### 9.3 `apply --auto` (tipo "Arréglalo todo")

`adkd fix --ai --apply` → la IA aplica todo, genera `git diff` y se detiene. El dev revisa el diff y commitea o descarta.

### 9.4 Modo bonus: `pr`

`adkd fix --ai --pr` → crea branch `adkd/fix-<timestamp>`, aplica, abre PR en remoto con la lista de cambios como descripción.

---

## 10. Integración profunda con MobiAI

### 10.1 Tres niveles de integración

**Nivel 1: skill instalable**

`adkd` se distribuye como una MobiAI-skill instalable vía:

```bash
mobiai skills install adkd
```

Esta instalación:

1. Descarga el binario `adkd`.
2. Registra el comando `mobiai doctor --code` como wrapper que invoca `adkd scan`.
3. Añade prompts pre-armados para que el agente MobiAI ya sepa cómo interpretar el output.

**Nivel 2: provider de findings para MobiAI**

MobiAI tiene `Graph` (índice semántico de archivos). `adkd` le pasa a `Graph` los findings como anotaciones, para que cuando un agente MobiAI trabaje sobre un archivo, **ya vea los problemas pendientes** en su contexto.

**Nivel 3: comando `mobiai doctor --code`**

`mobiai doctor` hoy solo chequea entorno. El nuevo sub-comando:

```bash
mobiai doctor --code
```

Corre `adkd scan`, muestra score, y si MobiAI tiene IA configurada, ofrece `Run fixes? [Y/n]`. Si dice que sí, invoca el `ai-fixer` de adkd con el provider MobiAI.

### 10.2 Beneficio mutuo

| Para MobiAI | Para adkd |
|---|---|
| Gana un "motor de reglas" que no tenía | Gana una distribución masiva (comunidad hispana) |
| Su `doctor` pasa de entorno a código | Su IA ya está pre-configurada |
| Su `Graph` se enriquece con findings | Su fixer ya está "conectado" al LLM elegido |

---

## 11. Roadmap por fases

### 🟢 Fase 0 — Definición (Sprint actual)

- ✅ Spec/manifesto (este doc).
- ✅ Landing con demo animada.
- ✅ Repository skeleton en GitHub.

### 🟡 Fase 1 — PoC MVP (Semana 1-2)

- CLI `adkd scan` con **12 reglas críticas** (subset de Compose + coroutines + lifecycle).
- Sub-analyzer Detekt + Android Lint ya integrados (modo read-only).
- Reporter console (rich) + JSON.
- **NO IA todavía** — todo manual.

### 🟡 Fase 2 — AI Fixer local (Semana 3)

- `adkd fix --ai` invocando **Claude Code** por línea de comandos.
- Prompt builder estructurado.
- Modo `suggest` (no toca código), valida con un proyecto de prueba (`examples/bad-project`).

### 🟠 Fase 3 — Health Score + CI (Semana 4)

- Cálculo del score robusto.
- Output SARIF + integración con **GitHub Code Scanning**.
- `adkd hook install` para pre-push.
- Quality gate en CI con `fail-below`.

### 🟠 Fase 4 — Rule packs extensibles (Semana 5-6)

- Sistema de rule-packs publicable en npm/GitHub (estilo ESLint shareable configs).
- Documentación autogenerada por regla desde TS source.

### 🔴 Fase 5 — Integración MobiAI (Semana 7-8)

- `adkd` instalable como MobiAI-skill.
- `mobiai doctor --code`.
- Findings en `Graph`.
- Provider "MobiAI" en `ai-fixer` bridge.

### 🔴 Fase 6 — Comunidad (Mes 3+)

- Marketplace de rule-packs.
- Dashboard web público de proyectos rankeados (opt-in, anon).
- Plugin Android Studio.

---

## 12. Quickstart para developers

```bash
# 1. Instalar (cualquiera de las 3 formas)
npm i -g adkd
brew install adkd
# o standalone (como MobiAI)
curl -fsSL https://adkd.dev/install.sh | sh

# 2. Inicializar en tu proyecto Android
cd tu-proyecto-android
adkd init
# → crea adkd.config.ts con defaults

# 3. Escanear
adkd scan
# → muestra score + findings rich console

# 4. (Opcional) Dejar que la IA arregle
adkd fix --ai
# → sugiere fixes, te pregunta antes de aplicar

# 5. Instalar hook de pre-push (bloquea pushes con score bajo)
adkd hook install --fail-below 80
```

---

## 13. Cómo contribuir reglas

Una regla custom es una **función TypeScript** que recibe un AST de Kotlin + metadata y emite 0-N findings:

```ts
// rules/my-pack/compose-heavy-composable.ts
import type { Rule, Finding } from 'adkd'

export default {
  id: 'compose-heavy-composable',
  severity: 'warning',
  docs: {
    description: 'Composable con más de X líneas debería dividirse',
    category: 'compose-performance',
  },
  configSchema: {
    maxLines: { type: 'number', default: 250 },
  },
  check(file, ctx): Finding[] {
    if (file.lines > ctx.config.maxLines && file.hasComposableAnnotation) {
      return [{
        ruleId: 'compose-heavy-composable',
        severity: 'warning',
        file: file.path,
        line: 1,
        message: `Composable con ${file.lines} líneas, debería dividirse (max: ${ctx.config.maxLines}).`,
        fixHint: 'Extrae sub-composables por responsabilidad visual.',
      }]
    }
    return []
  },
} satisfies Rule
```

Publicar:

```bash
adkd rules publish ./my-rule-pack
# → publica a npm como @mi-org/adkd-rule-*
```

---

## 14. Referencias y links

### 14.1 Inspiración directa

- [`react-doctor`](https://github.com/millionco/react-doctor) — el modelo a seguir.
- [`Million.js`](https://million.dev/) — performance React, autores de react-doctor.
- [MobiAI](https://mobiai.dev/) y [Aris Guimera](https://github.com/ArisGuimera) — el aliado de distribución.

### 14.2 Herramientas Android reutilizadas

- [Detekt](https://detekt.dev/) — static analysis Kotlin.
- [Android Lint](https://developer.android.com/studio/write/lint) — reglas oficiales.
- [KtLint](https://ktlint.github.io/) — estilo Kotlin.
- [Kotlin Symbol Processing (KSP)](https://kotlinlang.org/docs/ksp-overview.html) — para reglas con type info.
- [Compose Compiler Reports](https://developer.android.com/jetpack/compose/performance) — métricas de recomposición.

### 14.3 Formatos y specs

- [SARIF 2.1 (OASIS)](https://docs.oasis-open.org/sarif/sarif/v2.1.0/cs01/sarif-v2.1.0-cs01.html) — el estándar para code scanning.
- [JSON Schema para findings](https://json-schema.org/) — validación del output.

### 14.4 LLMs / agentes CLI compatibles

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/overview).
- [Cursor CLI](https://cursor.com/).
- [Gemini CLI](https://github.com/google-gemini/gemini-cli).
- [GitHub Copilot CLI](https://docs.github.com/en/copilot/github-copilot-in-the-command-line).

### 14.5 Docs internas del PoC

- `/examples/bad-project` — proyecto de prueba deliberadamente roto.
- `/examples/good-project` — el mismo proyecto tras un `adkd fix --ai`.
- `/docs/rules` — spec de cada regla, autogenerada desde TS source.

---

## 🔖 Licencia y atribución

- **MIT License** — mismo que react-doctor y MobiAI, alineado para máxima compatibilidad.
- Inspirado y acreditado: Million (`react-doctor`), Aris Guimera (`MobiAI`), JetBrains (`Detekt`, `KtLint`), Google (`Android Lint`).

---

> **Preguntas, críticas, ideas?** Abrir un issue en el repo con la etiqueta `discussion`. Las reglas más votadas se priorizan en el roadmap.

— *Fin del spec*
