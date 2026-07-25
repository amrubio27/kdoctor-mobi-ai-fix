# install.ps1 — Installer for kdoctor on Windows
# Usage:
#   irm https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'

Write-Host "Installing kdoctor for Windows..." -ForegroundColor Cyan

# 1. Determine destination directory in user LOCALAPPDATA
$installDir = "$env:LOCALAPPDATA\kdoctor\bin"
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

$exePath = Join-Path $installDir "kdoctor.exe"

# 2. Download latest release binary from GitHub Releases
$downloadUrl = "https://github.com/amrubio27/kdoctor-mobi-ai-fix/releases/latest/download/kdoctor-windows-amd64.exe"
Write-Host "Downloading latest release from $downloadUrl..." -ForegroundColor Yellow

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath -UseBasicParsing
} catch {
    # Fallback to local built executable if release download fails (e.g. initial dev phase)
    Write-Host "Notice: Release download unavailable, checking local repository..." -ForegroundColor Yellow
    $localExe = Join-Path $PSScriptRoot "kdoctor.exe"
    if (Test-Path $localExe) {
        Copy-Item -Path $localExe -Destination $exePath -Force
    } else {
        Write-Error "Failed to download kdoctor.exe from GitHub Releases: $_"
        exit 1
    }
}

# 3. Add installDir to User PATH if not present
$userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($userPath -notlike "*$installDir*") {
    Write-Host "Adding $installDir to User PATH..." -ForegroundColor Cyan
    $newPath = "$userPath;$installDir"
    [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
    $env:Path = "$env:Path;$installDir"
}

# 4. Verify installation
Write-Host "`nkdoctor installed successfully!" -ForegroundColor Green
Write-Host "Location: $exePath" -ForegroundColor Gray

try {
    & $exePath --version
} catch {
    Write-Host "Run 'kdoctor' in a new terminal window to start." -ForegroundColor Cyan
}
