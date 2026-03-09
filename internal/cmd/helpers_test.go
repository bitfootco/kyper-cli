package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bitfootco/kyper-cli/internal/api"
)

// captureStdout redirects os.Stdout for the duration of fn and returns what was printed.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func testAPIClient(handler http.Handler) (*api.Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return api.NewClientWithHTTP(srv.URL, srv.Client()), srv
}

func TestTailLogSuccess(t *testing.T) {
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "Step 1\n", "cursor": 7, "complete": false,
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "built", "log": "Done\n", "cursor": 12, "complete": true,
			})
		}
	}))
	defer srv.Close()

	status, err := tailLog(client, 1, 0)
	if err != nil {
		t.Fatalf("tailLog failed: %v", err)
	}
	if status != "built" {
		t.Errorf("expected status 'built', got %q", status)
	}
}

func TestTailLogBuildFailed(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "build_failed", "log": "Error: compilation failed\n", "cursor": 25, "complete": true,
		})
	}))
	defer srv.Close()

	status, err := tailLog(client, 1, 0)
	if err != nil {
		t.Fatalf("tailLog returned unexpected error: %v", err)
	}
	if status != "build_failed" {
		t.Errorf("expected status 'build_failed', got %q", status)
	}
}

func TestTailLogStartCursor(t *testing.T) {
	var gotCursor string
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCursor = r.URL.Query().Get("cursor")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "built", "log": "", "cursor": 50, "complete": true,
		})
	}))
	defer srv.Close()

	_, _ = tailLog(client, 1, 42)
	if gotCursor != "42" {
		t.Errorf("expected cursor=42, got %q", gotCursor)
	}
}

func TestSlugifyYAMLName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		slug  string
		want  string
	}{
		{
			name:  "simple name",
			input: "name: My Cool App\nversion: 1.0.0\n",
			slug:  "my-cool-app",
			want:  "name: my-cool-app\nversion: 1.0.0\n",
		},
		{
			name:  "quoted name",
			input: "name: \"My Cool App\"\nversion: 1.0.0\n",
			slug:  "my-cool-app",
			want:  "name: my-cool-app\nversion: 1.0.0\n",
		},
		{
			name:  "already a slug",
			input: "name: my-app\nversion: 1.0.0\n",
			slug:  "my-app",
			want:  "name: my-app\nversion: 1.0.0\n",
		},
		{
			name:  "preserves other fields",
			input: "name: My App\nversion: 1.0.0\ndescription: A great app\ncategory: productivity\n",
			slug:  "my-app",
			want:  "name: my-app\nversion: 1.0.0\ndescription: A great app\ncategory: productivity\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(slugifyYAMLName([]byte(tt.input), tt.slug))
			if got != tt.want {
				t.Errorf("slugifyYAMLName() =\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	t.Run("missing file returns empty map", func(t *testing.T) {
		got := parseEnvFile("/nonexistent/.env")
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("empty file returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".env")
		_ = os.WriteFile(path, []byte(""), 0644)
		got := parseEnvFile(path)
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("valid key=value pairs", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".env")
		content := "MY_KEY=hello\nOTHER_KEY=world\n"
		_ = os.WriteFile(path, []byte(content), 0644)
		got := parseEnvFile(path)
		if got["MY_KEY"] != "hello" {
			t.Errorf("expected MY_KEY=hello, got %q", got["MY_KEY"])
		}
		if got["OTHER_KEY"] != "world" {
			t.Errorf("expected OTHER_KEY=world, got %q", got["OTHER_KEY"])
		}
	})

	t.Run("comments and blank lines are skipped", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".env")
		content := "# this is a comment\n\nMY_KEY=hello\n# another comment\n"
		_ = os.WriteFile(path, []byte(content), 0644)
		got := parseEnvFile(path)
		if len(got) != 1 {
			t.Errorf("expected 1 entry, got %d: %v", len(got), got)
		}
		if got["MY_KEY"] != "hello" {
			t.Errorf("expected MY_KEY=hello, got %q", got["MY_KEY"])
		}
	})

	t.Run("lines without = are skipped", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".env")
		content := "NOEQUALS\nMY_KEY=hello\n"
		_ = os.WriteFile(path, []byte(content), 0644)
		got := parseEnvFile(path)
		if len(got) != 1 {
			t.Errorf("expected 1 entry, got %d: %v", len(got), got)
		}
	})

	t.Run("values with = in them are preserved", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".env")
		content := "MY_KEY=val=ue\n"
		_ = os.WriteFile(path, []byte(content), 0644)
		got := parseEnvFile(path)
		if got["MY_KEY"] != "val=ue" {
			t.Errorf("expected MY_KEY=val=ue, got %q", got["MY_KEY"])
		}
	})
}

func TestSlugFromTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My App", "my-app"},
		{"my-app", "my-app"},
		{"My Cool App!!!", "my-cool-app"},
		{"  hello  world  ", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"with_underscores", "with-underscores"},
		{"simple", "simple"},
		{"123-numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugFromTitle(tt.input)
			if got != tt.want {
				t.Errorf("slugFromTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- tailLogMilestones ---

func TestTailLogMilestonesSuccess(t *testing.T) {
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "$ docker build -t img .\nStep 1/5\n", "cursor": 30, "complete": false,
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "built", "log": "Successfully built\n", "cursor": 50, "complete": true,
			})
		}
	}))
	defer srv.Close()

	out := captureStdout(func() {
		status, err := tailLogMilestones(client, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "built" {
			t.Errorf("expected status 'built', got %q", status)
		}
	})
	if !strings.Contains(out, "Building image...") {
		t.Errorf("expected 'Building image...' milestone in output, got:\n%s", out)
	}
}

