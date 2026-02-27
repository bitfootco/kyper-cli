package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDepsFromGemfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("gem 'pg'\ngem 'redis'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["postgres"] {
		t.Error("expected to detect postgres from Gemfile")
	}
	if !found["redis"] {
		t.Error("expected to detect redis from Gemfile")
	}
}

func TestDetectDepsFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"dependencies": {"pg": "^8.0.0", "ioredis": "^5.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["postgres"] {
		t.Error("expected to detect postgres from package.json")
	}
	if !found["redis"] {
		t.Error("expected to detect redis from package.json")
	}
}

func TestDetectDepsFromRequirements(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("psycopg2==2.9.9\nredis>=4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["postgres"] {
		t.Error("expected to detect postgres from requirements.txt")
	}
	if !found["redis"] {
		t.Error("expected to detect redis from requirements.txt")
	}
}

func TestDetectDepsS3FromGemfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("gem 'aws-sdk-s3'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["s3"] {
		t.Error("expected to detect s3 from Gemfile aws-sdk-s3")
	}
}

func TestDetectDepsS3FromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"dependencies": {"@aws-sdk/client-s3": "^3.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}
	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["s3"] {
		t.Error("expected to detect s3 from package.json @aws-sdk/client-s3")
	}
}

func TestDetectDepsS3FromRequirements(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("boto3==1.34.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["s3"] {
		t.Error("expected to detect s3 from requirements.txt boto3")
	}
}

func TestDetectDepsS3FromDockerCompose(t *testing.T) {
	dir := t.TempDir()
	compose := "services:\n  storage:\n    image: chrislusf/seaweedfs:latest\n"
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		t.Fatal(err)
	}
	results := DetectDeps(dir)
	found := make(map[string]bool)
	for _, r := range results {
		found[r.Name] = true
	}
	if !found["s3"] {
		t.Error("expected to detect s3 from docker-compose.yml seaweedfs image")
	}
}

func TestDetectDepsDeduplication(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("gem 'pg'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  db:\n    image: postgres:16\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := DetectDeps(dir)
	count := 0
	for _, r := range results {
		if r.Name == "postgres" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 postgres dep (deduplicated), got %d", count)
	}
}
