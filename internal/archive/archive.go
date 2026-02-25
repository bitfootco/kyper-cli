package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var defaultExcludes = []string{
	".git/",
	".git",
	"*.log",
	"tmp/",
	"node_modules/",
}

// Create builds a zip archive from the given directory, respecting
// default exclude patterns and .kyperignore rules.
func Create(dir, outputPath string) error {
	ignorePatterns := loadIgnorePatterns(dir)
	allPatterns := make([]string, 0, len(defaultExcludes)+len(ignorePatterns))
	allPatterns = append(allPatterns, defaultExcludes...)
	allPatterns = append(allPatterns, ignorePatterns...)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = outFile.Close() }()

	w := zip.NewWriter(outFile)
	defer func() { _ = w.Close() }()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Check exclusion patterns
		if shouldExclude(relPath, info.IsDir(), allPatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories themselves (they're implicit in zip)
		if info.IsDir() {
			return nil
		}

		// Don't include the output file itself
		absOut, _ := filepath.Abs(outputPath)
		absPath, _ := filepath.Abs(path)
		if absOut == absPath {
			return nil
		}

		// Add file to zip
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		_, err = io.Copy(writer, file)
		return err
	})
}

func loadIgnorePatterns(dir string) []string {
	path := filepath.Join(dir, ".kyperignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

func shouldExclude(relPath string, isDir bool, patterns []string) bool {
	name := filepath.Base(relPath)

	for _, pattern := range patterns {
		// Directory pattern (ends with /)
		if strings.HasSuffix(pattern, "/") {
			dirName := strings.TrimSuffix(pattern, "/")
			if isDir && (name == dirName || relPath == dirName) {
				return true
			}
			// Also check if any path component matches
			parts := strings.Split(relPath, string(filepath.Separator))
			for _, p := range parts {
				if p == dirName {
					return true
				}
			}
			continue
		}

		// Exact match
		if name == pattern || relPath == pattern {
			return true
		}

		// Glob match on filename
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}

		// Glob match on full relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
	}

	return false
}
