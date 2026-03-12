package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appDirName = "gh-pr-review"
const configFileName = "config.json"

type AppConfig struct {
	DiffMode string `json:"diff_mode"`
}

func Load() (AppConfig, error) {
	path, err := path()
	if err != nil {
		return AppConfig{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AppConfig{}, nil
		}
		return AppConfig{}, fmt.Errorf("config read failed: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("config parse failed: %w", err)
	}
	if cfg.DiffMode != "split" {
		cfg.DiffMode = "unified"
	}
	return cfg, nil
}

func Save(cfg AppConfig) error {
	path, err := path()
	if err != nil {
		return err
	}
	if cfg.DiffMode != "split" {
		cfg.DiffMode = "unified"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config dir create failed: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config marshal failed: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("config write failed: %w", err)
	}
	return nil
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir lookup failed: %w", err)
	}
	return filepath.Join(dir, appDirName, configFileName), nil
}
