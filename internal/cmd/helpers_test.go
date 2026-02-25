package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bitfootco/kyper-cli/internal/api"
)

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
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "building", "log": "Step 1\n", "cursor": 7, "complete": false,
			})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "built", "log": "", "cursor": 50, "complete": true,
		})
	}))
	defer srv.Close()

	tailLog(client, 1, 42)
	if gotCursor != "42" {
		t.Errorf("expected cursor=42, got %q", gotCursor)
	}
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
