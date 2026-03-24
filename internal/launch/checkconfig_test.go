package launch_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/config"
	"github.com/sverrirab/wsl-host-start/internal/launch"
)

func TestCheckConfigReportNoAllowlist(t *testing.T) {
	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/Users/test/AppData/Local/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/Users/test/AppData/Local/wstart",
		ConfigLoaded:    true,
		AllowlistLoaded: false,
		AllowlistPath:   "/mnt/c/Users/test/AppData/Local/wstart/allowlist.toml",
		Config: &config.Config{
			Env: config.EnvConfig{
				Block: []string{"P4PASSWD"},
			},
			Drives:   config.DrivesConfig{AutoDetect: true, PreferAliases: true},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, false)
	out := buf.String()

	assertContains(t, out, "NOT ACTIVE")
	assertContains(t, out, "all programs allowed")
	assertContains(t, out, "P4PASSWD")
}

func TestCheckConfigReportEmptyAllowlist(t *testing.T) {
	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/wstart",
		ConfigLoaded:    true,
		AllowlistLoaded: true,
		AllowlistPath:   "/mnt/c/wstart/allowlist.toml",
		AllowlistRules:  []allowlist.Rule{},
		Config: &config.Config{
			Env:      config.EnvConfig{},
			Drives:   config.DrivesConfig{AutoDetect: true},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, false)
	out := buf.String()

	assertContains(t, out, "ACTIVE (empty")
	assertContains(t, out, "all programs DENIED")
}

func TestCheckConfigReportWithRules(t *testing.T) {
	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/wstart",
		ConfigLoaded:    true,
		AllowlistLoaded: true,
		AllowlistPath:   "/mnt/c/wstart/allowlist.toml",
		AllowlistRules: []allowlist.Rule{
			{Program: "p4", Commands: []string{"edit", "sync"}},
			{Program: "notepad.exe"},
		},
		Config: &config.Config{
			Env:      config.EnvConfig{},
			Drives:   config.DrivesConfig{AutoDetect: true},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, false)
	out := buf.String()

	assertContains(t, out, "ACTIVE (2 rules)")
	assertContains(t, out, "p4 [edit, sync]")
	assertContains(t, out, "notepad.exe (any args)")
}

func TestCheckConfigReportEnvAnalysis(t *testing.T) {
	t.Setenv("P4PORT", "ssl:host:1666")
	t.Setenv("P4CLIENT", "myclient")
	// P4PASSWD intentionally not set via Setenv — it's in block list anyway.

	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/wstart",
		ConfigLoaded:    true,
		AllowlistLoaded: false,
		AllowlistPath:   "/mnt/c/wstart/allowlist.toml",
		Config: &config.Config{
			Env: config.EnvConfig{
				Forward: []string{"P4PORT", "P4CLIENT", "P4PASSWD", "P4USER"},
				Block:   []string{"P4PASSWD"},
			},
			Drives:   config.DrivesConfig{AutoDetect: true},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
		ForwardedVars: []string{"P4PORT", "P4CLIENT"},
		BlockedVars:   []string{"P4PASSWD"},
		MissingVars:   []string{"P4USER"},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, false)
	out := buf.String()

	assertContains(t, out, "Forward:   P4PORT, P4CLIENT, P4PASSWD, P4USER")
	assertContains(t, out, "Block:     P4PASSWD")
	assertContains(t, out, "Will forward:  P4PORT, P4CLIENT")
	assertContains(t, out, "Blocked:       P4PASSWD")
	assertContains(t, out, "Not set:       P4USER")
}

func TestCheckConfigReportConfigNotFound(t *testing.T) {
	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/wstart",
		ConfigLoaded:    false,
		AllowlistLoaded: false,
		AllowlistPath:   "/mnt/c/wstart/allowlist.toml",
		Config: &config.Config{
			Env:      config.EnvConfig{Block: []string{"P4PASSWD"}},
			Drives:   config.DrivesConfig{AutoDetect: true, PreferAliases: true},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, false)
	out := buf.String()

	assertContains(t, out, "config.toml (not found — using defaults)")
}

func TestCheckConfigReportVerbose(t *testing.T) {
	report := &launch.ConfigReport{
		HelperPath:      "/mnt/c/wstart/wstart-host.exe",
		HelperDir:       "/mnt/c/wstart",
		ConfigLoaded:    true,
		AllowlistLoaded: false,
		AllowlistPath:   "/mnt/c/wstart/allowlist.toml",
		Config: &config.Config{
			Env: config.EnvConfig{},
			Drives: config.DrivesConfig{
				AutoDetect:    true,
				PreferAliases: true,
				Aliases:       map[string]string{"Z": `\\server\share`},
			},
			Defaults: config.DefaultsConfig{Verb: "open", Show: "normal"},
		},
	}

	var buf bytes.Buffer
	launch.CheckConfigReport(&buf, report, true)
	out := buf.String()

	assertContains(t, out, "Auto-detect: true")
	assertContains(t, out, "Prefer aliases: true")
	assertContains(t, out, `Z: → \\server\share`)
	assertContains(t, out, "Verb: open")
	assertContains(t, out, "Show: normal")

	// Non-verbose should NOT include drive/defaults sections.
	var bufShort bytes.Buffer
	launch.CheckConfigReport(&bufShort, report, false)
	shortOut := bufShort.String()
	if strings.Contains(shortOut, "Auto-detect") {
		t.Error("non-verbose output should not include drive details")
	}
}

func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output missing %q:\n%s", substr, output)
	}
}
