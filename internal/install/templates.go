//go:build windows

package install

const defaultConfig = `# config.toml — wstart configuration.
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
`

const defaultAllowlist = `# allowlist.toml — Restrict which programs wstart can launch.
#
# When this file contains [[allow]] entries, ONLY listed programs can be
# launched. Remove all [[allow]] entries to allow all programs.
#
# Program matching is case-insensitive, with or without .exe extension.
# For example, "notepad" matches notepad.exe, Notepad.EXE, etc.
#
# After editing, re-sign from an elevated PowerShell:
#   wstart-host.exe --sign-config

# --- Enabled by default ---

[[allow]]
program = "notepad"

[[allow]]
program = "explorer.exe"

# --- Examples (uncomment to enable) ---
#
# [[allow]]
# program = "code"
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
`
