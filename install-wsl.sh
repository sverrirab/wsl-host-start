#!/bin/bash
# install-wsl.sh — Install the wstart WSL CLI binary.
#
# Run this inside WSL. It copies wstart to ~/.local/bin/ and verifies
# that the host-side installation (wstart-host.exe) is already in place.
#
# Usage:
#   ./install-wsl.sh
#   ./install-wsl.sh path/to/wstart

set -euo pipefail

BIN="${1:-./bin/wstart}"

if [ ! -f "$BIN" ]; then
    echo "Error: Cannot find wstart at '$BIN'."
    echo "Build it first with 'make build' or pass the path as an argument."
    exit 1
fi

# --- Install wstart binary ---
mkdir -p ~/.local/bin
cp "$BIN" ~/.local/bin/wstart
chmod +x ~/.local/bin/wstart
echo "Installed wstart to ~/.local/bin/wstart"

# --- Check if host-side install has been done ---
HOST_OK=false
WINAPPDATA=$(cmd.exe /C 'echo %LOCALAPPDATA%' 2>/dev/null | tr -d '\r') || true
if [ -n "$WINAPPDATA" ] && [ "$WINAPPDATA" != "%LOCALAPPDATA%" ]; then
    WSLPATH=$(wslpath -u "$WINAPPDATA" 2>/dev/null) || true
    if [ -n "$WSLPATH" ] && [ -f "$WSLPATH/wstart/wstart-host.exe" ]; then
        HOST_OK=true
        echo "Verified wstart-host.exe at $WINAPPDATA\\wstart\\"
    fi
fi

# --- Summary ---
echo ""

if [ "$HOST_OK" = false ]; then
    echo "WARNING: wstart-host.exe not found on the Windows host."
    echo "Run the host install script from PowerShell first:"
    echo "  .\\install-host.ps1"
    echo ""
fi

# Check if ~/.local/bin is in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$HOME/.local/bin"; then
    echo "WARNING: ~/.local/bin is not in your PATH."
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo '  export PATH="$HOME/.local/bin:$PATH"'
    echo ""
fi

if [ "$HOST_OK" = true ]; then
    echo "Installation complete! Test it with:"
    echo "  wstart ."
fi
