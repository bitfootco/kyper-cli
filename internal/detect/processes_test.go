package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProcessesFromProcfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Procfile"), []byte("web: bin/rails server\nworker: bundle exec sidekiq\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectProcesses(dir)
	if len(results) != 2 {
		t.Fatalf("expected 2 processes, got %d", len(results))
	}

	webFound := false
	for _, r := range results {
		if r.Name == "web" && r.Command == "bin/rails server" {
			webFound = true
		}
	}
	if !webFound {
		t.Error("expected to find web process from Procfile")
	}
}

func TestDetectProcessesFromDockerfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM ruby:3.2\nCMD [\"bin/rails\", \"server\"]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectProcesses(dir)
	if len(results) == 0 {
		t.Fatal("expected to detect process from Dockerfile")
	}
	if results[0].Name != "web" {
		t.Errorf("expected 'web', got %q", results[0].Name)
	}
}

func TestDetectProcessesFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"scripts": {"start": "node server.js"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectProcesses(dir)
	if len(results) == 0 {
		t.Fatal("expected to detect process from package.json")
	}
	if results[0].Command != "node server.js" {
		t.Errorf("expected 'node server.js', got %q", results[0].Command)
	}
}
