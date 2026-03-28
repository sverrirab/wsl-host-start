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
`
