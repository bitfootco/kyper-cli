package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type StorageResult struct {
	Path   string
	Source string
}

var genericStorageConfigKeyRegexp = regexp.MustCompile(`(?i)(upload|uploads|media|storage|file|files|asset|assets)[A-Za-z0-9_.-]{0,32}(dir|path|root|folder|directory|location)|(dir|path|root|folder|directory|location)[A-Za-z0-9_.-]{0,32}(upload|uploads|media|storage|file|files|asset|assets)`)
var quotedStoragePathRegexp = regexp.MustCompile(`["']([^"']+)["']`)

func DetectStorage(dir string) []StorageResult {
	seen := make(map[string]bool)
	var results []StorageResult
	add := func(path, source string) {
		path = firstMutableSegment(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		results = append(results, StorageResult{Path: path, Source: source})
	}

	if isRails(dir) {
		if railsUsesSQLite(dir) || railsUsesLocalActiveStorage(dir) {
			add("storage", "Rails storage config")
		}
		return results
	}

	if isDjango(dir) {
		if media := djangoMediaRoot(dir); safeMutablePath(media) {
			add(media, "Django MEDIA_ROOT")
		}
		if db := djangoSQLitePath(dir); db != "" && safeMutablePath(db) {
			add(db, "Django SQLite config")
		}
		return results
	}

	if isLaravel(dir) {
		if laravelUsesLocalStorage(dir) {
			add("storage/app", "Laravel filesystem config")
		}
		if db := laravelSQLitePath(dir); strings.HasPrefix(db, "storage/") {
			add("storage", "Laravel SQLite config")
		}
		return results
	}

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && ignoredDir(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !smallConfigFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, candidate := range genericStoragePathCandidates(string(data)) {
			if safeMutablePath(candidate) {
				rel, _ := filepath.Rel(dir, path)
				add(candidate, rel)
			}
		}
		return nil
	})

	return results
}

func isRails(dir string) bool {
	if fileExists(filepath.Join(dir, "config/application.rb")) {
		return true
	}
	data, err := os.ReadFile(filepath.Join(dir, "Gemfile.lock"))
	return err == nil && regexp.MustCompile(`(?m)^\s{4}rails \(`).Match(data)
}

func isDjango(dir string) bool {
	return fileExists(filepath.Join(dir, "manage.py"))
}

func isLaravel(dir string) bool {
	return fileExists(filepath.Join(dir, "artisan"))
}

func railsUsesSQLite(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "config/database.yml"))
	if err != nil {
		return false
	}

	var parsed map[string]interface{}
	if yaml.Unmarshal(data, &parsed) == nil {
		if production, ok := parsed["production"]; ok {
			return configContainsSQLite(production)
		}
		return configContainsSQLite(parsed)
	}

	text := strings.ToLower(string(data))
	productionIndex := strings.Index(text, "production:")
	if productionIndex < 0 {
		return strings.Contains(text, "adapter: sqlite3")
	}
	nextEnv := regexp.MustCompile(`(?m)^[a-z_]+:`).FindStringIndex(text[productionIndex+len("production:"):])
	productionBlock := text[productionIndex:]
	if nextEnv != nil {
		productionBlock = text[productionIndex : productionIndex+len("production:")+nextEnv[0]]
	}
	return strings.Contains(productionBlock, "adapter: sqlite3")
}

func configContainsSQLite(value interface{}) bool {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, nested := range typed {
			if key == "adapter" && nested == "sqlite3" {
				return true
			}
			if configContainsSQLite(nested) {
				return true
			}
		}
	case map[interface{}]interface{}:
		for key, nested := range typed {
			if key == "adapter" && nested == "sqlite3" {
				return true
			}
			if configContainsSQLite(nested) {
				return true
			}
		}
	case []interface{}:
		for _, nested := range typed {
			if configContainsSQLite(nested) {
				return true
			}
		}
	}
	return false
}

func railsUsesLocalActiveStorage(dir string) bool {
	prod, _ := os.ReadFile(filepath.Join(dir, "config/environments/production.rb"))
	prodText := strings.ToLower(string(prod))
	service := railsActiveStorageProductionService(prodText)
	if service == "" {
		return false
	}
	if remoteActiveStorageService(service) {
		return false
	}
	if service == "local" {
		return true
	}
	return railsStorageServiceUsesDisk(dir, service)
}

func railsActiveStorageProductionService(content string) string {
	match := regexp.MustCompile(`config\.active_storage\.service\s*=\s*:([a-zA-Z0-9_]+)`).FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func remoteActiveStorageService(service string) bool {
	switch service {
	case "amazon", "s3", "spaces", "digitalocean", "gcs", "google", "azure", "mirror":
		return true
	default:
		return false
	}
}

func railsStorageServiceUsesDisk(dir, service string) bool {
	storage, err := os.ReadFile(filepath.Join(dir, "config/storage.yml"))
	if err != nil {
		return false
	}
	sectionRegexp := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(service) + `:\s*$\n(?:(?:\s{2,}|\t).*\n?)*`)
	section := sectionRegexp.FindString(string(storage))
	return strings.Contains(section, "service: Disk")
}

