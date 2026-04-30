package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// DefaultConfigPath returns the default per-user config path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".picobot", "config.json")
}

// LoadConfig loads config from ~/.picobot/config.json if present, then applies any environment variable overrides on top.
func LoadConfig() (Config, error) {
	path := DefaultConfigPath()
	var cfg Config
	f, err := os.Open(path)
	if err == nil {
		defer func() { _ = f.Close() }()
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return Config{}, err
		}
	}
	// env vars always take precedence over the config file, enabling runtime overrides without editing config.json.
	applyEnvOverrides(&cfg)
	return cfg, nil
}

// applyEnvOverrides updates config fields from all environment variables
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PICOBOT_MODEL"); v != "" {
		cfg.Agents.Defaults.Model = v
	}
	if v := os.Getenv("PICOBOT_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Agents.Defaults.MaxTokens = n
		}
	}
	if v := os.Getenv("PICOBOT_MAX_TOOL_ITERATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Agents.Defaults.MaxToolIterations = n
		}
	}
	if v := os.Getenv("PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR"); v != "" {
		b := v != "false" && v != "0" && v != "False" && v != "FALSE"
		cfg.Agents.Defaults.EnableToolActivityIndicator = &b
	}
}
