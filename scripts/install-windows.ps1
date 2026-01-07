#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Vectra Guard - Windows Installer

.DESCRIPTION
  Installs the Windows Vectra Guard binary and adds it to the current user's PATH.

  This script:
    - Downloads the latest vectra-guard Windows binary
    - Installs it to "$env:ProgramFiles\VectraGuard"
    - Adds that directory to the User PATH in the registry
    - Updates PATH for the current PowerShell session

.NOTES
  - Intended to be invoked via:
      Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force
      irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Write-Host "🛡️  Vectra Guard - Windows Installer" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""

if ($IsLinux -or $IsMacOS) {
    Write-Host "❌ This installer is intended for Windows PowerShell only." -ForegroundColor Red
    exit 1
}

$installRoot = $env:ProgramFiles
if (-not $installRoot) {
    $installRoot = "C:\Program Files"
}

$installDir = Join-Path $installRoot "VectraGuard"
$binaryPath = Join-Path $installDir "vectra-guard.exe"

Write-Host "📦 Installing to: $installDir" -ForegroundColor Yellow

if (-not (Test-Path -LiteralPath $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

$releaseUrl = "https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-windows-amd64.exe"

Write-Host "⬇️  Downloading latest Vectra Guard binary..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $releaseUrl -OutFile $binaryPath -UseBasicParsing

Write-Host "✅ Downloaded to: $binaryPath" -ForegroundColor Green

# Ensure executable bit is set (primarily for completeness)
try {
    & icacls $binaryPath /grant "*S-1-1-0:RX" | Out-Null
} catch {
    # Non-fatal; continue
}

# Update User PATH (registry) so future shells can find vectra-guard
$regPath = "HKCU:\Environment"
try {
    $currentPath = (Get-ItemProperty -Path $regPath -Name Path -ErrorAction SilentlyContinue).Path
} catch {
    $currentPath = $null
}

if ($currentPath) {
    if ($currentPath -notlike "*$installDir*") {
        $newPath = "$currentPath;$installDir"
    } else {
        $newPath = $currentPath
    }
} else {
    $newPath = $installDir
}

Set-ItemProperty -Path $regPath -Name Path -Value $newPath
[Environment]::SetEnvironmentVariable("Path", $newPath, "User")

# Update PATH for current session
if ($env:PATH -notlike "*$installDir*") {
    $env:PATH = "$installDir;$env:PATH"
}

Write-Host "✅ Updated User PATH to include: $installDir" -ForegroundColor Green
Write-Host ""

Write-Host "🔍 Verifying installation..." -ForegroundColor Cyan

try {
    $version = vectra-guard version 2>$null
    if ($LASTEXITCODE -eq 0 -and $version) {
        Write-Host "✅ vectra-guard is installed and available on PATH." -ForegroundColor Green
        Write-Host "   Version: $version"
    } else {
        Write-Host "⚠️  vectra-guard installed, but could not verify via PATH." -ForegroundColor Yellow
        Write-Host "   You may need to restart PowerShell or sign out/in for PATH changes to apply."
    }
} catch {
    Write-Host "⚠️  vectra-guard installed, but could not verify via PATH." -ForegroundColor Yellow
    Write-Host "   You may need to restart PowerShell or sign out/in for PATH changes to apply."
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Optionally enable PowerShell protection:" -ForegroundColor Cyan
Write-Host "     irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-powershell-protection.ps1 | iex" -ForegroundColor Yellow
Write-Host "  2. Start using vectra-guard:" -ForegroundColor Cyan
Write-Host "     vectra-guard --help" -ForegroundColor Yellow

