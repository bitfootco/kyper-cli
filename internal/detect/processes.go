package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type ProcessResult struct {
	Name    string
	Command string
	Source  string
}

// DetectProcesses scans a directory for process definitions.
func DetectProcesses(dir string) []ProcessResult {
	var results []ProcessResult

	// Check Procfile
	procfile := filepath.Join(dir, "Procfile")
	if lines := readLines(procfile); len(lines) > 0 {
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				cmd := strings.TrimSpace(parts[1])
				results = append(results, ProcessResult{Name: name, Command: cmd, Source: "Procfile"})
			}
		}
	}

	// Check Dockerfile for CMD
	dockerfile := filepath.Join(dir, "Dockerfile")
	if data, err := os.ReadFile(dockerfile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(line), "CMD ") {
				cmd := strings.TrimSpace(line[4:])
				// Only add web if not already found
				hasWeb := false
				for _, r := range results {
					if r.Name == "web" {
						hasWeb = true
						break
					}
				}
				if !hasWeb {
					results = append(results, ProcessResult{Name: "web", Command: cmd, Source: "Dockerfile"})
				}
				break
			}
		}
	}

	// Check package.json start script
	pkgPath := filepath.Join(dir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(data, &pkg) == nil {
			if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
				if start, ok := scripts["start"].(string); ok {
					hasWeb := false
					for _, r := range results {
						if r.Name == "web" {
							hasWeb = true
							break
						}
					}
					if !hasWeb {
						results = append(results, ProcessResult{Name: "web", Command: start, Source: "package.json"})
					}
				}
			}
		}
	}

	return results
}
