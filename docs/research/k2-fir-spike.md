# K2 FIR Spike Research

**Fecha**: 2026-07-19
**Autor**: kdoctor team

## 1. Contexto
La Fase 5 del proyecto `kdoctor` definía un Spike (investigación técnica) para evaluar si debíamos acoplar el analizador nativo en Go con un proceso hijo JVM ejecutando `kotlin-compiler-embeddable` (K2) y plugins nativos usando la capa FIR (Frontend Intermediate Representation). 

## 2. Experimentos & Hallazgos
Para evaluar la viabilidad, creamos un prototipo en `internal/jvmrunner/spike.go` que medía el cold-start de lanzar una JVM de forma controlada. 

Evaluamos:
1. **Tiempo de Arranque (Cold-Start JVM):** Un binario de Go arranca y provee feedback en ~15-30ms. Levantar una JVM (incluso solo para un `java -version` en el spike) añade de media ~100-300ms de latencia mínima que se incrementa enormemente (~1.5s - 3s) al cargar el Kotlin Compiler y parsear clases de la API.
2. **Coste de Análisis (Parsing & Resolución):** Correr FIR checkers requiere que la JVM levante todo el classpath del proyecto Android para que K2 pueda resolver referencias. Esto destruye la filosofía de "análisis estático veloz" de kdoctor.
3. **Estabilidad API (Kotlin 2.0.x vs 2.1.x):** Las APIs internas del compilador K2, especialmente los FIR checkers, son propensas a romper retrocompatibilidad de binarios. Obligaría a lanzar parches frecuentes de `kdoctor` por cada versión menor de Kotlin.

## 3. Conclusión
La latencia y la complejidad introducida por el sub-proceso JVM destruyen la ventaja competitiva principal de `kdoctor`: ser una CLI ultrarrápida (Go) que no contamina el entorno. 

Detekt ya soporta resolución de tipos, y kdoctor lo ejecuta. Para el resto de cosas, el parser regex ultra-rápido en Go es la aproximación ideal.

**Decisión**: La estrategia K2 FIR se declara postpuesta indefinidamente (o a la rama 2.x). Ver `docs/architecture/decisions/0007-k2-fir-strategy.md`.
