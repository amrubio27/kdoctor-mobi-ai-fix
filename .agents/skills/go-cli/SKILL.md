---
name: go-cli
description: Automatiza la ejecución de tests, compilación, formateo y linter para código Go en kdoctor de acuerdo a HONEY.md.
when_to_use: Usar al validar cambios en código Go, correr pruebas unitarias (go test ./...), comprobar formateo (gofmt) o verificar el binario kdoctor.
---

# Go CLI Skill

## Comandos Principales

### 1. Formateo y Análisis Estático
```bash
gofmt -l .
go vet ./...
```

### 2. Pruebas Unitarias
```bash
go test ./...
go test -v ./internal/...
```

### 3. Compilación
```bash
go build -o kdoctor.exe ./cmd/kdoctor
go build -o kdoctor-mcp.exe ./cmd/kdoctor-mcp
```
