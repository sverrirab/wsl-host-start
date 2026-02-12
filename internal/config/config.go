// Package config loads the wstart TOML configuration file.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Host     HostConfig     `toml:"host"`
	Drives   DrivesConfig   `toml:"drives"`
	Env      EnvConfig      `toml:"env"`
	Defaults DefaultsConfig `toml:"defaults"`
}

type HostConfig struct {
	Helper string `toml:"helper"`
}

type DrivesConfig struct {
	// Manual drive alias overrides: letter → Windows target path.
	Aliases       map[string]string `toml:"aliases"`
	PreferAliases bool              `toml:"prefer_aliases"`
	AutoDetect    bool              `toml:"auto_detect"`
}

type EnvConfig struct {
	Forward []string `toml:"forward"`
	Block   []string `toml:"block"`
}

type DefaultsConfig struct {
	Verb string `toml:"verb"`
	Show string `toml:"show"`
}

// Load reads the config from ~/.config/wstart/config.toml.
// Returns defaults if the file doesn't exist.
func Load() (*Config, error) {
	cfg := defaults()

	path := configPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Drives: DrivesConfig{
			PreferAliases: true,
			AutoDetect:    true,
		},
		Env: EnvConfig{
			Block: []string{
				"P4PASSWD",
				"P4TICKETS",
				"P4TRUST",
			},
		},
		Defaults: DefaultsConfig{
			Verb: "open",
			Show: "normal",
		},
	}
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "wstart", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wstart", "config.toml")
}
