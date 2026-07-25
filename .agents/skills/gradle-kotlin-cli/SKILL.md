---
name: gradle-kotlin-cli
description: Automatiza la ejecución de Gradle y análisis Kotlin en proyectos target (sandbox) para verificar que los arreglos sugeridos por kdoctor compilan correctamente.
when_to_use: Usar al ejecutar tests end-to-end de kdoctor contra un proyecto Android/KMP target.
---

# Gradle & Kotlin CLI Skill

## Comandos Principales

### 1. Compilación y Diagnóstico Gradle
```bash
./gradlew assembleDebug --daemon
./gradlew lint
```

### 2. Ejecución de Detekt en Proyecto Target
```bash
./gradlew detekt
```
