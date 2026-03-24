//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/config"
	"github.com/sverrirab/wsl-host-start/internal/drives"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
	"github.com/sverrirab/wsl-host-start/internal/signing"
)

func runCheckConfig(verbose bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	dir := filepath.Dir(exePath)

	printHostCheckConfig(os.Stdout, dir, verbose)
	return nil
}

func printHostCheckConfig(w io.Writer, dir string, verbose bool) {
	fmt.Fprintf(w, "Host:      %s\n", filepath.Join(dir, "wstart-host.exe"))
	fmt.Fprintf(w, "Version:   %s\n", version)

	// Config
	configPath := filepath.Join(dir, config.ConfigFile)
	cfg, cfgErr := config.Load(dir)
	if cfgErr != nil {
		fmt.Fprintf(w, "Config:    %s (error: %v)\n", configPath, cfgErr)
	} else if _, err := os.Stat(configPath); err != nil {
		fmt.Fprintf(w, "Config:    %s (not found — using defaults)\n", configPath)
	} else {
		fmt.Fprintf(w, "Config:    %s\n", configPath)
	}

	// Deny list
	fmt.Fprintf(w, "\n--- Deny List ---\n")
	fmt.Fprintf(w, "Blocked:   %s\n", strings.Join(allowlist.DeniedPrograms(), ", "))

	// Allowlist
	fmt.Fprintf(w, "\n--- Allowlist ---\n")
	al, alErr := allowlist.Load(dir)
	if alErr != nil {
		fmt.Fprintf(w, "File:      %s\n", filepath.Join(dir, allowlist.AllowlistFile))
		fmt.Fprintf(w, "Status:    ERROR (%v)\n", alErr)
	} else {
		fmt.Fprintf(w, "File:      %s\n", al.Path)
		if !al.Loaded {
			fmt.Fprintf(w, "Status:    NOT ACTIVE (file not found — all programs allowed)\n")
		} else if len(al.List.Allow) == 0 {
			fmt.Fprintf(w, "Status:    ACTIVE (empty — all programs DENIED)\n")
		} else {
			fmt.Fprintf(w, "Status:    ACTIVE (%d rules)\n", len(al.List.Allow))
			for _, rule := range al.List.Allow {
				if len(rule.Commands) == 0 {
					fmt.Fprintf(w, "  allow:   %s (any args)\n", rule.Program)
				} else {
					fmt.Fprintf(w, "  allow:   %s [%s]\n", rule.Program, strings.Join(rule.Commands, ", "))
				}
				// Warn if this rule targets a denied program.
				if allowlist.CheckDenyList(rule.Program) != nil {
					fmt.Fprintf(w, "           ^ WARNING: this program is on the deny list and will always be blocked\n")
				}
			}
		}
	}

	// Config signing
	fmt.Fprintf(w, "\n--- Config Signing ---\n")
	_, keyFound, keyErr := signing.LoadKey()
	if keyErr != nil {
		fmt.Fprintf(w, "Key:       ERROR (%v)\n", keyErr)
	} else if !keyFound {
		fmt.Fprintf(w, "Key:       NOT SET (run --sign-config to initialize)\n")
	} else {
		fmt.Fprintf(w, "Key:       present (HKCU\\Software\\wstart)\n")
		results, verErr := signing.VerifyAllConfigs(dir)
		if verErr != nil {
			fmt.Fprintf(w, "Status:    ERROR (%v)\n", verErr)
		} else {
			for _, r := range results {
				name := filepath.Base(r.Path)
				if !r.Exists {
					fmt.Fprintf(w, "  %-20s (not present)\n", name+":")
				} else if r.SigErr == nil {
					fmt.Fprintf(w, "  %-20s OK\n", name+":")
				} else {
					fmt.Fprintf(w, "  %-20s FAILED (%v)\n", name+":", r.SigErr)
				}
			}
		}
	}

	// Environment forwarding config
	if cfg != nil {
		fmt.Fprintf(w, "\n--- Environment ---\n")
		if len(cfg.Env.Forward) == 0 {
			fmt.Fprintf(w, "Forward:   (none)\n")
		} else {
			fmt.Fprintf(w, "Forward:   %s\n", strings.Join(cfg.Env.Forward, ", "))
		}
		if len(cfg.Env.Block) == 0 {
			fmt.Fprintf(w, "Block:     (none)\n")
		} else {
			fmt.Fprintf(w, "Block:     %s\n", strings.Join(cfg.Env.Block, ", "))
		}
	}

	// Drives
	fmt.Fprintf(w, "\n--- Drives ---\n")
	if cfg != nil && verbose {
		fmt.Fprintf(w, "Auto-detect: %v\n", cfg.Drives.AutoDetect)
		fmt.Fprintf(w, "Prefer aliases: %v\n", cfg.Drives.PreferAliases)
		if len(cfg.Drives.Aliases) > 0 {
			for letter, target := range cfg.Drives.Aliases {
				fmt.Fprintf(w, "  %s: → %s\n", letter, target)
			}
		}
	}

	resp, driveErr := drives.Enumerate()
	if driveErr != nil {
		fmt.Fprintf(w, "Enumeration: error (%v)\n", driveErr)
	} else {
		var fixed, subst, network, other int
		for _, d := range resp.Drives {
			switch d.Type {
			case protocol.DriveFixed:
				fixed++
			case protocol.DriveSubst:
				subst++
			case protocol.DriveNetwork:
				network++
			default:
				other++
			}
		}
		fmt.Fprintf(w, "Detected:  %d drives (%d fixed, %d subst, %d network",
			len(resp.Drives), fixed, subst, network)
		if other > 0 {
			fmt.Fprintf(w, ", %d other", other)
		}
		fmt.Fprintf(w, ")\n")
		if verbose {
			for _, d := range resp.Drives {
				line := fmt.Sprintf("  %s: [%s]", d.Letter, d.Type)
				if d.Target != "" {
					line += fmt.Sprintf(" → %s", d.Target)
				}
				if d.Label != "" {
					line += fmt.Sprintf(" (%s)", d.Label)
				}
				fmt.Fprintln(w, line)
			}
		}
	}

	// Defaults
	if cfg != nil && verbose {
		fmt.Fprintf(w, "\n--- Defaults ---\n")
		fmt.Fprintf(w, "Verb: %s\n", cfg.Defaults.Verb)
		fmt.Fprintf(w, "Show: %s\n", cfg.Defaults.Show)
	}
}
