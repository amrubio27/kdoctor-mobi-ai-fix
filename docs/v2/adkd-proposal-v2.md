# adkd Â· Documento Maestro Revisado (V2) â€” MobIAI Skill Addon

> ExtraÃ­do del PDF "MobiAi Cli PoC Addon.pdf". La segunda revisiÃ³n sustituye a la V1 en cuanto a arquitectura tÃ©cnica, manteniendo las reglas del catÃ¡logo original.

---

 Documento Maestro de Definición:

adkd (Android/KMP Doctor AI Fix)

Este documento define la arquitectura, fundamentos y hoja de ruta iterativa para el desarrollo
de adkd, una herramienta de análisis estático y reparación automática de código potenciada
por IA para el ecosistema Android, Kotlin Multiplatform (KMP) y Compose Multiplatform (CMP).
El objetivo es construir un orquestador que evalúe la salud del proyecto y delegue la resolución
de problemas a un LLM (a través de MobiAI), garantizando la seguridad del código, el respeto
por la arquitectura y la escalabilidad del análisis.

1. Fundamentos Arquitectónicos: El Enfoque Híbrido

Para evitar cuellos de botella en el parseo y mantener la herramienta robusta frente a las
actualizaciones de Kotlin, adkd se divide en dos capas con responsabilidades únicas y estrictas:

    El Cerebro (Capa JVM - Análisis Estático): Utiliza herramientas nativas del ecosistema
        (Detekt, Lint). Las reglas personalizadas se construyen sobre la API PSI (Program
        Structure Interface) de Kotlin. Esta capa lee el AST, detecta antipatrones, violaciones de
        SOLID y malos olores en Compose, y exporta los hallazgos.

    El Orquestador (Capa TypeScript - CLI): Un ejecutable Node.js ágil que gestiona el
        escaneo incremental (mediante hashes de archivos modificados en Git), lee los reportes
        XML/SARIF de la capa JVM, normaliza los datos, calcula el Health Score y estructura la
        información para la IA.

2. Gestión del Contexto para la IA (Prevención de

Alucinaciones)

