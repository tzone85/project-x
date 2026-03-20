package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads configuration from the given YAML file path, merging values on
// top of Defaults(). If path is empty, the defaults are returned as-is. The
// returned Config is validated before being returned.
func Load(path string) (Config, error) {
	cfg := Defaults()

	if path == "" {
		if err := cfg.Validate(); err != nil {
			return Config{}, fmt.Errorf("default config validation: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// FindConfigFile searches for a configuration file in well-known locations,
// returning the first match. It checks (in order):
//
//  1. ./px.yaml
//  2. ./px.config.yaml
//  3. ~/.px/config.yaml
//
// Returns an empty string if no file is found.
func FindConfigFile() string {
	candidates := []string{
		"px.yaml",
		"px.config.yaml",
	}

	// Check current-directory candidates.
	for _, name := range candidates {
		abs, err := filepath.Abs(name)
		if err != nil {
			continue
		}
		if fileExists(abs) {
			return abs
		}
	}

	// Check home-directory candidate.
	homePath := expandHome("~/.px/config.yaml")
	if fileExists(homePath) {
		return homePath
	}

	return ""
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}

	return path
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
