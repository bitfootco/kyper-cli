package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZip(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "app.rb"), []byte("puts 'hello'"), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM ruby"), 0644)
	os.MkdirAll(filepath.Join(dir, "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "lib", "helper.rb"), []byte("module Helper; end"), 0644)

	outPath := filepath.Join(t.TempDir(), "output.zip")
	if err := Create(dir, outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify zip contents
	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	files := make(map[string]bool)
	for _, f := range r.File {
		files[f.Name] = true
	}

	if !files["app.rb"] {
		t.Error("expected app.rb in zip")
	}
	if !files["Dockerfile"] {
		t.Error("expected Dockerfile in zip")
	}
	if !files["lib/helper.rb"] {
		t.Error("expected lib/helper.rb in zip")
	}
}

func TestCreateZipExcludesGit(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "app.rb"), []byte("hello"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644)

	outPath := filepath.Join(t.TempDir(), "output.zip")
	if err := Create(dir, outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == ".git" || filepath.Dir(f.Name) == ".git" {
			t.Errorf("zip should not contain .git files, found: %s", f.Name)
		}
	}
}

func TestCreateZipExcludesNodeModules(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "index.js"), []byte("console.log('hi')"), 0644)
	os.MkdirAll(filepath.Join(dir, "node_modules", "express"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "express", "index.js"), []byte(""), 0644)

	outPath := filepath.Join(t.TempDir(), "output.zip")
	if err := Create(dir, outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(filepath.Dir(f.Name)) == "node_modules" || filepath.Base(f.Name) == "node_modules" {
			t.Errorf("zip should not contain node_modules, found: %s", f.Name)
		}
	}
}

func TestCreateZipRespectsKyperignore(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "app.rb"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "secret.key"), []byte("shh"), 0644)
	os.WriteFile(filepath.Join(dir, ".kyperignore"), []byte("secret.key\n"), 0644)

	outPath := filepath.Join(t.TempDir(), "output.zip")
	if err := Create(dir, outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "secret.key" {
			t.Error("zip should not contain secret.key (in .kyperignore)")
		}
	}
}

func TestCreateZipExcludesLogFiles(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "app.rb"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log data"), 0644)

	outPath := filepath.Join(t.TempDir(), "output.zip")
	if err := Create(dir, outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "debug.log" {
			t.Error("zip should not contain .log files")
		}
	}
}