func TestTailLogMilestonesBuildFailed(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "build_failed", "log": "Error: could not compile\n", "cursor": 25, "complete": true,
		})
	}))
	defer srv.Close()

	status, err := tailLogMilestones(client, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "build_failed" {
		t.Errorf("expected status 'build_failed', got %q", status)
	}
}

func TestTailLogMilestonesAllMilestonesDetected(t *testing.T) {
	// Deliver all three markers across separate poll responses.
	responses := []map[string]interface{}{
		{"status": "building", "log": "$ docker build -t img .\n", "cursor": 23, "complete": false},
		{"status": "building", "log": "$ docker push registry/img\n", "cursor": 50, "complete": false},
		{"status": "building", "log": "--- Security scan (Trivy) ---\n", "cursor": 80, "complete": false},
		{"status": "built", "log": "done\n", "cursor": 85, "complete": true},
	}
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&calls, 1)) - 1
		if n >= len(responses) {
			n = len(responses) - 1
		}
		_ = json.NewEncoder(w).Encode(responses[n])
	}))
	defer srv.Close()

	out := captureStdout(func() {
		status, err := tailLogMilestones(client, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "built" {
			t.Errorf("expected 'built', got %q", status)
		}
	})
	for _, label := range []string{"Building image...", "Pushing to registry...", "Running security scan..."} {
		if !strings.Contains(out, label) {
			t.Errorf("expected milestone %q in output, got:\n%s", label, out)
		}
	}
}

func TestTailLogMilestonesMilestonesNotRepeated(t *testing.T) {
	// Same marker appears in two separate chunks — label must only print once.
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "$ docker build -t img .\n", "cursor": 23, "complete": false,
			})
		case 2:
			// Second chunk repeats the marker (shouldn't happen in practice but guard against it).
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "$ docker build (cache)\n", "cursor": 46, "complete": false,
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "built", "log": "done\n", "cursor": 51, "complete": true,
			})
		}
	}))
	defer srv.Close()

	out := captureStdout(func() {
		_, _ = tailLogMilestones(client, 1)
	})
	count := strings.Count(out, "Building image...")
	if count != 1 {
		t.Errorf("expected 'Building image...' exactly once, got %d times in:\n%s", count, out)
	}
}

func TestTailLogMilestonesCursorPropagated(t *testing.T) {
	var cursors []string
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "step\n", "cursor": 42, "complete": false,
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "built", "log": "done\n", "cursor": 47, "complete": true,
			})
		}
	}))
	defer srv.Close()

	captureStdout(func() {
		_, _ = tailLogMilestones(client, 1)
	})
	if len(cursors) < 2 {
		t.Fatalf("expected at least 2 polls, got %d", len(cursors))
	}
	if cursors[0] != "0" {
		t.Errorf("first cursor should be 0, got %q", cursors[0])
	}
	if cursors[1] != "42" {
		t.Errorf("second cursor should be 42, got %q", cursors[1])
	}
}

// --- printBuildFailureContext ---

func TestPrintBuildFailureContextTrivyPath(t *testing.T) {
	log := "lots of docker output\n" +
		"layers pulled\n" +
		"Scan result: FAILED\nCVE-2024-1234 HIGH\nCVE-2024-5678 CRITICAL\n"

	out := captureStdout(func() { printBuildFailureContext(log) })

	if strings.Contains(out, "lots of docker output") {
		t.Error("should not print output before 'Scan result: FAILED'")
	}
	if !strings.Contains(out, "Scan result: FAILED") {
		t.Error("expected 'Scan result: FAILED' in output")
	}
	if !strings.Contains(out, "CVE-2024-5678 CRITICAL") {
		t.Error("expected CVE detail in output")
	}
}

func TestPrintBuildFailureContextDockerPathLast25Lines(t *testing.T) {
	// Build a 30-line log; only the last 25 should appear.
	var lines []string
	for i := 1; i <= 30; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	log := strings.Join(lines, "\n") + "\n"

	out := captureStdout(func() { printBuildFailureContext(log) })

	if strings.Contains(out, "line 1\n") || strings.Contains(out, "line 5\n") {
		t.Error("should not include lines 1–5 (before the last-25 window)")
	}
	if !strings.Contains(out, "line 6") {
		t.Error("expected line 6 (start of last-25 window) to be present")
	}
	if !strings.Contains(out, "line 30") {
		t.Error("expected line 30 (last line) to be present")
	}
}

func TestPrintBuildFailureContextDockerPathShortLog(t *testing.T) {
	// Fewer than 25 lines — should print the whole thing, no panic.
	log := "error: bad FROM\nno such image\n"
	out := captureStdout(func() { printBuildFailureContext(log) })
	if !strings.Contains(out, "error: bad FROM") {
		t.Error("expected full short log in output")
	}
}
