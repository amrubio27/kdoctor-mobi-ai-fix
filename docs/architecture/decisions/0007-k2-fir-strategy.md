# 7. K2 FIR Strategy: Go Puro vs JVM Subprocess

Date: 2026-07-19

## Status

Accepted

## Context

El analizador estático `kdoctor` fue diseñado bajo la premisa de ser un binario ultra-rápido en Go, con análisis semántico delegado a Detekt y análisis regex/AST gestionado internamente en Go.
Con la llegada de Kotlin 2.0 y el nuevo Kotlin Compiler (K2), surge la pregunta técnica de si `kdoctor` debería acoplarse con la API de compilación de Kotlin a través de FIR (Frontend Intermediate Representation) para crear sus propias reglas nativas basadas en resolución de tipos (e.g. `compose-remember-missing`), corriendo la JVM como un subproceso gestionado por Go.

## Decision

**Mantendremos `kdoctor` en Go puro y posponemos la integración JVM/K2 FIR al roadmap de una potencial versión 2.x.**

Tras investigar el coste operacional (Spike, ver `docs/research/k2-fir-spike.md`), hemos determinado que:
1. Añadir una JVM incrementa drásticamente la latencia de cold-start de `kdoctor`.
2. La necesidad de parsear todo el classpath para resolver símbolos de FIR destruiría el rendimiento en CLI.
3. Requiere atar la versión de la herramienta a versiones específicas de `kotlin-compiler-embeddable` debido a la inestabilidad de la API interna.

Confiaremos el análisis estructural pesado a Detekt y a las nuevas versiones de lint que Kotlin provea nativamente (e.g. a través del formato SARIF), manteniendo nuestra lógica central liviana, rápida y agnóstica.

## Consequences

- Evitamos dependencias JVM directas en la base del código principal.
- La ejecución en CLI se mantiene veloz (tiempo de parseo de regex puro + el tiempo estándar que ya tarda Detekt).
- Las tareas 5.3 y posteriores referentes a implementaciones directas de checkers en FIR quedan oficialmente aplazadas (roadmap-2.x).
