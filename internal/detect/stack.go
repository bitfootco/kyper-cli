package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type StackResult struct {
	Name   string
	Source string
}

// DetectStack identifies the application framework/stack from project files.
func DetectStack(dir string) []StackResult {
	var results []StackResult

	checks := []struct {
		path   string
		stack  string
		source string
	}{
		{"config/application.rb", "rails", "config/application.rb"},
		{"manage.py", "django", "manage.py"},
		{"artisan", "laravel", "artisan"},
		{"go.mod", "go", "go.mod"},
	}

	for _, c := range checks {
		if fileExists(filepath.Join(dir, c.path)) {
			results = append(results, StackResult{Name: c.stack, Source: c.source})
		}
	}

	// Check package.json for Node frameworks
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		data, err := os.ReadFile(pkgPath)
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				deps := mergeJSONMaps(pkg, "dependencies", "devDependencies")
				if _, ok := deps["next"]; ok {
					results = append(results, StackResult{Name: "next", Source: "package.json"})
				} else if _, ok := deps["@nestjs/core"]; ok {
					results = append(results, StackResult{Name: "nest", Source: "package.json"})
				} else if _, ok := deps["express"]; ok {
					results = append(results, StackResult{Name: "express", Source: "package.json"})
				} else if _, ok := deps["koa"]; ok {
					results = append(results, StackResult{Name: "koa", Source: "package.json"})
				}
			}
		}
	}

	// Check for Prisma
	prismaLocations := []string{
		"prisma/schema.prisma",
		"schema.prisma",
	}
	for _, loc := range prismaLocations {
		if fileExists(filepath.Join(dir, loc)) {
			results = append(results, StackResult{Name: "prisma", Source: loc})
			break
		}
	}

	return results
}

// StackNames returns just the stack names from results.
func StackNames(results []StackResult) []string {
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Name
	}
	return names
}

func mergeJSONMaps(pkg map[string]interface{}, keys ...string) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, key := range keys {
		if m, ok := pkg[key].(map[string]interface{}); ok {
			for k, v := range m {
				merged[k] = v
			}
		}
	}
	return merged
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var result []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}
