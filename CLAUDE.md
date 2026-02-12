# wsl-host-start

Two cooperating Go binaries: `wstart` (WSL/Linux CLI) and `wstart-host.exe` (Windows helper).

## Build

```bash
make build        # Both binaries
make build-wsl    # Linux binary only
make build-host   # Windows binary only
make test         # Run tests
make clean        # Remove bin/
```

## Architecture

- `cmd/wstart/` — WSL CLI entry point (GOOS=linux)
- `cmd/wstart-host/` — Windows helper entry point (GOOS=windows)
- `internal/protocol/` — Shared JSON request/response types
- `internal/config/` — TOML config loading (WSL side)
- `internal/pathconv/` — Path translation with drive alias resolution (WSL side)
- `internal/drivecache/` — Drive cache management (WSL side)
- `internal/interop/` — WSL environment detection (WSL side)
- `internal/launch/` — Orchestration (WSL side)
- `internal/drives/` — Win32 drive enumeration (Windows side)
- `internal/shellexec/` — ShellExecuteExW wrapper (Windows side)

## Conventions

- Standard `flag` package for CLI parsing (no cobra)
- `golang.org/x/sys/windows` for Win32 API calls
- `github.com/BurntSushi/toml` for config
- JSON over stdin/stdout for IPC between WSL CLI and Windows helper
- No fallback mode — host helper must be installed
