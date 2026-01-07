#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Vectra Guard - PowerShell Protection Installer (Windows Native)

.DESCRIPTION
  Adds Vectra Guard integration to the current user's PowerShell profile so that:
  - A Vectra Guard session is started automatically
  - Commands are logged to Vectra Guard sessions
  - A convenient `vg` alias is available

  This is the Windows-native equivalent of the universal shell protection for bash/zsh/fish.

.NOTES
  - Works on Windows PowerShell and PowerShell 7+
  - Installs for the current user / current host only (uses $PROFILE)
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Write-Host "🛡️  Vectra Guard - PowerShell Protection Installer" -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan
Write-Host ""

function Test-CommandExists {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Name
    )

    return [bool](Get-Command -Name $Name -ErrorAction SilentlyContinue)
}

# 1. Verify vectra-guard is installed
if (-not (Test-CommandExists -Name 'vectra-guard')) {
    Write-Host "❌ vectra-guard not found in PATH" -ForegroundColor Red
    Write-Host "   Install the Windows binary first, then re-run this script." -ForegroundColor Yellow
    Write-Host "   Latest release: https://github.com/xadnavyaai/vectra-guard/releases/latest"
    exit 1
}

Write-Host "✅ vectra-guard found in PATH" -ForegroundColor Green
Write-Host ""

# 2. Determine profile path
$profilePath = $PROFILE
$profileDir  = Split-Path -Parent $profilePath

Write-Host "Using PowerShell profile:" -NoNewline
Write-Host " $profilePath" -ForegroundColor Yellow

if (-not (Test-Path -LiteralPath $profileDir)) {
    Write-Host "Creating profile directory: $profileDir"
    New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
}

if (-not (Test-Path -LiteralPath $profilePath)) {
    Write-Host "Creating new profile file..." -ForegroundColor Yellow
    New-Item -ItemType File -Path $profilePath -Force | Out-Null
}

# 3. Backup existing profile
$backupPath = "$profilePath.vectra-backup"
Copy-Item -LiteralPath $profilePath -Destination $backupPath -Force
Write-Host "✅ Backed up profile to: $backupPath" -ForegroundColor Green
Write-Host ""

# 4. Append Vectra Guard integration block
$integrationBlock = @'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if (Get-Command vectra-guard -ErrorAction SilentlyContinue) {
    function Initialize-VectraGuardSession {
        if (-not $env:VECTRAGUARD_SESSION_ID) {
            $sessionFile = Join-Path $HOME ".vectra-guard-session"

            if (Test-Path -LiteralPath $sessionFile) {
                try {
                    $existing = Get-Content -LiteralPath $sessionFile -ErrorAction Stop | Select-Object -Last 1
                    if ($existing) {
                        # Verify session is still valid
                        if (vectra-guard session show $existing *> $null) {
                            $env:VECTRAGUARD_SESSION_ID = $existing
                        }
                    }
                } catch {
                    # Ignore errors and fall through to creating a new session
                }
            }

            if (-not $env:VECTRAGUARD_SESSION_ID) {
                try {
                    $session = vectra-guard session start --agent "$env:USERNAME-pwsh" --workspace $HOME 2>$null | Select-Object -Last 1
                    if ($session) {
                        $env:VECTRAGUARD_SESSION_ID = $session
                        $session | Out-File -FilePath $sessionFile -Encoding utf8
                    }
                } catch {
                    # If session start fails, do not block shell startup
                }
            }
        }
    }

    function Write-VectraGuardCommandLog {
        # Log the most recent command to the active session
        if (-not $env:VECTRAGUARD_SESSION_ID) {
            return
        }

        try {
            $last = Get-History -Count 1 -ErrorAction SilentlyContinue
            if (-not $last -or -not $last.CommandLine) {
                return
            }

            # Avoid logging the same command multiple times in a row
            if ($script:VectraGuardLastHistoryId -eq $last.Id) {
                return
            }

            $script:VectraGuardLastHistoryId = $last.Id

            vectra-guard exec --session $env:VECTRAGUARD_SESSION_ID -- echo "logged: $($last.CommandLine)" *> $null
        } catch {
            # Never break the prompt on logging errors
        }
    }

    # Preserve any existing prompt function
    if (-not $global:VectraGuard_OriginalPromptSaved) {
        $global:VectraGuard_OriginalPromptSaved = $true
        $global:VectraGuard_OriginalPrompt = $function:prompt
    }

    function global:prompt {
        Initialize-VectraGuardSession
        Write-VectraGuardCommandLog

        if ($global:VectraGuard_OriginalPrompt) {
            & $global:VectraGuard_OriginalPrompt
        } else {
            "PS $($executionContext.SessionState.Path.CurrentLocation)> "
        }
    }

    # Convenience alias
    Set-Alias -Name vg -Value vectra-guard -Scope Global -ErrorAction SilentlyContinue
}

# End Vectra Guard Integration
# ============================================================================

'@

Add-Content -LiteralPath $profilePath -Value $integrationBlock

Write-Host "✅ Vectra Guard PowerShell integration installed" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Close and reopen PowerShell, OR run:" -NoNewline
Write-Host " `. $PROFILE`" -ForegroundColor Yellow
Write-Host "  2. Run:" -NoNewline
Write-Host "  `echo \$env:VECTRAGUARD_SESSION_ID`" -ForegroundColor Yellow
Write-Host "     to confirm a session is active."
Write-Host "  3. Use the 'vg' alias for convenience, e.g.:" -ForegroundColor Cyan
Write-Host "     vg exec -- npm install" -ForegroundColor Yellow

