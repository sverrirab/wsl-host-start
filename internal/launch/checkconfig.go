package launch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/config"
)

// configReport holds the diagnostic information about the active configuration.
type configReport struct {
	HelperPath string
	HelperDir  string

	// Config
	ConfigLoaded bool
	Config       *config.Config

	// Allowlist
	AllowlistLoaded bool
	AllowlistPath   string
	AllowlistRules  []allowlist.Rule

	// Env analysis
	ForwardedVars []string // vars that would be forwarded (set in env and not blocked)
	BlockedVars   []string // vars in forward list that are blocked
	MissingVars   []string // vars in forward list that are not set in env
}

// CheckConfig loads the active configuration and prints a diagnostic report.
func CheckConfig(verbose bool) error {
	report, err := buildConfigReport()
	if err != nil {
		return err
	}
	printConfigReport(os.Stdout, report, verbose)
	return nil
}

// buildConfigReport gathers all config state without printing anything.
func buildConfigReport() (*configReport, error) {
	report := &configReport{}

	helperPath, err := findHelper()
	if err != nil {
		return nil, err
	}
	report.HelperPath = helperPath
	report.HelperDir = filepath.Dir(helperPath)

	cfg, err := config.Load(report.HelperDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	report.ConfigLoaded = true
	report.Config = cfg

	al, err := allowlist.Load(report.HelperDir)
	if err != nil {
		return nil, fmt.Errorf("loading allowlist: %w", err)
	}
	report.AllowlistLoaded = al.Loaded
	report.AllowlistPath = al.Path
	if al.Loaded && al.List != nil {
		report.AllowlistRules = al.List.Allow
	}

	// Analyze env forwarding.
	blocked := make(map[string]bool)
	for _, b := range cfg.Env.Block {
		blocked[strings.ToUpper(b)] = true
	}
	for _, name := range cfg.Env.Forward {
		upper := strings.ToUpper(name)
		if blocked[upper] {
			report.BlockedVars = append(report.BlockedVars, name)
			continue
		}
		if _, ok := os.LookupEnv(name); ok {
			report.ForwardedVars = append(report.ForwardedVars, name)
		} else {
			report.MissingVars = append(report.MissingVars, name)
		}
	}

	return report, nil
}

// checkConfigReport generates the diagnostic text for a given report.
// Extracted for testability.
func checkConfigReport(w io.Writer, report *configReport, verbose bool) {
	printConfigReport(w, report, verbose)
}

func printConfigReport(w io.Writer, report *configReport, verbose bool) {
	fmt.Fprintf(w, "Helper:    %s\n", report.HelperPath)
	configPath := filepath.Join(report.HelperDir, config.ConfigFile)
	if report.ConfigLoaded {
		fmt.Fprintf(w, "Config:    %s\n", configPath)
	} else {
		fmt.Fprintf(w, "Config:    %s (not found — using defaults)\n", configPath)
	}

	// Deny list
	fmt.Fprintf(w, "\n--- Deny List ---\n")
	fmt.Fprintf(w, "Blocked:   %s\n", strings.Join(allowlist.DeniedPrograms(), ", "))

	// Allowlist
	fmt.Fprintf(w, "\n--- Allowlist ---\n")
	fmt.Fprintf(w, "File:      %s\n", report.AllowlistPath)
	if !report.AllowlistLoaded {
		fmt.Fprintf(w, "Status:    NOT ACTIVE (file not found — all programs allowed)\n")
	} else if len(report.AllowlistRules) == 0 {
		fmt.Fprintf(w, "Status:    ACTIVE (empty — all programs DENIED)\n")
	} else {
		fmt.Fprintf(w, "Status:    ACTIVE (%d rules)\n", len(report.AllowlistRules))
		for _, rule := range report.AllowlistRules {
			if len(rule.Commands) == 0 {
				fmt.Fprintf(w, "  allow:   %s (any args)\n", rule.Program)
			} else {
				fmt.Fprintf(w, "  allow:   %s [%s]\n", rule.Program, strings.Join(rule.Commands, ", "))
			}
		}
	}

	// Env forwarding
	fmt.Fprintf(w, "\n--- Environment ---\n")
	if len(report.Config.Env.Forward) == 0 {
		fmt.Fprintf(w, "Forward:   (none)\n")
	} else {
		fmt.Fprintf(w, "Forward:   %s\n", strings.Join(report.Config.Env.Forward, ", "))
	}
	if len(report.Config.Env.Block) == 0 {
		fmt.Fprintf(w, "Block:     (none)\n")
	} else {
		fmt.Fprintf(w, "Block:     %s\n", strings.Join(report.Config.Env.Block, ", "))
	}

	if len(report.ForwardedVars) > 0 {
		fmt.Fprintf(w, "Will forward:  %s\n", strings.Join(report.ForwardedVars, ", "))
	}
	if len(report.BlockedVars) > 0 {
		fmt.Fprintf(w, "Blocked:       %s (in forward list but blocked)\n", strings.Join(report.BlockedVars, ", "))
	}
	if len(report.MissingVars) > 0 {
		fmt.Fprintf(w, "Not set:       %s (in forward list but not in environment)\n", strings.Join(report.MissingVars, ", "))
	}

	// Drive config
	if verbose {
		fmt.Fprintf(w, "\n--- Drives ---\n")
		fmt.Fprintf(w, "Auto-detect: %v\n", report.Config.Drives.AutoDetect)
		fmt.Fprintf(w, "Prefer aliases: %v\n", report.Config.Drives.PreferAliases)
		if len(report.Config.Drives.Aliases) > 0 {
			for letter, target := range report.Config.Drives.Aliases {
				fmt.Fprintf(w, "  %s: → %s\n", letter, target)
			}
		}

		fmt.Fprintf(w, "\n--- Defaults ---\n")
		fmt.Fprintf(w, "Verb: %s\n", report.Config.Defaults.Verb)
		fmt.Fprintf(w, "Show: %s\n", report.Config.Defaults.Show)
	}
}
