# setup-mcp.ps1
# Script de automatización para compilar kdoctor y kdoctor-mcp y mostrar la configuración de Cursor

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = [System.IO.Path]::GetFullPath((Join-Path $scriptDir ".."))

Set-Location $rootDir

Write-Host "==> Compilando kdoctor.exe..." -ForegroundColor Cyan
go build -o kdoctor.exe ./cmd/kdoctor

Write-Host "==> Compilando kdoctor-mcp.exe..." -ForegroundColor Cyan
go build -o kdoctor-mcp.exe ./cmd/kdoctor-mcp

$kdoctorBin = Join-Path $rootDir "kdoctor.exe"
$kdoctorMcpBin = Join-Path $rootDir "kdoctor-mcp.exe"

if (Test-Path $kdoctorBin) {
    Write-Host "[OK] kdoctor.exe compilado correctamente: $kdoctorBin" -ForegroundColor Green
} else {
    Write-Error "[ERROR] No se pudo encontrar $kdoctorBin"
}

if (Test-Path $kdoctorMcpBin) {
    Write-Host "[OK] kdoctor-mcp.exe compilado correctamente: $kdoctorMcpBin" -ForegroundColor Green
} else {
    Write-Error "[ERROR] No se pudo encontrar $kdoctorMcpBin"
}

$kdoctorBinEscaped = $kdoctorBin.Replace('\', '\\')
$kdoctorMcpBinEscaped = $kdoctorMcpBin.Replace('\', '\\')

$mcpJsonConfig = @"
{
  "mcpServers": {
    "kdoctor": {
      "command": "$kdoctorMcpBinEscaped",
      "env": {
        "KDOCTOR_BIN": "$kdoctorBinEscaped"
      }
    }
  }
}
"@

Write-Host "========================================================" -ForegroundColor Yellow
Write-Host " Configuración para Cursor (Settings -> MCP -> Add MCP Server):" -ForegroundColor Yellow
Write-Host "========================================================" -ForegroundColor Yellow
Write-Host $mcpJsonConfig
Write-Host "========================================================" -ForegroundColor Yellow

$installAntigravityScript = Join-Path $scriptDir "install-antigravity-mcp.ps1"
if (Test-Path $installAntigravityScript) {
    Write-Host "`n==> Registrando MCP Tools y Skill en Antigravity IDE..." -ForegroundColor Cyan
    & $installAntigravityScript
}
