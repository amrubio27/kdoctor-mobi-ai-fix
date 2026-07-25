---
name: github-cli
description: Gestiona flujos de GitHub Actions, revisiones de Pull Requests e integración SARIF para kdoctor.
when_to_use: Usar para validar archivos de workflow (.github/workflows) o probar la salida SARIF de kdoctor scan --sarif.
---

# GitHub CLI & Actions Skill

## Comandos Principales

### 1. Validación de Workflows y Diff
```bash
git diff main
```

### 2. Inspección de Salida SARIF
```bash
kdoctor scan --sarif > output.sarif.json
```
