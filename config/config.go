package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all persisted application settings.
type Config struct {
	GameDir     string `json:"game_dir"`
	PluginName  string `json:"plugin_name"` // e.g. "Ascy"
	PluginLang  string `json:"plugin_lang"` // e.g. "e", "ru", "cn", "k" (derived from .dat filename)
	RunMode     string `json:"run_mode"`    // "click" | "tray"
	AutoStartup bool   `json:"auto_startup"`
	Configured  bool   `json:"configured"`
}

func configDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "RebornPluginAutoinstaller")
	return dir, os.MkdirAll(dir, 0755)
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk, returning defaults if not found.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), nil
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// DefaultConfig returns a zero-value config with safe defaults.
func DefaultConfig() *Config {
	return &Config{
		RunMode:    "click",
		Configured: false,
	}
}
