# install-antigravity-mcp.ps1
# Script para registrar kdoctor MCP Server y Skill en Antigravity IDE

$ErrorActionPreference = "Stop"

$mcpDir = "C:\Users\Miguel\.gemini\antigravity-ide\mcp\kdoctor"
if (!(Test-Path $mcpDir)) {
    New-Item -ItemType Directory -Force -Path $mcpDir | Out-Null
}

$scanJson = @'
{
  "name": "kdoctor_scan",
  "description": "Run a kdoctor quality scan on a project directory and return a JSON report.",
  "parameters": {
    "type": "object",
    "properties": {
      "projectDir": { "type": "string", "description": "Project directory to scan." },
      "projectType": { "type": "string", "description": "Project type: android, kmp, cmp, compose, jvm, gradle, plain." },
      "format": { "type": "string", "description": "Output format: json or sarif." },
      "detektBin": { "type": "string", "description": "Explicit path to detekt." },
      "failBelow": { "type": "integer", "description": "Fail if health score is below this value." },
      "diffRef": { "type": "string", "description": "Git ref for diff scan." },
      "baselinePath": { "type": "string", "description": "Baseline file path." }
    }
  }
}
'@

$rulesJson = @'
{
  "name": "kdoctor_rules",
  "description": "List the curated kdoctor rule catalog with status and severity.",
  "parameters": {
    "type": "object",
    "properties": {}
  }
}
'@

$initJson = @'
{
  "name": "kdoctor_init",
  "description": "Bootstrap kdoctor configuration files in a project directory.",
  "parameters": {
    "type": "object",
    "properties": {
      "projectDir": { "type": "string", "description": "Project directory." },
      "projectType": { "type": "string", "description": "Project type." },
      "force": { "type": "boolean", "description": "Overwrite existing config files." }
    }
  }
}
'@

$doctorJson = @'
{
  "name": "kdoctor_doctor",
  "description": "Diagnose the kdoctor environment (Go, Detekt, Gradle, LLM providers).",
  "parameters": {
    "type": "object",
    "properties": {}
  }
}
'@

$fixSuggestJson = @'
{
  "name": "kdoctor_fix_suggest",
  "description": "Generate AI-driven fix suggestions for a project without applying them.",
  "parameters": {
    "type": "object",
    "properties": {
      "projectDir": { "type": "string", "description": "Project directory to analyze." },
      "detektBin": { "type": "string", "description": "Explicit path to detekt." },
      "contextLines": { "type": "integer", "description": "Number of context lines around findings." }
    }
  }
}
'@

$instructions = @'
# kdoctor MCP Server
Exposes kdoctor CLI capabilities for auditing and diagnosing Android/KMP/CMP Kotlin code.
Binary: C:\Users\Miguel\Desktop\doctor mobi ai fix\kdoctor.exe
MCP Server: C:\Users\Miguel\Desktop\doctor mobi ai fix\kdoctor-mcp.exe
'@

Set-Content -Path (Join-Path $mcpDir "kdoctor_scan.json") -Value $scanJson -Encoding UTF8
Set-Content -Path (Join-Path $mcpDir "kdoctor_rules.json") -Value $rulesJson -Encoding UTF8
Set-Content -Path (Join-Path $mcpDir "kdoctor_init.json") -Value $initJson -Encoding UTF8
Set-Content -Path (Join-Path $mcpDir "kdoctor_doctor.json") -Value $doctorJson -Encoding UTF8
Set-Content -Path (Join-Path $mcpDir "kdoctor_fix_suggest.json") -Value $fixSuggestJson -Encoding UTF8
Set-Content -Path (Join-Path $mcpDir "instructions.md") -Value $instructions -Encoding UTF8

Write-Host "[OK] MCP Tools registrados exitosamente en Antigravity IDE ($mcpDir)" -ForegroundColor Green