Para que el LLM realice refactorizaciones complejas sin destruir la arquitectura ni perderse en
archivos gigantes, adkd implementa tres técnicas de inyección de contexto:

   1. Skeleton Context (Extracción de Esqueletos): El orquestador extrae y envía a la IA
        únicamente las firmas públicas (contratos, interfaces, constructores) de las
        dependencias del archivo infractor. Se oculta la lógica de implementación interna para
        ahorrar tokens y enfocar a la IA en el contrato.

   2. Boundary Constraints (Manifiesto de Arquitectura): Se inyecta dinámicamente un
        conjunto de reglas basadas en la estructura de módulos (ej. "Estás en commonMain, no
        puedes usar dependencias de androidMain ni java.*").

   3. Plan-and-Execute: Para problemas arquitectónicos (ej. un Composable masivo
        acoplado a datos), se fuerza a la IA a devolver primero un plan de refactorización. Solo
        tras definir las interfaces y responsabilidades, se le pide la generación del código.

3. Motor de Evaluación: Health Score Escalable

El Health Score abandona la deducción lineal absoluta. Se calcula mediante una métrica de
densidad para ser justo tanto en micro-módulos como en monolitos legacy:

    Métrica base: Issues per 1K Lines of Code (Issues/KLOC).
    Ponderación: Los errores de arquitectura (acoplamiento circular) y rendimiento crítico
        en Compose (listas sin llaves, recomposiciones inútiles) tienen mayor peso que los avisos
        de estilo.
    Salida: Un porcentaje (0-100%) que representa la salud del código analizado,
        permitiendo establecer Quality Gates en CI/CD (ej. fallar si el score cae por debajo del
        85%).

4. Circuito Cerrado de Validación (Safe Auto-Fix)

La IA no tiene la última palabra. Antes de presentar un git diff al desarrollador, adkd implementa
un TDD inverso:

   1. La IA propone y aplica el parche temporalmente.
   2. adkd lanza en background la compilación incremental (./gradlew assembleDebug o el

        target afectado).
   3. Si el parche rompe la compilación o los tests afectados, el error del compilador se

 inyecta de vuelta a la IA para que lo corrija (máximo 2 intentos).
    Roadmap de Desarrollo Iterativo (Instrucciones

para la IA)

Esta sección divide el proyecto en fases secuenciales. La IA debe utilizar estos pasos como un
backlog estricto, completando y validando cada bloque antes de avanzar al siguiente.

Fase 1: El Orquestador Base y Normalización (TypeScript)

Objetivo: Crear el esqueleto del CLI y el contrato de datos.
   1. Inicialización: Configurar un proyecto Node.js/TypeScript con un binario CLI básico
        usando librerías modernas de terminal (ej. clack/prompts, commander).
   2. Módulo Runner: Crear un servicio que ejecute un comando de Gradle (ej. ./gradlew
        detekt) en un proceso hijo (spawn).
   3. Escaneo Incremental: Implementar la lógica para leer qué archivos han cambiado según
        Git (git status o git diff) para pasar solo esos paths al Runner.
   4. Módulo Adapter: Escribir el parser que toma la salida XML/SARIF de Detekt/Lint y la
        transforma en la interfaz Finding (JSON normalizado con id, file, line, message, severity).

Fase 2: Motor de Puntuación (Health Score)

Objetivo: Implementar la lógica matemática para evaluar la salud del código.
   1. Contador LOC: Crear un script ligero que cuente las líneas de código totales
        (excluyendo comentarios y blancos) de los archivos analizados.
   2. Algoritmo de Densidad: Implementar la fórmula de Issues/KLOC ponderada por
        severidad.
   3. Reporter UI: Diseñar la salida en la terminal. Mostrar el breakdown de severidades y el
        Health Score final con un diseño visual claro.

Fase 3: Integración como Skill de MobiAI

Objetivo: Conectar el orquestador con la infraestructura de ejecución de IA.
   1. Adaptador MobiAI: Exponer el comando adkd scan --json para que MobiAI lo consuma
        internamente.

   2. Generador de Prompts: Desarrollar el módulo que toma un Finding específico y
        construye el prompt base estructurado (Rol + Tarea + Archivo + Error).

   3. Inyección en Graph: Escribir la documentación de integración para que los hallazgos de
        adkd se inyecten como anotaciones en el Graph de MobiAI.

Fase 4: Inyección de Contexto Arquitectónico

Objetivo: Implementar los escudos contra alucinaciones del LLM.
   1. Analizador de Manifiesto (Boundary Builder): Crear un módulo que lea el
        build.gradle.kts del módulo analizado para determinar sus dependencias y establecer las
        reglas restrictivas del prompt (ej. "Módulo Dominio: prohibido importar UI").
   2. Extractor de Esqueletos (Skeleton Parser): Utilizar una herramienta ligera (o
        expresiones regulares avanzadas como PoC) para extraer firmas de funciones e
        interfaces de los archivos importados por el código infractor, adjuntándolas al prompt.

Fase 5: Circuito Cerrado de Validación (Safe Fix)

Objetivo: Garantizar que los cambios de la IA compilan.
   1. Gestor de Estados (Sandbox): Crear un mecanismo que aplique el código de la IA en
        memoria o en un stash temporal.
   2. Bucle de Compilación: Ejecutar el target de Gradle afectado.
   3. Feedback Loop: Si falla, capturar el stderr, anexarlo al prompt y reenviarlo a MobiAI. Si
        acierta, aplicar el parche al sistema de archivos real y generar el git diff.

Fase 6: Reglas Nativas (Capa JVM)

Objetivo: Desarrollar las reglas específicas de Android/KMP que detectan los antipatrones.
   1. Setup Detekt Custom Rules: Configurar un módulo Kotlin JVM dedicado a empaquetar
        reglas personalizadas de Detekt.
   2. Reglas de Compose: Implementar reglas basadas en PSI para detectar (ej.) estado sin
        remember o modificadores inestables.
   3. Reglas de Corrutinas: Detectar inyección de Dispatchers hardcodeados o scopes no
        estructurados.
   4. Distribución: Asegurar que el CLI Typescript sepa cómo inyectar el .jar de estas reglas
        custom al ejecutar la tarea de Gradle en el proyecto destino.
        
        
        Segunda revision de la propuesta (VER CON DETENIMIENTO):

   
             Documento Maestro de Definición Revisado: adkd (Android/KMP Doctor AI Fix)

Visión General: adkd es un motor de análisis estático nativo y un orquestador de
       refactorización agentiva para Kotlin Multiplatform (KMP) y Compose
       Multiplatform (CMP). A diferencia de los linters tradicionales, adkd no solo calcula
       un Health Score del proyecto, sino que utiliza una arquitectura híbrida
       ultrarrápida, reglas basadas en el nuevo compilador de Kotlin (K2) e integración
       semántica profunda con MobiAI Graph para delegar arreglos a modelos de IA
       utilizando estrategias de prompting validadas científicamente.

1. Arquitectura de Ingestión Híbrida (El fin del cuello de botella)

Para evitar los tiempos muertos que supondría invocar tareas de Gradle desde un
       entorno Node.js, adkd adopta un enfoque híbrido, desarrollando su núcleo CLI en
       Go, alineándose con el ecosistema de MobiAI.

El análisis se divide en dos fases:
     Pase Rápido (Motor Sintáctico Local): Un escáner ultrarrápido en Go que
         procesa el código línea por línea mediante expresiones regulares optimizadas
         (reutilizando la tecnología de MobiAI Graph). Es capaz de escanear archivos en
         milisegundos para detectar antipatrones estructurales básicos en Compose (ej.
         estado mutable sin bloque remember).
     Pase Profundo (Puente de Ingestión Diferida): Para análisis que requieren
         resolución de tipos compleja, adkd delega la ejecución a un proceso en
         background de Detekt/Lint a través de Gradle, y luego parsea eficientemente el
         reporte generado en formato estándar JSON o SARIF. Este reporte se une al del
         pase rápido para conformar el Health Score unificado.

2. Motor de Reglas K2 (FIR) y Sinergia Comunitaria

En lugar de basarnos en la API PSI (que quedará obsoleta o marginada) o reescribir
       docenas de reglas desde cero, el análisis profundo de adkd aprovecha:

     Frontend Compiler Plugins (K2): Las reglas personalizadas complejas de adkd
         se construirán como extensiones del frontend del compilador Kotlin K2,
         enganchándose nativamente mediante FirAdditionalCheckersExtension.
         Esto permite interceptar los nodos de sintaxis abstracta (AST) directamente
         durante la compilación oficial de JetBrains.

     Empaquetado de Reglas Comunitarias: adkd se nutrirá de configuraciones ya
         probadas por la comunidad, como el paquete detekt-rules-compose de
         appKODE (para atrapar errores comunes como ReusedModifierInstance o
         ModifierHeightWithText), enfocando el valor de adkd en puntuar y reparar,
         no en reinventar reglas.

3. Estrategia de IA: Prompts de Calidad vs. "Safe Auto-Fix"

El circuito de retroalimentación propuesto originalmente (donde la IA compilaba
       iterativamente el código) ha sido rediseñado basándose en estudios empíricos
       sobre la calidad de código generado por LLMs.

     Evitar RCI (Recursive Criticism and Improvement): Se ha demostrado que las
    estrategias donde el LLM itera y critica recursivamente sus propios errores en
    bucle tienden a degradar la calidad del código, aumentando la verbosidad
    excesiva, introduciendo refactorizaciones alucinadas y rompiendo el formato.
 Quality-Focused Prompting: En su lugar, el puente de IA de adkd utilizará
    estrategias Quality-Focused (Prompts Centrados en Calidad) de un solo paso.
    Añadir directivas estrictas de calidad en un prompt simple reduce la densidad
    de los "malos olores" de código entre un 7% y un 15% sin los peligros de la
    alucinación recursiva.
 La Brecha Sintaxis-Lógica: Como los LLMs suelen generar código
    sintácticamente válido (98%) pero lógicamente deficiente (alucinando variables
    o colisionando namespaces), adkd bloqueará cambios que no superen su
    validación estructural.

4. Integración Profunda con MobiAI Graph (Prevención de Alucinaciones)

En vez de parsear manifiestos de arquitectura desde build.gradle.kts, adkd se
       sincroniza con MobiAI Graph.

     Cuando la IA recibe la instrucción de arreglar un problema complejo (ej.
         clean-boundary-leak), interroga automáticamente al índice local
         regenerable de Graph.

     Esto le permite al agente IA (como Claude Code o Cursor) recibir en
         milisegundos una lista clasificada de los 10 archivos relevantes que tocan esa
         dependencia, reduciendo drásticamente el consumo de tokens y evitando
         modificaciones a ciegas.

5. Sinergia CI/CD: Flujo de Trabajo en Pull Requests

La verdadera adopción requiere ser invisible pero útil en Integración Continua.
       Inspirado en herramientas como react-doctor:

     GitHub Actions / GitLab CI Nativas: adkd proveerá acciones oficiales que
         comentan directamente sobre las Pull Requests.

     Líneas Base (Baselines) Incrementales: Para que los proyectos Legacy puedan
         adoptar adkd sin bloquear todas las compilaciones, el CLI generará archivos de
         baseline. El sistema --diff main asegurará que el Health Score solo castigue
         o requiera reparación para los code smells nuevos introducidos en el commit
         actual, ignorando pacíficamente la deuda técnica histórica.

 Roadmap de Desarrollo Modificado (Fases)

Fas  Hito                                  Entregables Técnicos

e
Fas    Motor Sintáctico   CLI en Go con expresiones regulares para escanear
    e      Local (Go)         Composables simples (búsqueda de faltas de
    1                         remember o state hoisting) en
                              milisegundos.
Fas    Puente de
    e      Ingestión y    Comando Cobra CLI (adkd scan). Lanzamiento de
    2      Reportes           Detekt en background, parseo del archivo SARIF
                              generado y unificación en un Health Score.

Fas    Integración        Implementación de adkd scan --diff main
    e      CI/CD y            para GitHub Actions, gestión de líneas base
    3      Baselines          (baseline.xml) y inline comments en las PR.

Fas    Motor de           Creación del archivo SKILL.md para empaquetar
    e      Refactorizaci      adkd como habilidad de MobiAI. Prompts
    4      ón Agentiva        Quality-Focused conectados a MobiAI Graph
                              para arreglos locales.

Fas    Checkers           Desarrollo de plugins del compilador Kotlin usando
    e      Avanzados K2       FirAdditionalCheckersExtension para
    5                         reglas arquitectónicas precisas a prueba de
                              futuro.

¿Qué logramos con estas correcciones? Al pasarnos a Go, evitamos la lentitud de
       Node.js. Al utilizar MobiAI Graph, evitamos reinventar parsers de dependencias.
       Al aplicar promting Quality-Focused, nos protegemos contra el código
       defectuoso de las iteraciones IA. Y al usar extensiones FIR de K2, garantizamos
       que la herramienta sobreviva a la próxima década del ecosistema Kotlin.

   5.
