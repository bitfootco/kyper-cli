package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "config"), 0755)
	os.WriteFile(filepath.Join(dir, "config", "application.rb"), []byte(""), 0644)

	results := DetectStack(dir)
	if len(results) == 0 {
		t.Fatal("expected to detect rails")
	}
	if results[0].Name != "rails" {
		t.Errorf("expected 'rails', got %q", results[0].Name)
	}
}

func TestDetectDjango(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manage.py"), []byte(""), 0644)

	results := DetectStack(dir)
	if len(results) == 0 {
		t.Fatal("expected to detect django")
	}
	if results[0].Name != "django" {
		t.Errorf("expected 'django', got %q", results[0].Name)
	}
}

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	results := DetectStack(dir)
	if len(results) == 0 {
		t.Fatal("expected to detect go")
	}
	if results[0].Name != "go" {
		t.Errorf("expected 'go', got %q", results[0].Name)
	}
}

func TestDetectExpress(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"dependencies": {"express": "^4.18.0"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)

	results := DetectStack(dir)
	found := false
	for _, r := range results {
		if r.Name == "express" {
			found = true
		}
	}
	if !found {
		t.Error("expected to detect express")
	}
}

func TestDetectNext(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"dependencies": {"next": "^14.0.0", "react": "^18.0.0"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)

	results := DetectStack(dir)
	found := false
	for _, r := range results {
		if r.Name == "next" {
			found = true
		}
	}
	if !found {
		t.Error("expected to detect next")
	}
}

func TestDetectPrisma(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "prisma"), 0755)
	os.WriteFile(filepath.Join(dir, "prisma", "schema.prisma"), []byte(""), 0644)

	results := DetectStack(dir)
	found := false
	for _, r := range results {
		if r.Name == "prisma" {
			found = true
		}
	}
	if !found {
		t.Error("expected to detect prisma")
	}
}

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	results := DetectStack(dir)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}
