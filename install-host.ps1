#Requires -Version 5.1
<#
.SYNOPSIS
    Installs wstart-host.exe and creates default configuration files.

.DESCRIPTION
    This script installs the wstart Windows host helper to %LOCALAPPDATA%\wstart\.
    It creates commented-out example files for the allowlist (which programs can
    be launched) and the config (drive mappings, env forwarding, defaults).

    Run this from the project root or pass -BinPath to the binary.

.EXAMPLE
    .\install-host.ps1
    .\install-host.ps1 -BinPath .\bin\wstart-host.exe
#>
param(
    [string]$BinPath = ".\bin\wstart-host.exe"
)

$ErrorActionPreference = "Stop"

$installDir = Join-Path $env:LOCALAPPDATA "wstart"

# --- Locate wstart-host.exe ---
if (-not (Test-Path $BinPath)) {
    # Try alongside this script
    $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
    $altPath = Join-Path $scriptDir "bin\wstart-host.exe"
    if (Test-Path $altPath) {
        $BinPath = $altPath
    } else {
        Write-Error "Cannot find wstart-host.exe at '$BinPath'. Build it first with 'make build' or pass -BinPath."
        exit 1
    }
}

# --- Install wstart-host.exe ---
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

Copy-Item -Path $BinPath -Destination (Join-Path $installDir "wstart-host.exe") -Force
Write-Host "Installed wstart-host.exe to $installDir\" -ForegroundColor Green

# --- Create example config.toml (if not already present) ---
$configPath = Join-Path $installDir "config.toml"
if (-not (Test-Path $configPath)) {
    $configContent = @'
# config.toml — wstart configuration.
# All configuration lives here on the Windows host.
# Uncomment and edit the sections below to customize behavior.
#
# [drives]
# # Use aliased drive letters when translating paths (default: true)
# prefer_aliases = true
#
# # Query the host for subst/network drives automatically (default: true)
# auto_detect = true
#
# # Manual drive alias overrides (supplements auto-detection)
# [drives.aliases]
# P = "C:\\dev\\workspace"
# Z = "\\\\server\\share"
#
# [env]
# # Environment variables to forward to Windows processes
# forward = ["P4PORT", "P4CLIENT", "P4USER", "P4CONFIG"]
#
# # Variables that are NEVER forwarded (defaults shown)
# block = ["P4PASSWD", "P4TICKETS", "P4TRUST"]
#
# [defaults]
# verb = "open"
# show = "normal"  # normal | min | max | hidden
'@
    Set-Content -Path $configPath -Value $configContent -Encoding UTF8
    Write-Host "Created example config at $configPath" -ForegroundColor Green
} else {
    Write-Host "Config already exists at $configPath (skipped)" -ForegroundColor Yellow
}

# --- Create example allowlist.toml (if not already present) ---
$allowlistPath = Join-Path $installDir "allowlist.toml"
if (-not (Test-Path $allowlistPath)) {
    $allowlistContent = @'
# allowlist.toml — Restrict which programs wstart can launch.
# Delete this file (or leave it absent) to allow all programs.
# Uncomment and edit the sections below to enable the allowlist.
#
# [[allow]]
# program = "p4"
# commands = [
#     # Information
#     "info", "where", "have", "opened", "changes", "describe",
#     "filelog", "annotate", "print", "fstat", "depots", "dirs",
#     "files", "sizes", "users", "clients", "branches", "labels",
#     # Diff
#     "diff", "diff2",
#     # Workspace sync & resolve
#     "sync", "resolve", "resolved",
#     # File editing workflow
#     "edit", "add", "delete", "revert", "move", "copy", "rename",
#     "lock", "unlock",
#     # Changelist management
#     "change", "submit", "shelve", "unshelve",
#     # Login
#     "login", "logout", "set",
# ]
#
# [[allow]]
# program = "notepad.exe"
#
# [[allow]]
# program = "explorer.exe"
#
# [[allow]]
# program = "code"
'@
    Set-Content -Path $allowlistPath -Value $allowlistContent -Encoding UTF8
    Write-Host "Created example allowlist at $allowlistPath" -ForegroundColor Green
} else {
    Write-Host "Allowlist already exists at $allowlistPath (skipped)" -ForegroundColor Yellow
}

# --- Summary ---
Write-Host ""
Write-Host "Host installation complete!" -ForegroundColor Cyan
Write-Host ""
Write-Host "Configuration files:" -ForegroundColor Cyan
Write-Host "  Config:    $configPath" -ForegroundColor White
Write-Host "  Allowlist: $allowlistPath" -ForegroundColor White
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Edit the config and allowlist files above as needed"
Write-Host ""
Write-Host "  2. Install the WSL-side binary by running inside WSL:"
Write-Host "     ./install-wsl.sh" -ForegroundColor White
Write-Host ""
Write-Host "  3. Test it from WSL:"
Write-Host "     wstart ." -ForegroundColor White
