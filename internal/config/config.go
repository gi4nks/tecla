package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CustomRecommendation struct {
	Condition string `json:"condition"`
	Text      string `json:"text"`
	Command   string `json:"command,omitempty"`
}

type Config struct {
	IgnoredPaths          []string               `json:"ignored_paths"`
	DefaultIgnoredDirs    []string               `json:"default_ignored_dirs"`
	StaleThresholdDays    int                    `json:"stale_threshold_days"`
	AutoFetch             bool                   `json:"auto_fetch"`
	CustomRecommendations []CustomRecommendation `json:"custom_recommendations"`
}

func (c *Config) Validate() error {
	if c.StaleThresholdDays < 0 {
		return fmt.Errorf("stale_threshold_days cannot be negative")
	}
	for i, cr := range c.CustomRecommendations {
		if cr.Condition == "" {
			return fmt.Errorf("custom_recommendation[%d]: condition cannot be empty", i)
		}
		if cr.Text == "" {
			return fmt.Errorf("custom_recommendation[%d]: text cannot be empty", i)
		}
	}
	return nil
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tecla"), nil
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultCfg(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := defaultCfg()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.IgnoredPaths == nil {
		cfg.IgnoredPaths = []string{}
	}

	return cfg, nil
}

func defaultCfg() *Config {
	return &Config{
		IgnoredPaths:       []string{},
		DefaultIgnoredDirs: []string{"node_modules", "dist", "build", ".cache", ".venv", "target", ".terraform"},
		StaleThresholdDays: 30,
	}
}

func Save(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) AddIgnore(path string) bool {
	for _, p := range c.IgnoredPaths {
		if p == path {
			return false
		}
	}
	c.IgnoredPaths = append(c.IgnoredPaths, path)
	return true
}
