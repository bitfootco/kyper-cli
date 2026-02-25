package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"0.0.1", 0, 0, 1, false},
		{"10.20.30", 10, 20, 30, false},
		{"1.2", 0, 0, 0, true},
		{"1.2.3.4", 0, 0, 0, true},
		{"a.b.c", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if major != tt.wantMajor || minor != tt.wantMinor || patch != tt.wantPatch {
				t.Errorf("parseVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestReplaceVersion(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		oldVer     string
		newVer     string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "patch bump",
			input:      "name: my-app\nversion: 1.2.3\ncategory: productivity\n",
			oldVer:     "1.2.3",
			newVer:     "1.2.4",
			wantOutput: "name: my-app\nversion: 1.2.4\ncategory: productivity\n",
		},
		{
			name:       "minor bump preserves surrounding comments",
			input:      "# comment\nversion: 0.0.1\n# another\n",
			oldVer:     "0.0.1",
			newVer:     "0.1.0",
			wantOutput: "# comment\nversion: 0.1.0\n# another\n",
		},
		{
			name:    "version not found returns error",
			input:   "version: 2.0.0\n",
			oldVer:  "1.0.0",
			newVer:  "1.0.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := replaceVersion([]byte(tt.input), tt.oldVer, tt.newVer)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(result) != tt.wantOutput {
				t.Errorf("replaceVersion output = %q, want %q", string(result), tt.wantOutput)
			}
		})
	}
}

const tagTestKyperYML = `name: test-app
version: 1.0.0
description: A test app
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: bin/start
pricing:
  one_time: 9.99
`

func setupTagTest(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.WriteFile("kyper.yml", []byte(tagTestKyperYML), 0644); err != nil {
		t.Fatalf("writing kyper.yml: %v", err)
	}
}

func saveTagState(t *testing.T) {
	t.Helper()
	origBump := bumpFlag
	origJSON := jsonOutput
	t.Cleanup(func() {
		bumpFlag = origBump
		jsonOutput = origJSON
	})
}

func TestRunTagBump(t *testing.T) {
	tests := []struct {
		bump        string
		wantVersion string
	}{
		{"patch", "1.0.1"},
		{"minor", "1.1.0"},
		{"major", "2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.bump, func(t *testing.T) {
			setupTagTest(t)
			saveTagState(t)

			bumpFlag = tt.bump
			jsonOutput = false

			if err := runTag(nil, nil); err != nil {
				t.Fatalf("runTag error: %v", err)
			}

			content, err := os.ReadFile("kyper.yml")
			if err != nil {
				t.Fatalf("reading kyper.yml: %v", err)
			}
			if !strings.Contains(string(content), "version: "+tt.wantVersion) {
				t.Errorf("kyper.yml content = %q, want version %s", string(content), tt.wantVersion)
			}
		})
	}
}

func TestRunTagInvalidBump(t *testing.T) {
	setupTagTest(t)
	saveTagState(t)

	bumpFlag = "invalid"
	jsonOutput = false

	err := runTag(nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid bump, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --bump value") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTagMalformedVersion(t *testing.T) {
	setupTagTest(t)
	saveTagState(t)

	content := strings.Replace(tagTestKyperYML, "version: 1.0.0", "version: notvalid", 1)
	if err := os.WriteFile("kyper.yml", []byte(content), 0644); err != nil {
		t.Fatalf("writing kyper.yml: %v", err)
	}

	bumpFlag = "patch"
	jsonOutput = false

	err := runTag(nil, nil)
	if err == nil {
		t.Fatal("expected error for malformed version, got nil")
	}
}

func TestRunTagJSONWithBump(t *testing.T) {
	setupTagTest(t)
	saveTagState(t)

	bumpFlag = "patch"
	jsonOutput = true

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	runErr := runTag(nil, nil)
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if runErr != nil {
		t.Fatalf("runTag error: %v", runErr)
	}

	var result map[string]string
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("parsing JSON output: %v\noutput: %q", jsonErr, buf.String())
	}
	if result["previous_version"] != "1.0.0" {
		t.Errorf("previous_version = %q, want 1.0.0", result["previous_version"])
	}
	if result["new_version"] != "1.0.1" {
		t.Errorf("new_version = %q, want 1.0.1", result["new_version"])
	}
}

func TestRunTagJSONWithoutBump(t *testing.T) {
	setupTagTest(t)
	saveTagState(t)

	bumpFlag = ""
	jsonOutput = true

	err := runTag(nil, nil)
	if err == nil {
		t.Fatal("expected error when --json without --bump, got nil")
	}
	if !strings.Contains(err.Error(), "--bump") {
		t.Errorf("expected error to mention --bump, got: %v", err)
	}
}
