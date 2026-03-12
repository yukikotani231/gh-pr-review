package config

import (
	"os"
	"testing"
)

func TestLoadMissingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DiffMode != "" {
		t.Fatalf("DiffMode = %q, want empty default", cfg.DiffMode)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := Save(AppConfig{DiffMode: "split"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DiffMode != "split" {
		t.Fatalf("DiffMode = %q, want split", cfg.DiffMode)
	}

	wantPath, err := path()
	if err != nil {
		t.Fatalf("path() error = %v", err)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected config file at %s: %v", wantPath, err)
	}
}
