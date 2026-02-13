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

## Usage

```bash
wstart document.pdf              # Open in default PDF viewer
wstart .                         # Open current directory in Explorer
wstart https://google.com        # Open URL in default browser
wstart -verb runas cmd.exe       # Launch elevated command prompt
wstart -verb print report.docx   # Print a document
wstart -verb edit config.ini     # Open in registered editor
wstart -wait installer.exe       # Wait for process to exit
wstart -min notepad.exe          # Start minimized
```

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
  -version         Print version
```

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

## Installation

### Prerequisites

- WSL (1 or 2) with [interop enabled](https://learn.microsoft.com/en-us/windows/wsl/wsl-config#interop-settings) (the default)
- Go 1.24+ (for building from source — the installed binaries have no dependencies)

### From prebuilt binaries

Download `wstart` and `wstart-host.exe` from [CI artifacts](https://github.com/sverrirab/wsl-host-start/actions/workflows/ci.yml), then:

**Step 1 — Windows host** (PowerShell):

```powershell
.\install-host.ps1
```

Installs `wstart-host.exe` to `%LOCALAPPDATA%\wstart\` and creates commented-out example `config.toml` and `allowlist.toml` files that you can customize.

**Step 2 — WSL** (bash):

```bash
./install-wsl.sh
```

Installs `wstart` to `~/.local/bin/` and verifies the host-side installation is in place.

### From source

Build on any machine (macOS, Linux, Windows with Go):

```bash
git clone https://github.com/sverrirab/wsl-host-start.git
cd wsl-host-start
make build
```

This cross-compiles both `bin/wstart` (linux/amd64) and `bin/wstart-host.exe` (windows/amd64).

Then run the install scripts as described above, or use `make install` from WSL to do both steps at once.

## Configuration

All configuration lives on the Windows host in `%LOCALAPPDATA%\wstart\` (created by `install-host.ps1`). wstart works out of the box for common cases. For advanced setups (subst drives, Perforce, env forwarding), edit `config.toml`:

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

### Allowlist (host-side security)

The Windows helper supports an optional allowlist that restricts which programs and subcommands can be executed. Create `allowlist.toml` in the same directory as `wstart-host.exe` (typically `%LOCALAPPDATA%\wstart\`):

```toml
# allowlist.toml — only these programs can be launched via wstart.
# Delete this file to allow all programs.

# Perforce — read-only and standard workflow commands only
[[allow]]
program = "p4"
commands = [
    # Information
    "info", "where", "have", "opened", "changes", "describe",
    "filelog", "annotate", "print", "fstat", "depots", "dirs",
    "files", "sizes", "users", "clients", "branches", "labels",
    # Diff
    "diff", "diff2",
    # Workspace sync & resolve
    "sync", "resolve", "resolved",
    # File editing workflow
    "edit", "add", "delete", "revert", "move", "copy", "rename",
    "lock", "unlock",
    # Changelist management
    "change", "submit", "shelve", "unshelve",
    # Login
    "login", "logout", "set",
]

# General tools
[[allow]]
program = "notepad.exe"

[[allow]]
program = "explorer.exe"

[[allow]]
program = "code"
```

If the file is absent, all programs are allowed. When present, the helper checks each request before executing:

- **Program matching**: case-insensitive, with or without `.exe`, works with full paths (`C:\Program Files\Perforce\p4.exe` matches `p4`)
- **Subcommand matching**: finds the first positional argument, skipping flags — so `p4 -c myclient edit file.txt` correctly matches `edit`
- **No commands list**: any arguments are allowed (e.g., `notepad.exe`)
- **Denied requests**: return `SE_ERR_ACCESSDENIED` with a descriptive error message

### Using Perforce from WSL

With wstart configured, you can run Perforce commands from your WSL shell with correct drive mapping:

```bash
# Sync your workspace (cwd is translated to the subst drive)
wstart -wait p4 sync

# Edit a file — p4 sees the correct workspace-relative path
wstart -wait p4 edit //depot/main/src/file.cpp

# Check what files you have open
wstart -wait p4 opened

# Diff against depot
wstart -wait p4 diff ./src/file.cpp

# Submit a changelist
wstart -wait p4 submit -d "Fix buffer overflow"

# View workspace info
wstart -wait p4 info

# Shell alias for convenience
alias p4='wstart -wait p4'
p4 sync
p4 edit file.cpp
p4 submit -d "my change"
```

The `-wait` flag is important for p4 — it makes wstart block until the command finishes so you see the output and get the correct exit code.

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
  allowlist/         Host-side program/subcommand allowlist
  config/            TOML config loading
  pathconv/          Path translation with drive alias resolution
  drivecache/        TTL-based cache of drive enumeration
  interop/           WSL environment detection
  launch/            Orchestration (WSL side)
  drives/            Win32 drive enumeration (Windows side)
  shellexec/         ShellExecuteExW wrapper (Windows side)
```

### CI

Builds and lints are run via [GitHub Actions](https://github.com/sverrirab/wsl-host-start/actions/workflows/ci.yml). The workflow lints both platforms separately (`GOOS=linux` and `GOOS=windows`), runs tests on platform-independent packages, and cross-compiles both binaries.

## License

MIT
