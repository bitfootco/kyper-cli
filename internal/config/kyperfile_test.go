package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKyperFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kyper.yml")
	content := `name: My App
version: 1.0.0
description: A test app
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: bin/rails server
  worker: bundle exec sidekiq
pricing:
  one_time: 29.99
  subscription: 9.99
healthcheck:
  path: /up
  interval: 30
  timeout: 10
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	kf, raw, err := LoadKyperFile(path)
	if err != nil {
		t.Fatalf("LoadKyperFile failed: %v", err)
	}
	if len(raw) == 0 {
		t.Error("expected raw bytes")
	}
	if kf.Name != "My App" {
		t.Errorf("expected title 'My App', got %q", kf.Name)
	}
	if kf.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", kf.Version)
	}
	if kf.Category != "productivity" {
		t.Errorf("expected category 'productivity', got %q", kf.Category)
	}
	if kf.Docker.Dockerfile != "./Dockerfile" {
		t.Errorf("expected dockerfile './Dockerfile', got %q", kf.Docker.Dockerfile)
	}
	if len(kf.Processes) != 2 {
		t.Errorf("expected 2 processes, got %d", len(kf.Processes))
	}
	if kf.Processes["web"] != "bin/rails server" {
		t.Errorf("unexpected web process: %q", kf.Processes["web"])
	}
	if kf.Pricing.OneTime == nil || *kf.Pricing.OneTime != 29.99 {
		t.Errorf("unexpected one_time price: %v", kf.Pricing.OneTime)
	}
	if kf.Pricing.Subscription == nil || *kf.Pricing.Subscription != 9.99 {
		t.Errorf("unexpected subscription price: %v", kf.Pricing.Subscription)
	}
	if kf.Healthcheck.Path != "/up" {
		t.Errorf("expected healthcheck path '/up', got %q", kf.Healthcheck.Path)
	}
}

func TestDepEntryStringFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kyper.yml")
	content := `name: test
version: 1.0.0
description: test
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: start
deps:
  - postgres
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	kf, _, err := LoadKyperFile(path)
	if err != nil {
		t.Fatalf("LoadKyperFile failed: %v", err)
	}
	if len(kf.Deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(kf.Deps))
	}
	if kf.Deps[0].Name != "postgres" || kf.Deps[0].Version != "" {
		t.Errorf("expected postgres with no version, got %+v", kf.Deps[0])
	}
}

func TestDepEntryColonPinnedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kyper.yml")
	content := `name: test
version: 1.0.0
description: test
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: start
deps:
  - "redis:7"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	kf, _, err := LoadKyperFile(path)
	if err != nil {
		t.Fatalf("LoadKyperFile failed: %v", err)
	}
	if len(kf.Deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(kf.Deps))
	}
	if kf.Deps[0].Name != "redis" || kf.Deps[0].Version != "7" {
		t.Errorf("expected redis:7, got %+v", kf.Deps[0])
	}
}

func TestDepEntryHashFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kyper.yml")
	content := `name: test
version: 1.0.0
description: test
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: start
deps:
  - postgres: "16"
    storage_gb: 50
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	kf, _, err := LoadKyperFile(path)
	if err != nil {
		t.Fatalf("LoadKyperFile failed: %v", err)
	}
	if len(kf.Deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(kf.Deps))
	}
	d := kf.Deps[0]
	if d.Name != "postgres" || d.Version != "16" || d.StorageGB != 50 {
		t.Errorf("expected postgres:16 with 50GB, got %+v", d)
	}
}

func TestLoadKyperFileMissing(t *testing.T) {
	_, _, err := LoadKyperFile("/nonexistent/kyper.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadKyperFileMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kyper.yml")
	if err := os.WriteFile(path, []byte("{{{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadKyperFile(path)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}
