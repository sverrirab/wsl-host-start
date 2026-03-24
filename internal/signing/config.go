//go:build windows

package signing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/config"
)

// configFiles returns the paths to sign/verify in the given directory.
// Only files that actually exist are returned.
func configFiles(dir string) []string {
	var paths []string
	for _, name := range []string{config.ConfigFile, allowlist.AllowlistFile} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}
	return paths
}

// SignAllConfigs signs all existing config files in dir using the registry key.
// Creates the key if it doesn't exist yet.
func SignAllConfigs(dir string) error {
	key, err := EnsureKey()
	if err != nil {
		return err
	}

	files := configFiles(dir)
	if len(files) == 0 {
		return nil
	}

	for _, f := range files {
		if err := SignFile(key, f); err != nil {
			return fmt.Errorf("signing %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// VerifyResult holds the verification status for a single config file.
type VerifyResult struct {
	Path    string
	Exists  bool
	SigErr  error // nil = valid, non-nil = problem
}

// VerifyAllConfigs checks signatures on all config files in dir.
// Returns results for each potential config file (whether it exists or not).
// If no signing key exists in the registry, returns an error.
func VerifyAllConfigs(dir string) ([]VerifyResult, error) {
	key, found, err := LoadKey()
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no signing key found in registry — run wstart-host.exe --sign-config to initialize")
	}

	names := []string{config.ConfigFile, allowlist.AllowlistFile}
	results := make([]VerifyResult, len(names))
	for i, name := range names {
		p := filepath.Join(dir, name)
		results[i].Path = p
		if _, err := os.Stat(p); err != nil {
			results[i].Exists = false
			continue
		}
		results[i].Exists = true
		results[i].SigErr = VerifyFile(key, p)
	}
	return results, nil
}

// VerifyOrErr checks all config files and returns an error if any existing
// file has an invalid or missing signature.
func VerifyOrErr(dir string) error {
	results, err := VerifyAllConfigs(dir)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.Exists && r.SigErr != nil {
			return fmt.Errorf("config signature check failed: %w\nRun wstart-host.exe --sign-config after making legitimate edits", r.SigErr)
		}
	}
	return nil
}
