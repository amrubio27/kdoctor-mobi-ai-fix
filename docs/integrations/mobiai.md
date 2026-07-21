# kdoctor + MobiAI Integration

`kdoctor` puede utilizarse de forma fluida como una "skill" de MobiAI, permitiendo auditar proyectos de Android y aplicar correcciones asistidas por IA directamente desde la CLI de MobiAI.

## Instalación

En tu proyecto Android/KMP, donde ya tengas MobiAI inicializado, ejecuta:

```bash
mobiai skills install kdoctor
```

Esto descargará la skill de kdoctor basándose en el `plugin.json` y expondrá sus comandos en el ecosistema de MobiAI.

## Flujo Recomendado (`mobiai doctor --code`)

Una vez instalado, el comando principal es `mobiai doctor --code`. Este comando actúa como un envoltorio inteligente:

1. **Escaneo Base**: Ejecuta silenciosamente `kdoctor scan --mobiai --json`.
2. **Volcado a Graph**: El flag `--mobiai` hace que `kdoctor` exporte sus hallazgos al grafo interno de MobiAI (en `.mobiai/graph/findings.jsonl`).
3. **Análisis de Impacto**: MobiAI evalúa el Health Score y la severidad de los hallazgos.
4. **Auto-Fix (Opcional)**: Si se encuentran issues reparables, MobiAI preguntará al usuario si desea ejecutar `kdoctor fix --ai`. Si acepta, MobiAI delegará en el fixer de kdoctor pasándole el contexto y las instrucciones.

## Beneficios de Integración
- **Contexto Compartido**: Las anotaciones que kdoctor deja en el grafo de MobiAI ayudan a otros agentes a tener cuidado extra al editar archivos que ya tienen deudas técnicas detectadas.
- **Provider Agnostic**: El comando `fix --ai` usará por defecto el proveedor de LLM que ya hayas configurado en MobiAI (e.g., Claude, Cursor, Gemini, etc.).

## Ejemplo Completo
```bash
# Dar contexto a MobiAI sobre nuestro objetivo
$ mobiai graph context "android audit module app"

# Ejecutar la skill
$ mobiai doctor --code
> [kdoctor] Health Score: 78/100
> [kdoctor] 3 Critical issues found in compose-performance.
> [MobiAI] Do you want kdoctor to fix these issues automatically? [Y/n] Y
> [kdoctor] Applying fixes via interactive mode...
```
