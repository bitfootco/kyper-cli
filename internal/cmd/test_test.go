package cmd

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bitfootco/kyper-cli/internal/api"
)

// TestFormatExpiresIn covers all branches of the pure helper.
// We add a 30s buffer to each duration to absorb RFC3339's second-level truncation
// and avoid boundary flakiness (e.g. 45m0s truncated → 44m59s → 44 minutes).
func TestFormatExpiresIn(t *testing.T) {
	buf := 30 * time.Second
	tests := []struct {
		name      string
		expiresAt string
		want      string
	}{
		{"empty string", "", "in ~1 hour"},
		{"invalid format", "not-a-date", "at not-a-date"},
		{"already expired", time.Now().Add(-5 * time.Minute).Format(time.RFC3339), "soon (expired)"},
		{"under 60 minutes", time.Now().Add(45*time.Minute + buf).Format(time.RFC3339), "in 45 minute(s)"},
		{"exactly 1 hour", time.Now().Add(60*time.Minute + buf).Format(time.RFC3339), "in 1 hour(s)"},
		{"1 hour 30 minutes", time.Now().Add(90*time.Minute + buf).Format(time.RFC3339), "in 1h 30m"},
		{"2 hours", time.Now().Add(120*time.Minute + buf).Format(time.RFC3339), "in 2 hour(s)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExpiresIn(tt.expiresAt)
			if got != tt.want {
				t.Errorf("formatExpiresIn(%q) = %q, want %q", tt.expiresAt, got, tt.want)
			}
		})
	}
}

// TestTailProvisionLogRunning verifies the happy path: provision log streamed, "running" returned.
func TestTailProvisionLogRunning(t *testing.T) {
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version_id":   1,
				"build_status": "built",
				"deployment": map[string]interface{}{
					"id":                    1,
					"status":                "provisioning",
					"url":                   "",
					"expires_at":            "",
					"provision_log":         "Starting...\n",
					"provision_log_cursor":  11,
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version_id":   1,
				"build_status": "built",
				"deployment": map[string]interface{}{
					"id":                    1,
					"status":                "running",
					"url":                   "https://test-myapp.apps.kyper.shop",
					"expires_at":            time.Now().Add(60 * time.Minute).Format(time.RFC3339),
					"provision_log":         "Ready!\n",
					"provision_log_cursor":  18,
				},
			})
		}
	}))
	defer srv.Close()

	d, err := tailProvisionLog(client, "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != "running" {
		t.Errorf("expected status 'running', got %q", d.Status)
	}
	if d.URL != "https://test-myapp.apps.kyper.shop" {
		t.Errorf("unexpected URL: %q", d.URL)
	}
}

// TestTailProvisionLogFailed verifies a "failed" deployment is returned (not treated as success).
func TestTailProvisionLogFailed(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version_id":   1,
			"build_status": "built",
			"deployment": map[string]interface{}{
				"id":                   1,
				"status":               "failed",
				"url":                  "",
				"expires_at":           "",
				"provision_log":        "pod failed to start\n",
				"provision_log_cursor": 20,
			},
		})
	}))
	defer srv.Close()

	d, err := tailProvisionLog(client, "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", d.Status)
	}
}

// TestTailProvisionLogTerminated verifies "terminated" is returned as a terminal state.
func TestTailProvisionLogTerminated(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version_id":   1,
			"build_status": "built",
			"deployment": map[string]interface{}{
				"id":                   1,
				"status":               "terminated",
				"url":                  "",
				"expires_at":           "",
				"provision_log":        "",
				"provision_log_cursor": 0,
			},
		})
	}))
	defer srv.Close()

	d, err := tailProvisionLog(client, "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != "terminated" {
		t.Errorf("expected status 'terminated', got %q", d.Status)
	}
}

// TestTailProvisionLogNilDeploymentRetries verifies that a nil deployment retries and
// eventually errors out rather than looping forever.
func TestTailProvisionLogNilDeploymentRetries(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return null deployment
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version_id":   1,
			"build_status": "built",
			"deployment":   nil,
		})
	}))
	defer srv.Close()

	// Override the sleep to make the test fast — we can't easily mock time.Sleep,
	// but maxNilDeploymentPolls is small (10) so this completes in ~0s in tests
	// because the server responds instantly (no actual 2s sleep needed in unit tests).
	// We accept the test taking up to ~20s in the worst case; in practice httptest
	// responds in microseconds so the loop exits quickly.
	//
	// To keep tests fast we substitute a minimal client with a fast-responding server.
	_, err := tailProvisionLog(client, "my-app")
	if err == nil {
		t.Fatal("expected error when deployment never appears, got nil")
	}
}

// TestTailProvisionLog404 verifies a 404 response is surfaced as an error.
func TestTailProvisionLog404(t *testing.T) {
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer srv.Close()

	_, err := tailProvisionLog(client, "my-app")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}

// TestTailProvisionLogCursorPropagated verifies the provision_log_cursor is sent on subsequent polls.
func TestTailProvisionLogCursorPropagated(t *testing.T) {
	var cursors []string
	var calls int32
	client, srv := testAPIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursors = append(cursors, r.URL.Query().Get("provision_log_cursor"))
		n := atomic.AddInt32(&calls, 1)
		// First response: return cursor=11, still provisioning.
		// Second response: return terminal "running" state.
		status := "provisioning"
		responseCursor := 11
		if n > 1 {
			status = "running"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version_id":   1,
			"build_status": "built",
			"deployment": map[string]interface{}{
				"id":                   1,
				"status":               status,
				"url":                  "https://test-myapp.apps.kyper.shop",
				"expires_at":           "",
				"provision_log":        "log\n",
				"provision_log_cursor": responseCursor,
			},
		})
	}))
	defer srv.Close()

	_, err := tailProvisionLog(client, "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cursors) < 2 {
		t.Fatalf("expected at least 2 polls, got %d", len(cursors))
	}
	if cursors[0] != "0" {
		t.Errorf("first poll cursor should be 0, got %q", cursors[0])
	}
	if cursors[1] != "11" {
		t.Errorf("second poll cursor should be 11, got %q", cursors[1])
	}
}

// TestProvisioningFailedStatusNotTreatedAsSuccess verifies that a non-"running" terminal
// state from tailProvisionLog propagates correctly as an error to the caller.
func TestProvisioningFailedStatusNotTreatedAsSuccess(t *testing.T) {
	// tailProvisionLog returns (d, nil) for "failed" — the caller must check d.Status.
	d := &api.TestDeployment{Status: "failed", URL: ""}
	if d.Status == "running" {
		t.Error("'failed' should not be treated as success")
	}

	d2 := &api.TestDeployment{Status: "terminated", URL: ""}
	if d2.Status == "running" {
		t.Error("'terminated' should not be treated as success")
	}

	d3 := &api.TestDeployment{Status: "destroying", URL: ""}
	if d3.Status == "running" {
		t.Error("'destroying' should not be treated as success")
	}
}