func djangoMediaRoot(dir string) string {
	for _, settings := range djangoSettingsFiles(dir) {
		data, _ := os.ReadFile(settings)
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "MEDIA_ROOT") {
				continue
			}
			if path := extractPythonBaseDirPath(line); path != "" {
				return path
			}
		}
	}
	return ""
}

func djangoSQLitePath(dir string) string {
	for _, settings := range djangoSettingsFiles(dir) {
		data, _ := os.ReadFile(settings)
		content := string(data)
		if !strings.Contains(content, "sqlite3") {
			continue
		}
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, "NAME") {
				if path := extractPythonBaseDirPath(line); strings.HasSuffix(path, ".sqlite3") || strings.HasSuffix(path, ".sqlite") {
					return path
				}
			}
		}
	}
	return ""
}

func djangoSettingsFiles(dir string) []string {
	var paths []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && ignoredDir(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "settings") && strings.HasSuffix(path, ".py") {
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}

func extractPythonBaseDirPath(line string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`BASE_DIR\s*/\s*["']([^"']+)["']`),
		regexp.MustCompile(`os\.path\.join\(BASE_DIR,\s*["']([^"']+)["']`),
		regexp.MustCompile(`["']([^"']+)["']`),
	}
	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(line); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func laravelUsesLocalStorage(dir string) bool {
	env, _ := os.ReadFile(filepath.Join(dir, ".env"))
	fs, _ := os.ReadFile(filepath.Join(dir, "config/filesystems.php"))
	content := string(env) + "\n" + string(fs)
	return strings.Contains(content, "FILESYSTEM_DISK=local") ||
		strings.Contains(content, "'default' => env('FILESYSTEM_DISK', 'local')") ||
		strings.Contains(content, "storage_path('app")
}

func laravelSQLitePath(dir string) string {
	env, _ := os.ReadFile(filepath.Join(dir, ".env"))
	for _, line := range strings.Split(string(env), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "DB_DATABASE=") {
			path := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "DB_DATABASE=")), `"'`)
			if strings.HasSuffix(path, ".sqlite") || strings.HasSuffix(path, ".sqlite3") {
				return path
			}
		}
	}
	config, _ := os.ReadFile(filepath.Join(dir, "config/database.php"))
	text := string(config)
	if match := regexp.MustCompile(`storage_path\('([^']+\.sqlite3?)'\)`).FindStringSubmatch(text); len(match) > 1 {
		return "storage/" + match[1]
	}
	if match := regexp.MustCompile(`database_path\('([^']+\.sqlite3?)'\)`).FindStringSubmatch(text); len(match) > 1 {
		return "database/" + match[1]
	}
	return ""
}

func safeMutablePath(path string) bool {
	clean := normalizeRelativeStoragePath(path)
	if clean == "" {
		return false
	}
	first := strings.Split(clean, "/")[0]
	return first == "media" || first == "uploads" || first == "storage" ||
		hasStoragePrefix(clean, "var/media") || hasStoragePrefix(clean, "data/uploads")
}

func firstMutableSegment(path string) string {
	clean := normalizeRelativeStoragePath(path)
	if clean == "" {
		return ""
	}
	for _, prefix := range []string{"var/media", "data/uploads", "storage/app/public", "storage/app"} {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return prefix
		}
	}
	first := strings.Split(clean, "/")[0]
	if first == "media" || first == "uploads" || first == "storage" {
		return first
	}
	return clean
}

func normalizeRelativeStoragePath(path string) string {
	raw := strings.TrimPrefix(strings.TrimSpace(path), "./")
	if raw == "" || strings.HasPrefix(raw, "/") {
		return ""
	}
	clean := filepath.Clean(raw)
	if clean == "." || strings.Contains(clean, "..") {
		return ""
	}
	return clean
}

func genericStoragePathCandidates(content string) []string {
	var candidates []string
	for _, line := range strings.Split(content, "\n") {
		if !genericStorageConfigKeyRegexp.MatchString(line) {
			continue
		}
		for _, match := range quotedStoragePathRegexp.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				candidates = append(candidates, match[1])
			}
		}
	}
	return candidates
}

func hasStoragePrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func smallConfigFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".js", ".mjs", ".cjs", ".ts", ".tsx", ".json", ".rb", ".py", ".php", ".yml", ".yaml":
	default:
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() <= 128*1024
}

func ignoredDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "tmp", "log", "dist", "build":
		return true
	default:
		return false
	}
}
