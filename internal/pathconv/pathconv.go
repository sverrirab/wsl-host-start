// Package pathconv translates paths between WSL and Windows formats,
// applying drive alias resolution for subst/network drives.
package pathconv

import (
	"path/filepath"
	"strings"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

// Alias maps a Windows drive letter to its target path.
type Alias struct {
	Letter string // e.g. "P"
	Target string // e.g. "C:\\dev\\workspace"
}

// Converter translates WSL paths to Windows paths with alias awareness.
type Converter struct {
	aliases       []Alias
	preferAliases bool
}

// NewConverter creates a path converter with the given drive aliases.
func NewConverter(drives []protocol.DriveInfo, configAliases map[string]string, preferAliases bool) *Converter {
	c := &Converter{preferAliases: preferAliases}

	// Build alias list from auto-detected drives (subst drives only).
	for _, d := range drives {
		if d.Type == protocol.DriveSubst && d.Target != "" {
			c.aliases = append(c.aliases, Alias{
				Letter: d.Letter,
				Target: d.Target,
			})
		}
	}

	// Config overrides take priority — add or replace.
	for letter, target := range configAliases {
		letter = strings.ToUpper(letter)
		found := false
		for i, a := range c.aliases {
			if strings.EqualFold(a.Letter, letter) {
				c.aliases[i].Target = target
				found = true
				break
			}
		}
		if !found {
			c.aliases = append(c.aliases, Alias{Letter: letter, Target: target})
		}
	}

	return c
}

// IsBareCommand returns true if the input looks like a bare command name
// (e.g. "p4", "notepad.exe") rather than a path or URL. Bare commands
// contain no path separators and don't start with "." — they should be
// passed through to Windows for PATH + PATHEXT resolution.
func IsBareCommand(s string) bool {
	if strings.ContainsAny(s, `/\`) {
		return false
	}
	if strings.HasPrefix(s, ".") {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	return true
}

// ToWindows converts a WSL path to a Windows path.
// It resolves relative paths, calls wslpath, and applies alias mapping.
// URLs (http://, https://) and bare command names are returned unchanged.
func (c *Converter) ToWindows(wslPath string) (string, error) {
	// Pass URLs through unchanged.
	lower := strings.ToLower(wslPath)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return wslPath, nil
	}

	// Bare command names (no path separators, not starting with ".") pass
	// through unchanged — Windows resolves them via PATH + PATHEXT.
	if IsBareCommand(wslPath) {
		return wslPath, nil
	}

	// Resolve relative paths to absolute.
	if !filepath.IsAbs(wslPath) {
		abs, err := filepath.Abs(wslPath)
		if err != nil {
			return "", err
		}
		wslPath = abs
	}

	// Translate using wslpath.
	winPath, err := runWslpath(wslPath)
	if err != nil {
		return "", err
	}

	// Apply alias mapping if enabled.
	if c.preferAliases {
		winPath = c.applyAlias(winPath)
	}

	return winPath, nil
}

// applyAlias performs longest-prefix matching to replace a physical path
// with an aliased drive letter.
// Example: C:\dev\workspace\project with alias P:→C:\dev\workspace becomes P:\project
func (c *Converter) applyAlias(winPath string) string {
	bestLen := 0
	bestLetter := ""
	bestTarget := ""

	normalized := strings.ToLower(strings.ReplaceAll(winPath, "/", `\`))

	for _, a := range c.aliases {
		target := strings.ToLower(strings.ReplaceAll(a.Target, "/", `\`))
		// Ensure target ends without trailing backslash for consistent matching.
		target = strings.TrimRight(target, `\`)

		if !strings.HasPrefix(normalized, target) {
			continue
		}
		// Ensure we match at a path boundary: either exact match or followed by \.
		rest := normalized[len(target):]
		if rest != "" && !strings.HasPrefix(rest, `\`) {
			continue
		}
		if len(target) > bestLen {
			bestLen = len(target)
			bestLetter = a.Letter
			bestTarget = target
		}
	}

	if bestLen == 0 {
		return winPath
	}

	// Replace the matched prefix with the drive letter.
	// Normalize forward slashes to backslashes in the suffix.
	suffix := strings.ReplaceAll(winPath[len(bestTarget):], "/", `\`)
	return strings.ToUpper(bestLetter) + ":" + suffix
}
