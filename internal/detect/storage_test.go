package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeStorageTestFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectStorageRailsSQLite(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "config/application.rb", "")
	writeStorageTestFile(t, dir, "config/database.yml", "production:\n  adapter: sqlite3\n")

	results := DetectStorage(dir)
	if len(results) != 1 || results[0].Path != "storage" {
		t.Fatalf("expected Rails storage mount, got %+v", results)
	}
}

func TestDetectStorageRailsRemoteActiveStorageDoesNotInfer(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "config/application.rb", "")
	writeStorageTestFile(t, dir, "config/environments/production.rb", "config.active_storage.service = :azure\n")
	writeStorageTestFile(t, dir, "config/storage.yml", "local:\n  service: Disk\n")

	results := DetectStorage(dir)
	if len(results) != 0 {
		t.Fatalf("expected no storage mount, got %+v", results)
	}
}

func TestDetectStorageRailsDoesNotInferActiveStorageFromGeneratedLocalConfig(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "config/application.rb", "")
	writeStorageTestFile(t, dir, "config/environments/production.rb", "# configured from credentials\n")
	writeStorageTestFile(t, dir, "config/storage.yml", "local:\n  service: Disk\n")

	results := DetectStorage(dir)
	if len(results) != 0 {
		t.Fatalf("expected no storage mount, got %+v", results)
	}
}

func TestDetectStorageDjangoMediaRoot(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "manage.py", "")
	writeStorageTestFile(t, dir, "project/settings.py", "MEDIA_ROOT = BASE_DIR / \"media\"\n")

	results := DetectStorage(dir)
	if len(results) != 1 || results[0].Path != "media" {
		t.Fatalf("expected Django media mount, got %+v", results)
	}
}

func TestDetectStorageLaravelStorageApp(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "artisan", "")
	writeStorageTestFile(t, dir, ".env", "FILESYSTEM_DISK=local\nDB_DATABASE=database/database.sqlite\n")

	results := DetectStorage(dir)
	if len(results) != 1 || results[0].Path != "storage/app" {
		t.Fatalf("expected Laravel storage/app mount only, got %+v", results)
	}
}

func TestDetectStorageGenericStrongReference(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "package.json", `{"scripts":{"start":"node server.js"}}`)
	writeStorageTestFile(t, dir, "server.js", "const uploadDir = 'uploads';\n")

	results := DetectStorage(dir)
	if len(results) != 1 || results[0].Path != "uploads" {
		t.Fatalf("expected generic uploads mount, got %+v", results)
	}
}

func TestDetectStorageGenericIgnoresDependencyAndBuildDirectories(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "package.json", `{"scripts":{"start":"node server.js"}}`)
	writeStorageTestFile(t, dir, "node_modules/some-package/index.js", "const uploadDir = 'uploads';\n")
	writeStorageTestFile(t, dir, "dist/server.js", "const uploadDir = 'media';\n")

	results := DetectStorage(dir)
	if len(results) != 0 {
		t.Fatalf("expected no storage mount from ignored dirs, got %+v", results)
	}
}

func TestDetectStorageGenericDoesNotInferBareTopLevelData(t *testing.T) {
	dir := t.TempDir()
	writeStorageTestFile(t, dir, "package.json", `{"scripts":{"start":"node server.js"}}`)
	writeStorageTestFile(t, dir, "server.js", "const cacheDir = 'data/cache';\n")

	results := DetectStorage(dir)
	if len(results) != 0 {
		t.Fatalf("expected no storage mount from generic data/cache reference, got %+v", results)
	}
}

func TestDetectStorageBareDirectoryDoesNotInfer(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "uploads"), 0755); err != nil {
		t.Fatal(err)
	}

	results := DetectStorage(dir)
	if len(results) != 0 {
		t.Fatalf("expected no storage mount from bare directory, got %+v", results)
	}
}
