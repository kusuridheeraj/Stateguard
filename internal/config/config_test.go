package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestLoadUsesDefaultWhenNoFileExists(t *testing.T) {
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() {
		_ = os.Chdir(previous)
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	if cfg.Source != "defaults" {
		t.Fatalf("expected defaults source, got %q", cfg.Source)
	}
}

func TestLoadFromPath(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "safedata.yaml")
	content := []byte("project:\n  name: demo\npolicy:\n  retention:\n    window: 24h\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Project.Name != "demo" {
		t.Fatalf("expected project name demo, got %q", cfg.Project.Name)
	}
	if cfg.Source != path {
		t.Fatalf("expected source path %q, got %q", path, cfg.Source)
	}
}
