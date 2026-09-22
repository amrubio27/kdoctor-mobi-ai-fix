# kdoctor

**Auditoría de salud de código para Android, Kotlin Multiplatform y Compose Multiplatform.**

Analiza tu proyecto, te da una nota de 0 a 100 y te dice qué arreglar, con fichero, línea y una pista concreta. Pensado tanto para que lo leas tú como para que lo consuma tu agente de IA.

Inspirado en react-doctor. Diseñado como complemento de [MobiAI](https://github.com/ArisGuimera/MobiAI-Core).

```
Health Score: 85/100
9 errors  ·  18 warnings  ·  96 info  ·  123 total

Top clusters:
  1. [magic-numbers] 77 issues
  2. [compose-performance] 18 issues
  3. [architecture] 13 issues
```

---

## Por qué existe

Las herramientas de análisis estático te dan una lista. kdoctor te da **una lista priorizada y un número al que seguir la pista en el tiempo**.

La diferencia práctica está en tres decisiones:

- **El score mide densidad de deuda, no volumen.** Está normalizado por KLOC, así que un proyecto grande no puntúa peor por ser grande. Dos proyectos distintos son comparables, y el mismo proyecto lo es consigo mismo según crece.
- **Las reglas pesan según lo que cuestan después.** Seguridad ×2, arquitectura ×1,5, formato ×0,75. Cien *magic numbers* no tapan una fuga de credenciales.
- **El código de test se juzga con otra vara.** Un test importa legítimamente de la capa de datos e instancia colaboradores a mano. Las reglas de arquitectura y diseño no se aplican ahí; las de seguridad sí, porque una credencial filtrada lo está en cualquier fichero.

## Instalación

**Windows**
```powershell
irm https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.ps1 | iex
```

**macOS / Linux**
```bash
curl -fsSL https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.sh | sh
```

**Desde el código** (necesita Go 1.23+)
```bash
go install github.com/amrubio27/kdoctor-mobi-ai-fix/cmd/kdoctor@latest
```

Hay binarios para Windows, macOS (Intel y Apple Silicon) y Linux (x86-64 y ARM64) en [Releases](https://github.com/amrubio27/kdoctor-mobi-ai-fix/releases), con `sha256sums.txt`.

### Requisitos

Un **JDK 11-21**. Nada más.

kdoctor descarga detekt y el plugin de reglas de Compose la primera vez que los necesita, verifica su SHA-256 y los cachea en `~/.kdoctor/tools/`. No tienes que instalar detekt, ni configurar Gradle, ni tocar ningún fichero.

Si no encuentra un JDK compatible el escaneo **sigue funcionando** con las 20 reglas nativas, y te dice exactamente qué falta. `kdoctor doctor` te lo cuenta antes de empezar:

```
✓ Java    : 17  (JAVA_HOME)
✓ detekt  : cached
✓ Gradle  : gradlew encontrado

  Rules     : 117 catalogued, 64 live (20 native + 44 via detekt)
  Next scan : ✓ 64/64 rules
```

## Uso

```bash
cd tu-proyecto
kdoctor scan
```

Eso es todo. Sin `init`, sin configuración previa.

### Comandos

| comando | qué hace |
|---|---|
| `kdoctor scan` | Analiza y puntúa |
| `kdoctor doctor` | Qué podrá evaluar el próximo escaneo, y por qué |
| `kdoctor fix` | Plan de remediación para que lo aplique tu agente |
| `kdoctor init` | Genera `kdoctor.config.yaml` y `detekt.yml` a medida del stack |
| `kdoctor rules` | Inspecciona o actualiza el catálogo |

### Formatos de salida

```bash
kdoctor scan --summary   # solo el score y los clusters principales
kdoctor scan --json      # estructurado, para agentes y scripts
kdoctor scan --html      # informe autocontenido para compartir
kdoctor scan --md        # markdown legible
kdoctor scan --sarif     # GitHub Code Scanning
```

El informe HTML es un único fichero que funciona sin conexión, con filtros combinables por severidad y por cluster.

### Sólo lo que has cambiado

```bash
kdoctor scan --diff main
```

Filtra los hallazgos a las líneas que tocaste respecto a esa rama. Es lo más útil al revisar: centra la atención en lo que acabas de escribir, no en la deuda histórica.

### Integración continua

```bash
kdoctor scan --json --fail-below 80
```

Sale con código distinto de cero si el score baja del umbral. También puedes fijarlo en `kdoctor.config.yaml`; el flag tiene prioridad.

```yaml
- name: kdoctor
  run: kdoctor scan --sarif --out=results.sarif

- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

## Para agentes de IA

kdoctor **no llama a ningún modelo de lenguaje y nunca edita tus ficheros**. El agente que lo invoca ya tiene un modelo; lanzar otro duplica coste y latencia.

En su lugar emite lo que ese modelo necesita:

```bash
kdoctor fix --json
```

Por cada hallazgo: la ventana de código numerada, la regla, la pista de arreglo y **el rango exacto de líneas a reemplazar**. Sin ese rango, el agente tiene que adivinar qué sustituir.

Después de aplicar los cambios:

```bash
kdoctor fix --validate src/main/kotlin/MiFichero.kt
```

Pasa un lexer de Kotlin sobre el fichero y comprueba que sigue balanceado. Es un *sanity check*, no un compilador: caza el fallo típico de un parche que se dejó una llave por el camino.

### Servidor MCP

`kdoctor-mcp` expone `kdoctor_scan`, `kdoctor_rules`, `kdoctor_init`, `kdoctor_doctor` y `kdoctor_fix_suggest` por stdio a Claude Code, Cursor o Cline:

```json
{
  "mcpServers": {
    "kdoctor": {
      "command": "/ruta/a/kdoctor-mcp",
      "env": { "KDOCTOR_BIN": "/ruta/a/kdoctor" }
    }
  }
}
```

Se publica junto al binario principal en cada release.

### Como skill

[`SKILL.md`](SKILL.md) sigue el estándar agentskills.io y es autocontenido. Cópialo al host que uses:

```bash
mkdir -p ~/.claude/skills/kdoctor && cp SKILL.md ~/.claude/skills/kdoctor/
```

Con MobiAI instalado encadena bien con MobiAI Graph: `mobiai graph context "audit module app"` acota el radio de impacto, `kdoctor scan --diff main --json` lo mide. Ver [docs/integrations/mobiai.md](docs/integrations/mobiai.md).

## El catálogo de reglas

**117 reglas catalogadas, 64 activas.** De las activas:

| origen | reglas | qué cubren |
|---|---:|---|
| Detectores nativos en Go | 20 | Arquitectura por capas, Compose, corrutinas, seguridad. Sin JVM |
| detekt core | 23 | Complejidad, naming, dead code, excepciones |
| Plugin de reglas de Compose | 21 | `ModifierMissing`, `LambdaParameterInRestartableEffect`, `ViewModelForwarding`… |

Las 53 restantes están marcadas `planned` y **no se evalúan**: el catálogo no promete lo que no cumple. `kdoctor rules list` muestra el estado de cada una y de dónde se cargó.

Clusters con reglas activas: `architecture`, `compose-performance`, `coroutines`, `security`, `error-handling`, `complexity`, `naming`, `dead-code`, `testing`, `formatting`, `magic-numbers`, `clean-code`.

Todavía sin implementar: `kmp`, `memory`, `accessibility`, `lifecycle`.

## Cómo se calcula el score

```
penalización = severidad × peso_del_cluster × rendimientos_decrecientes
score        = 100 − (críticos / √KLOC + resto / KLOC)
```

- **Severidad**: error 5, warning 2, info 0,5.
- **Peso del cluster**: seguridad ×2, arquitectura ×1,5, corrutinas/memoria/lifecycle ×1,25, Compose/testing/KMP ×1, el resto ×0,75.
- **Rendimientos decrecientes**: la cuarta vez que la misma regla salta en el mismo fichero cuenta un cuarto de lo que contó la primera.
- **Los hallazgos críticos** (errores de seguridad, arquitectura o memoria) se normalizan por `√KLOC` en vez de por `KLOC`, así que su peso relativo crece con el tamaño del proyecto: no se diluyen, pero tampoco clavan el resultado.

Calibrado contra proyectos públicos de referencia, no eligiendo constantes a ojo:

| proyecto | score |
|---|---:|
| [skydoves/pokedex-kmp](https://github.com/skydoves/pokedex-kmp) | 85 |
| Un proyecto KMP real de 16,6 KLOC en desarrollo activo | 84 |
| `examples/bad-project` (antipatrones a propósito) | 55 |

**Léelo como una tendencia, no como una nota.** La pregunta útil es "¿este cambio lo mejora o lo empeora?", no "¿72 está bien?".

Por debajo de ~1 KLOC el score no es fiable: hay demasiada varianza para que la densidad signifique nada.

## Configuración

Opcional. `kdoctor init --type=cmp` genera un punto de partida.

```yaml
projectType: cmp          # android | kmp | cmp

excludes:
  - "**/build/**"

# Severidad por cluster o por regla: error | warning | info | off
rules:
  formatting: off
  security: warning
  coroutine-dispatchers-hardcoded: info

score:
  failBelow: 80           # --fail-below tiene prioridad sobre esto
```

Si tu proyecto ya tiene un `detekt.yml`, kdoctor lo usa. Sus reglas se **añaden** a las de detekt por defecto en vez de reemplazarlas, que es lo que casi siempre se quiere decir.

### Variables de entorno

| variable | para qué |
|---|---|
| `KDOCTOR_JAVA` | Apuntar a un JDK concreto |
| `KDOCTOR_DETEKT_JAR` | Usar tu propio detekt en vez del descargado |
| `KDOCTOR_NO_DOWNLOAD` | Prohibir accesos a red (entornos herméticos, Docker) |
| `KDOCTOR_DETEKT_VERSION` | Fijar otra versión de detekt |

## Docker

```bash
docker build -t kdoctor .
docker run --rm -v "$PWD":/project kdoctor scan
```

La imagen trae detekt, el plugin de Compose y un JDK 17 dentro. Funciona sin red y sin flags.

## Contribuir

El catálogo vive en [`scripts/genschema/main.go`](scripts/genschema/main.go), que es la fuente canónica. Para añadir una regla:

1. Añade la entrada al catálogo.
2. `go run ./scripts/genschema -out rules/metadata.json`
3. Copia el resultado a `internal/core/rulemap/metadata.json` (es el que se embebe en el binario).
4. Si es nativa, escribe el detector en [`internal/core/rules/rules.go`](internal/core/rules/rules.go) y cabléalo por ID.

`validateCatalog()` comprueba IDs duplicados, severidades válidas y **colisiones de `detektRule`**: dos reglas no pueden reclamar la misma regla de detekt, porque el índice sólo conserva una y la otra queda inalcanzable.

Antes de abrir un PR:

```bash
gofmt -l .      # sin salida
go vet ./...
go test ./...
go build ./...
```

Si trabajas sobre este repo con un agente, lee primero [`HONEY.md`](HONEY.md).

## Licencia

MIT.
