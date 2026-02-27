package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type DepResult struct {
	Name   string
	Source string
}

// Mapping from library/package names to infrastructure deps
var gemfileMappings = map[string]string{
	"pg":              "postgres",
	"mysql2":          "mysql",
	"redis":           "redis",
	"elasticsearch":   "elasticsearch",
	"opensearch-ruby": "opensearch",
	"aws-sdk-s3":      "s3",
	"fog-aws":         "s3",
}

var packageJSONMappings = map[string]string{
	"pg":                                     "postgres",
	"prisma":                                 "postgres",
	"mysql2":                                 "mysql",
	"redis":                                  "redis",
	"ioredis":                                "redis",
	"@elastic/elasticsearch":                 "elasticsearch",
	"@opensearch-project/opensearch":         "opensearch",
	"@aws-sdk/client-s3":                     "s3",
	"aws-sdk":                                "s3",
}

var pythonMappings = map[string]string{
	"psycopg2":        "postgres",
	"psycopg2-binary": "postgres",
	"psycopg":         "postgres",
	"mysqlclient":     "mysql",
	"PyMySQL":         "mysql",
	"pymysql":         "mysql",
	"redis":           "redis",
	"elasticsearch":   "elasticsearch",
	"opensearch-py":   "opensearch",
	"boto3":           "s3",
	"botocore":        "s3",
}

// DetectDeps scans project files for infrastructure dependencies.
func DetectDeps(dir string) []DepResult {
	seen := make(map[string]bool)
	var results []DepResult

	addDep := func(name, source string) {
		if !seen[name] {
			seen[name] = true
			results = append(results, DepResult{Name: name, Source: source})
		}
	}

	// docker-compose.yml
	detectDockerComposeDeps(filepath.Join(dir, "docker-compose.yml"), addDep)

	// Gemfile
	detectGemfileDeps(filepath.Join(dir, "Gemfile"), addDep)

	// package.json
	detectPackageJSONDeps(filepath.Join(dir, "package.json"), addDep)

	// requirements.txt
	detectPythonDeps(filepath.Join(dir, "requirements.txt"), "requirements.txt", addDep)

	// Pipfile
	detectPipfileDeps(filepath.Join(dir, "Pipfile"), addDep)

	return results
}

func detectDockerComposeDeps(path string, addDep func(string, string)) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := strings.ToLower(string(data))

	depKeywords := map[string]string{
		"postgres":      "postgres",
		"mysql":         "mysql",
		"redis":         "redis",
		"elasticsearch": "elasticsearch",
		"opensearch":    "opensearch",
		"seaweedfs":     "s3",
		"minio":         "s3",
	}

	for keyword, dep := range depKeywords {
		if strings.Contains(content, keyword) {
			addDep(dep, "docker-compose.yml")
		}
	}
}

func detectGemfileDeps(path string, addDep func(string, string)) {
	lines := readLines(path)
	for _, line := range lines {
		// Match gem 'name' or gem "name"
		for lib, dep := range gemfileMappings {
			if strings.Contains(line, "'"+lib+"'") || strings.Contains(line, "\""+lib+"\"") {
				addDep(dep, "Gemfile")
			}
		}
	}
}

func detectPackageJSONDeps(path string, addDep func(string, string)) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var pkg map[string]interface{}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}

	allDeps := mergeJSONMaps(pkg, "dependencies", "devDependencies")
	for lib, dep := range packageJSONMappings {
		if _, ok := allDeps[lib]; ok {
			addDep(dep, "package.json")
		}
	}
}

func detectPythonDeps(path, source string, addDep func(string, string)) {
	lines := readLines(path)
	for _, line := range lines {
		// Strip version specifiers
		pkgName := strings.Split(line, "==")[0]
		pkgName = strings.Split(pkgName, ">=")[0]
		pkgName = strings.Split(pkgName, "<=")[0]
		pkgName = strings.Split(pkgName, "~=")[0]
		pkgName = strings.TrimSpace(pkgName)

		if dep, ok := pythonMappings[pkgName]; ok {
			addDep(dep, source)
		}
	}
}

func detectPipfileDeps(path string, addDep func(string, string)) {
	lines := readLines(path)
	for _, line := range lines {
		// Extract the package name (TOML key format: name = "version")
		key := strings.SplitN(line, "=", 2)[0]
		key = strings.TrimSpace(key)
		key = strings.Trim(key, "\"'")
		if dep, ok := pythonMappings[key]; ok {
			addDep(dep, "Pipfile")
		}
	}
}
