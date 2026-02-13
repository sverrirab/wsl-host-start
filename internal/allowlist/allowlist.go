// Package allowlist enforces a host-side list of allowed programs and subcommands.
// The allowlist file lives next to wstart-host.exe and is only loaded by the
// Windows helper — the WSL side cannot bypass it.
//
// If the allowlist file does not exist, all programs are allowed.
// If the file exists but is empty, nothing is allowed.
package allowlist

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// AllowlistFile is the expected filename in the helper's directory.
const AllowlistFile = "allowlist.toml"

// Rule defines a single allowed program and optionally its permitted subcommands.
type Rule struct {
	// Program name to match (e.g. "p4", "notepad.exe", "explorer.exe").
	// Matched case-insensitively against the base name of the resolved file,
	// with or without .exe extension.
	Program string `toml:"program"`

	// If set, only these subcommands are allowed. The first positional
	// argument (skipping flags that start with "-") is checked against
	// this list. If empty, any arguments are allowed.
	Commands []string `toml:"commands,omitempty"`
}

// List holds parsed allowlist rules.
type List struct {
	Allow []Rule `toml:"allow"`
}

// LoadResult describes how the allowlist was loaded.
type LoadResult struct {
	// Loaded is true if an allowlist file was found and parsed.
	Loaded bool
	// Path is the allowlist file path that was checked.
	Path string
	// List contains the rules (nil if not loaded).
	List *List
}

// Load reads the allowlist from the given directory (typically the
// directory containing wstart-host.exe). Returns a LoadResult indicating
// whether an allowlist was found.
func Load(dir string) (*LoadResult, error) {
	path := filepath.Join(dir, AllowlistFile)
	result := &LoadResult{Path: path}

	var list List
	if _, err := toml.DecodeFile(path, &list); err != nil {
		// File doesn't exist → no allowlist → everything allowed.
		return result, nil
	}

	result.Loaded = true
	result.List = &list
	return result, nil
}

// Check verifies that the given file and args are permitted by the allowlist.
// Returns nil if allowed, or an error describing why the request was denied.
//
// If no allowlist was loaded (lr.Loaded == false), everything is allowed.
func (lr *LoadResult) Check(file string, args []string) error {
	if !lr.Loaded {
		return nil
	}

	baseName := normalizeProgram(file)

	for _, rule := range lr.List.Allow {
		if !matchProgram(baseName, rule.Program) {
			continue
		}

		// Program matches. Check subcommand restriction.
		if len(rule.Commands) == 0 {
			return nil // No subcommand restriction — allow all.
		}

		subcmd := firstPositionalArg(args)
		if subcmd == "" {
			return fmt.Errorf("denied: %q requires a subcommand (allowed: %s)",
				rule.Program, strings.Join(rule.Commands, ", "))
		}

		for _, allowed := range rule.Commands {
			if strings.EqualFold(subcmd, allowed) {
				return nil
			}
		}

		return fmt.Errorf("denied: %q subcommand %q is not allowed (allowed: %s)",
			rule.Program, subcmd, strings.Join(rule.Commands, ", "))
	}

	return fmt.Errorf("denied: program %q is not in the allowlist (%s)",
		baseName, lr.Path)
}

// normalizeProgram extracts the base filename, lowercased, without .exe extension.
// Handles both forward and backslash separators regardless of the host OS.
func normalizeProgram(file string) string {
	// Handle both Unix and Windows path separators.
	if i := strings.LastIndexAny(file, `/\`); i >= 0 {
		file = file[i+1:]
	}
	base := strings.ToLower(file)
	base = strings.TrimSuffix(base, ".exe")
	return base
}

// matchProgram checks if a normalized base name matches a rule's program field.
func matchProgram(baseName, ruleProgram string) bool {
	rule := strings.ToLower(ruleProgram)
	rule = strings.TrimSuffix(rule, ".exe")
	return baseName == rule
}

// firstPositionalArg returns the first argument that doesn't start with "-".
// This skips flags like "-c", "--client" to find the subcommand.
func firstPositionalArg(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
		// Skip flag values: if a flag doesn't contain "=" it likely
		// consumes the next argument as its value (e.g. "-c myclient").
		if !strings.Contains(arg, "=") && i+1 < len(args) {
			i++ // skip the flag's value
		}
	}
	return ""
}
