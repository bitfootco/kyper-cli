package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// VersionSuggestion holds a version hint from a lockfile.
type VersionSuggestion struct {
	Dep     string
	Version string
	Source  string
}

// SuggestDepVersions reads lockfiles to suggest versions for detected deps.
func SuggestDepVersions(dir string, deps []DepResult) []VersionSuggestion {
	var suggestions []VersionSuggestion

	lockSources := []struct {
		file    string
		scanner func(path string, deps []DepResult) []VersionSuggestion
	}{
		{"Gemfile.lock", scanGemfileLock},
		{"package-lock.json", scanPackageLock},
	}

	for _, ls := range lockSources {
		path := filepath.Join(dir, ls.file)
		if fileExists(path) {
			suggestions = append(suggestions, ls.scanner(path, deps)...)
		}
	}

	return suggestions
}

var gemVersionRegexp = regexp.MustCompile(`^\s+(\S+)\s+\(([^)]+)\)`)

func scanGemfileLock(path string, deps []DepResult) []VersionSuggestion {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Map gem names back to dep names
	gemToDep := map[string]string{
		"pg":              "postgres",
		"mysql2":          "mysql",
		"redis":           "redis",
		"elasticsearch":   "elasticsearch",
		"opensearch-ruby": "opensearch",
	}

	wantDeps := make(map[string]bool)
	for _, d := range deps {
		wantDeps[d.Name] = true
	}

	var results []VersionSuggestion
	for _, line := range strings.Split(string(data), "\n") {
		matches := gemVersionRegexp.FindStringSubmatch(line)
		if len(matches) >= 3 {
			gemName := matches[1]
			gemVersion := matches[2]

			if depName, ok := gemToDep[gemName]; ok && wantDeps[depName] {
				majorVersion := strings.Split(gemVersion, ".")[0]
				results = append(results, VersionSuggestion{
					Dep:     depName,
					Version: majorVersion,
					Source:  "Gemfile.lock",
				})
			}
		}
	}

	return results
}

func scanPackageLock(path string, deps []DepResult) []VersionSuggestion {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	pkgToDep := map[string]string{
		"pg":                                     "postgres",
		"prisma":                                 "postgres",
		"mysql2":                                 "mysql",
		"redis":                                  "redis",
		"ioredis":                                "redis",
		"@elastic/elasticsearch":                 "elasticsearch",
		"@opensearch-project/opensearch":         "opensearch",
	}

	wantDeps := make(map[string]bool)
	for _, d := range deps {
		wantDeps[d.Name] = true
	}

	// Simple approach: look for version strings near package names
	content := string(data)
	var results []VersionSuggestion
	for pkgName, depName := range pkgToDep {
		if !wantDeps[depName] {
			continue
		}
		// Look for "pkgName": { "version": "X.Y.Z" } pattern
		idx := strings.Index(content, "\""+pkgName+"\"")
		if idx < 0 {
			continue
		}
		remaining := content[idx:]
		versionIdx := strings.Index(remaining, "\"version\"")
		if versionIdx < 0 || versionIdx > 200 {
			continue
		}
		versionStr := remaining[versionIdx:]
		// Extract version value
		re := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
		if m := re.FindStringSubmatch(versionStr); len(m) >= 2 {
			majorVersion := strings.Split(m[1], ".")[0]
			results = append(results, VersionSuggestion{
				Dep:     depName,
				Version: majorVersion,
				Source:  "package-lock.json",
			})
		}
	}

	return results
}
