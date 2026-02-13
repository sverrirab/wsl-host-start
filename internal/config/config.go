// Package config loads the wstart TOML configuration file.
// The config file lives alongside wstart-host.exe on the Windows host.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigFile is the expected filename in the helper's directory.
const ConfigFile = "config.toml"

type Config struct {
	Drives   DrivesConfig   `toml:"drives"`
	Env      EnvConfig      `toml:"env"`
	Defaults DefaultsConfig `toml:"defaults"`
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

// Load reads the config from the given directory (typically the directory
// containing wstart-host.exe). Returns defaults if the file doesn't exist.
func Load(dir string) (*Config, error) {
	cfg := defaults()

	if dir == "" {
		return cfg, nil
	}

	path := filepath.Join(dir, ConfigFile)
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
