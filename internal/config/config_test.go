package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := &Config{APIToken: "test-token-abc123"}
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if loaded.APIToken != cfg.APIToken {
		t.Errorf("expected token %q, got %q", cfg.APIToken, loaded.APIToken)
	}
}

func TestSaveCreatesFileWith0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := &Config{APIToken: "secret"}
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %04o", perm)
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom should not error for missing file: %v", err)
	}
	if cfg.APIToken != "" {
		t.Errorf("expected empty token, got %q", cfg.APIToken)
	}
}
