package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSuggestDepVersionsFromGemfileLock(t *testing.T) {
	dir := t.TempDir()
	lockContent := `GEM
  remote: https://rubygems.org/
  specs:
    pg (1.5.6)
    redis (5.2.0)
    rails (7.1.0)

PLATFORMS
  ruby

DEPENDENCIES
  pg
  redis
  rails
`
	if err := os.WriteFile(filepath.Join(dir, "Gemfile.lock"), []byte(lockContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps := []DepResult{
		{Name: "postgres", Source: "Gemfile"},
		{Name: "redis", Source: "Gemfile"},
	}

	suggestions := SuggestDepVersions(dir, deps)

	found := make(map[string]string)
	for _, s := range suggestions {
		found[s.Dep] = s.Version
	}

	if found["postgres"] != "1" {
		t.Errorf("expected postgres version '1', got %q", found["postgres"])
	}
	if found["redis"] != "5" {
		t.Errorf("expected redis version '5', got %q", found["redis"])
	}
}

func TestSuggestDepVersionsFromPackageLock(t *testing.T) {
	dir := t.TempDir()
	lockContent := `{
  "name": "my-app",
  "lockfileVersion": 2,
  "dependencies": {
    "pg": {
      "version": "8.12.0",
      "resolved": "https://registry.npmjs.org/pg/-/pg-8.12.0.tgz"
    },
    "ioredis": {
      "version": "5.4.1",
      "resolved": "https://registry.npmjs.org/ioredis/-/ioredis-5.4.1.tgz"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(lockContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps := []DepResult{
		{Name: "postgres", Source: "package.json"},
		{Name: "redis", Source: "package.json"},
	}

	suggestions := SuggestDepVersions(dir, deps)

	found := make(map[string]string)
	for _, s := range suggestions {
		found[s.Dep] = s.Version
	}

	if found["postgres"] != "8" {
		t.Errorf("expected postgres version '8', got %q", found["postgres"])
	}
	if found["redis"] != "5" {
		t.Errorf("expected redis version '5', got %q", found["redis"])
	}
}

func TestSuggestDepVersionsNoLockfiles(t *testing.T) {
	dir := t.TempDir()
	deps := []DepResult{{Name: "postgres", Source: "Gemfile"}}

	suggestions := SuggestDepVersions(dir, deps)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions without lockfiles, got %d", len(suggestions))
	}
}

func TestSuggestDepVersionsSkipsUnwantedDeps(t *testing.T) {
	dir := t.TempDir()
	lockContent := `GEM
  specs:
    pg (1.5.6)
    redis (5.2.0)
`
	if err := os.WriteFile(filepath.Join(dir, "Gemfile.lock"), []byte(lockContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Only ask for postgres, not redis
	deps := []DepResult{{Name: "postgres", Source: "Gemfile"}}

	suggestions := SuggestDepVersions(dir, deps)
	for _, s := range suggestions {
		if s.Dep == "redis" {
			t.Error("should not suggest redis when not in deps list")
		}
	}
}
