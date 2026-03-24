# wstart

[![CI](https://github.com/sverrirab/wsl-host-start/actions/workflows/ci.yml/badge.svg)](https://github.com/sverrirab/wsl-host-start/actions/workflows/ci.yml)

> **Alpha** — This project is under active development. APIs and config formats may change. Use at your own risk.

Launch Windows programs from WSL — the way `start` works on Windows.

## Why

[wslu/wslview was archived](https://github.com/wslutilities/wslu/discussions/329) in March 2025 and Ubuntu is dropping it from future releases. The common `cmd.exe /C start` alias fails on UNC paths, has no elevation support, and requires manual path translation.

**wstart** fills this gap with two small Go binaries that give you full [`ShellExecuteEx`](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecuteexw) access from any WSL shell:

- Open files with their default Windows application
- Launch programs with admin privileges (UAC elevation)
- Use any ShellExecuteEx verb — `open`, `edit`, `print`, `runas`, `explore`, `properties`
- Automatic path translation between WSL and Windows, including **subst and network drive aliases**
- Wait for launched processes and capture exit codes

That last point matters for tools like **Perforce** where the workspace root may live on a `subst`'d drive letter. wstart does longest-prefix alias matching so `p4` sees the drive letter it expects.

## Quick Start

```bash
wstart document.pdf              # Open in default PDF viewer
wstart .                         # Open current directory in Explorer
wstart https://google.com        # Open URL in default browser
wstart -verb runas cmd.exe       # Launch elevated command prompt
wstart -verb print report.docx   # Print a document
wstart -wait installer.exe       # Wait for process to exit
```

## Installation

### From a release (recommended)

1. Download the latest `wstart_*_windows_amd64.zip` from [GitHub Releases](https://github.com/sverrirab/wsl-host-start/releases).

2. Extract the zip and run the installer from **PowerShell**:

   ```powershell
   .\wstart-host.exe --install
   ```

   This will:
   - Copy both binaries to `%LOCALAPPDATA%\wstart\`
   - Create default `config.toml` and `allowlist.toml` (commented out)
   - Generate a signing key and sign the config files
   - Print the WSL setup commands

3. In your **WSL session**, create a symlink (the installer prints the exact command):

   ```bash
   mkdir -p ~/.local/bin
   ln -sf "/mnt/c/Users/<you>/AppData/Local/wstart/wstart" ~/.local/bin/wstart
   ```

4. Ensure `~/.local/bin` is in your PATH. If not, add to `~/.bashrc` or `~/.zshrc`:

   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```

5. Test it:

   ```bash
   wstart .
   ```

### From source

Build on any machine (macOS, Linux, Windows with Go 1.24+):

```bash
git clone https://github.com/sverrirab/wsl-host-start.git
cd wsl-host-start
make build
```

This cross-compiles both `bin/wstart` (linux/amd64) and `bin/wstart-host.exe` (windows/amd64). Then run the installer:

```powershell
.\bin\wstart-host.exe --install
```

### Upgrading

Download the new release zip (or `make build`), then run `--install` again — it will overwrite the binaries and re-sign config files. Your existing config and allowlist are preserved.

### Prerequisites

- WSL (1 or 2) with [interop enabled](https://learn.microsoft.com/en-us/windows/wsl/wsl-config#interop-settings) (the default)
- Go 1.24+ (only needed when building from source)

## Usage

### Flags

```
  -verb string     ShellExecuteEx verb (default "open")
  -dir string      Working directory (WSL or Windows path)
  -wait            Wait for the launched process to exit
  -min             Start minimized
  -max             Start maximized
  -hidden          Start hidden
  -dry-run         Print translated command without executing
  -verbose         Print diagnostic info
  -refresh-drives  Refresh drive cache and exit
  -check-config    Show active configuration diagnostics
  -version         Print version
```

### Host helper flags

The Windows helper (`wstart-host.exe`) has additional management flags:

```
  --install        Install binaries and create default configs
  --check-config   Print configuration diagnostics (config, allowlist, signing, drives)
  --sign-config    Re-sign config files after editing
  --verbose        Show extra detail in check-config output
```

## Configuration

All configuration lives on the Windows host in `%LOCALAPPDATA%\wstart\`. wstart works out of the box for common cases. For advanced setups (subst drives, Perforce, env forwarding), edit `config.toml`.

**Important:** After editing any config file, re-sign it from PowerShell:

```powershell
wstart-host.exe --sign-config
```

Config files are signed with an HMAC key stored in the Windows Registry to prevent tampering from WSL. If signatures are invalid, wstart will refuse to launch programs.

### config.toml

```toml
[drives]
# Manual drive alias overrides (supplements auto-detection)
[drives.aliases]
P = "C:\\dev\\workspace"
Z = "\\\\server\\share"

# Use aliased drive letters when translating paths (default: true)
prefer_aliases = true

# Query the host for subst/network drives automatically (default: true)
auto_detect = true

[env]
# Environment variables to forward to Windows processes
forward = ["P4PORT", "P4CLIENT", "P4USER", "P4CONFIG"]

# Variables that are NEVER forwarded (default includes P4PASSWD, P4TICKETS, P4TRUST)
block = ["P4PASSWD", "P4TICKETS", "P4TRUST"]

[defaults]
verb = "open"
show = "normal"  # normal | min | max | hidden
```

### Drive alias resolution

When `prefer_aliases = true`, wstart applies longest-prefix matching to replace physical paths with aliased drive letters. This is critical for Perforce:

```
WSL cwd:    /mnt/c/dev/workspace/project
Subst:      P: → C:\dev\workspace
Result:     P:\project            ← matches p4 workspace root
```

Run `wstart -refresh-drives` to update the cached drive mappings.

### Diagnostics

Check your active configuration from either side:

```bash
# From WSL
wstart -check-config

# From PowerShell (more detail — includes drives and signing status)
wstart-host.exe --check-config
wstart-host.exe --check-config --verbose
```

## Security

### Allowlist

The Windows helper supports an optional allowlist that restricts which programs and subcommands can be executed. Edit `allowlist.toml` in `%LOCALAPPDATA%\wstart\`:

```toml
# Only these programs can be launched via wstart.
# Delete this file to allow all programs.

[[allow]]
program = "p4"
commands = ["info", "sync", "edit", "submit", "diff", "opened"]

[[allow]]
program = "notepad.exe"

[[allow]]
program = "explorer.exe"

[[allow]]
program = "code"
```

If the file is absent, all programs are allowed. When present, the helper checks each request before executing:

- **Program matching**: case-insensitive, with or without `.exe`, works with full paths
- **Subcommand matching**: finds the first positional argument, skipping flags
- **Denied requests**: return `SE_ERR_ACCESSDENIED` with a descriptive error message

### Deny list (hardcoded)

The following programs are **always blocked** regardless of allowlist configuration, because they are shell/exec bypass vectors:

`cmd`, `powershell`, `pwsh`, `wscript`, `cscript`, `mshta`, `rundll32`, `regsvr32`, `bash`

This deny list is compiled into the binary and cannot be overridden by editing config files.

### Config signing

Config files (`config.toml`, `allowlist.toml`) are protected by HMAC-SHA256 signatures:

- A random signing key is stored in the **Windows Registry** (`HKCU\Software\wstart`), which is not accessible from the WSL filesystem
- Each config file has a companion `.sig` file containing its signature
- The host binary verifies signatures on every launch — tampered files are rejected
- After legitimate edits, re-sign with `wstart-host.exe --sign-config`

This prevents a compromised WSL process from silently modifying the allowlist to grant itself broader access.

## Using Perforce from WSL

With wstart configured, you can run Perforce commands from your WSL shell with correct drive mapping:

```bash
# Sync your workspace (cwd is translated to the subst drive)
wstart -wait p4 sync

# Edit a file
wstart -wait p4 edit //depot/main/src/file.cpp

# Check what files you have open
wstart -wait p4 opened

# Submit a changelist
wstart -wait p4 submit -d "Fix buffer overflow"

# Shell alias for convenience
alias p4='wstart -wait p4'
p4 sync
p4 edit file.cpp
```

The `-wait` flag is important for p4 — it makes wstart block until the command finishes so you see the output and get the correct exit code.

## Architecture

Two cooperating binaries connected via JSON over stdin/stdout:

```
WSL (Linux)                        Windows Host
┌──────────────┐   LaunchRequest   ┌───────────────────┐
│  wstart      │ ────────────────► │  wstart-host.exe  │
│  - parse CLI │   stdin (JSON)    │  - ShellExecuteExW │
│  - translate │ ◄──────────────── │  - drive enumerate │
│    paths     │   stdout (JSON)   │  - exit codes      │
└──────────────┘   LaunchResponse  └───────────────────┘
```

No daemon, no sockets, no PowerShell. The Windows helper calls Win32 APIs directly for speed and full control.

## Development

```bash
make build        # Cross-compile both binaries
make test         # Run tests
make clean        # Remove bin/
```

### Project layout

```
cmd/wstart/          WSL CLI entry point (linux/amd64)
cmd/wstart-host/     Windows helper entry point (windows/amd64)
internal/
  protocol/          Shared JSON request/response types
  allowlist/         Host-side program/subcommand allowlist + deny list
  config/            TOML config loading
  signing/           HMAC-SHA256 config signing (registry key + .sig files)
  install/           Self-installation logic (Windows side)
  pathconv/          Path translation with drive alias resolution
  drivecache/        TTL-based cache of drive enumeration
  interop/           WSL environment detection
  launch/            Orchestration (WSL side)
  drives/            Win32 drive enumeration (Windows side)
  shellexec/         ShellExecuteExW wrapper (Windows side)
```

### Releasing

Releases are built with [GoReleaser](https://goreleaser.com/) via GitHub Actions. To create a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow which cross-compiles both binaries and publishes a zip to GitHub Releases.

### CI

Builds and lints are run via [GitHub Actions](https://github.com/sverrirab/wsl-host-start/actions/workflows/ci.yml). The workflow lints both platforms separately (`GOOS=linux` and `GOOS=windows`), runs tests on platform-independent packages, and cross-compiles both binaries.

## License

MIT
