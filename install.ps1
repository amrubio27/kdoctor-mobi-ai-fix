# install.ps1 - Installer for kdoctor on Windows
#
# Usage:
#   irm https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.ps1 | iex
#
# Pin a version:
#   $env:KDOCTOR_VERSION = 'v0.7.0'; irm ... | iex

$ErrorActionPreference = 'Stop'

$repo = 'amrubio27/kdoctor-mobi-ai-fix'

Write-Host "Installing kdoctor for Windows..." -ForegroundColor Cyan

# 1. Destination under LOCALAPPDATA (no admin rights needed).
$installDir = "$env:LOCALAPPDATA\kdoctor\bin"
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

# 2. Resolve the release to fetch.
if ($env:KDOCTOR_VERSION) {
    $releasePath = "download/$($env:KDOCTOR_VERSION)"
    Write-Host "Version pinned to $($env:KDOCTOR_VERSION)" -ForegroundColor Gray
} else {
    $releasePath = 'latest/download'
}

# 3. Download. kdoctor is required; kdoctor-mcp is best-effort because older
#    releases did not publish it, and its absence must not fail the install.
$targets = @(
    @{ Asset = 'kdoctor-windows-amd64.exe';     Out = 'kdoctor.exe';     Required = $true  },
    @{ Asset = 'kdoctor-mcp-windows-amd64.exe'; Out = 'kdoctor-mcp.exe'; Required = $false }
)

foreach ($t in $targets) {
    $url = "https://github.com/$repo/releases/$releasePath/$($t.Asset)"
    $dest = Join-Path $installDir $t.Out
    Write-Host "Downloading $($t.Asset)..." -ForegroundColor Yellow
    try {
        Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
    } catch {
        if ($t.Required) {
            # The previous version fell back to "$PSScriptRoot\kdoctor.exe",
            # which is empty when this script runs through `irm | iex`. That
            # turned every download failure into a confusing "checking local
            # repository..." message instead of naming the URL that failed.
            Write-Host ""
            Write-Error @"
Could not download kdoctor from:
  $url

$($_.Exception.Message)

Check the URL in a browser: if it 404s, that asset is missing from the release.
You can also build from source:
  git clone https://github.com/$repo.git
  cd kdoctor-mobi-ai-fix
  go build -o kdoctor.exe ./cmd/kdoctor
"@
            exit 1
        }
        Write-Host "  skipped (not published in this release)" -ForegroundColor DarkGray
    }
}

$exePath = Join-Path $installDir 'kdoctor.exe'

# 4. Put it on PATH for future terminals, and this one.
$userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($userPath -notlike "*$installDir*") {
    Write-Host "Adding $installDir to User PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", [EnvironmentVariableTarget]::User)
    $env:Path = "$env:Path;$installDir"
}

Write-Host "`nkdoctor installed." -ForegroundColor Green
Write-Host "Location: $installDir" -ForegroundColor Gray

try { & $exePath --version } catch { }

Write-Host ""
Write-Host "Note: the first scan downloads detekt (~50 MB) into ~/.kdoctor/tools" -ForegroundColor Gray
Write-Host "and needs a JDK 11-21. Run 'kdoctor doctor' to check your setup." -ForegroundColor Gray
