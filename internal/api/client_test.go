package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := NewClientWithHTTP(srv.URL, srv.Client())
	return client, srv
}

func TestGetMe(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(User{ID: 1, Email: "dev@test.com", Role: "developer"})
	}))
	defer srv.Close()

	user, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}
	if user.Email != "dev@test.com" {
		t.Errorf("expected dev@test.com, got %q", user.Email)
	}
}

func TestDeviceAuthorize(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(DeviceGrant{Code: "abc-123", VerificationURI: "https://kyper.shop/device?code=abc-123"})
	}))
	defer srv.Close()

	grant, err := client.DeviceAuthorize()
	if err != nil {
		t.Fatalf("DeviceAuthorize failed: %v", err)
	}
	if grant.Code != "abc-123" {
		t.Errorf("expected code 'abc-123', got %q", grant.Code)
	}
}

func TestDeviceToken(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "abc-123" {
			t.Errorf("expected code 'abc-123', got %q", code)
		}
		_ = json.NewEncoder(w).Encode(TokenResponse{APIToken: "tok_123"})
	}))
	defer srv.Close()

	resp, err := client.DeviceToken("abc-123")
	if err != nil {
		t.Fatalf("DeviceToken failed: %v", err)
	}
	if resp.APIToken != "tok_123" {
		t.Errorf("expected token 'tok_123', got %q", resp.APIToken)
	}
}

func TestGetAppStatus(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apps/my-app/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(AppStatus{
			App:    "my-app",
			Status: "active",
			LatestVersion: &VersionInfo{
				ID:      1,
				Version: "1.0.0",
				Status:  "published",
			},
		})
	}))
	defer srv.Close()

	status, err := client.GetAppStatus("my-app")
	if err != nil {
		t.Fatalf("GetAppStatus failed: %v", err)
	}
	if status.App != "my-app" {
		t.Errorf("unexpected app slug: %q", status.App)
	}
	if status.LatestVersion.Version != "1.0.0" {
		t.Errorf("unexpected version: %q", status.LatestVersion.Version)
	}
}

func TestCreateVersion(t *testing.T) {
	var gotKyperYml string
	var gotZipName string

	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart content type, got %q", r.Header.Get("Content-Type"))
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm failed: %v", err)
			return
		}
		gotKyperYml = r.FormValue("kyper_yml")
		file, header, _ := r.FormFile("source_zip")
		if file != nil {
			gotZipName = header.Filename
			_ = file.Close()
		}

		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(VersionResponse{ID: 42, Version: "1.0.0", Status: "pending"})
	}))
	defer srv.Close()

	// Create a temp zip file
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "source.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip-content"), 0644); err != nil {
		t.Fatal(err)
	}

	vr, err := client.CreateVersion("my-app", "name: my-app\n", zipPath)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}
	if vr.ID != 42 {
		t.Errorf("expected version ID 42, got %d", vr.ID)
	}
	if gotKyperYml != "name: my-app\n" {
		t.Errorf("unexpected kyper_yml: %q", gotKyperYml)
	}
	if gotZipName != "source.zip" {
		t.Errorf("unexpected zip name: %q", gotZipName)
	}
}

func TestCreateApp(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/apps" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		if _, ok := req["app"]; !ok {
			t.Error("expected 'app' key in request body")
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(App{Slug: "my-app", Title: "My App"})
	}))
	defer srv.Close()

	app, err := client.CreateApp(map[string]interface{}{"title": "My App"})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	if app.Slug != "my-app" {
		t.Errorf("unexpected slug: %q", app.Slug)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string][]string{"errors": {"name is required", "version is invalid"}})
	}))
	defer srv.Close()

	_, err := client.GetMe()
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("expected status 422, got %d", apiErr.StatusCode)
	}
	if len(apiErr.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(apiErr.Messages))
	}
}

func TestAPIErrorNotFound(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer srv.Close()

	_, err := client.GetApp("nonexistent")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true")
	}
}

func TestGetBuildLog(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/versions/42/build_log" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("cursor") != "10" {
			t.Errorf("unexpected cursor: %s", r.URL.Query().Get("cursor"))
		}
		_ = json.NewEncoder(w).Encode(BuildLog{Status: "building", Log: "Step 1...\n", Cursor: 20, Complete: false})
	}))
	defer srv.Close()

	log, err := client.GetBuildLog(42, 10)
	if err != nil {
		t.Fatalf("GetBuildLog failed: %v", err)
	}
	if log.Log != "Step 1...\n" {
		t.Errorf("unexpected log: %q", log.Log)
	}
	if log.Cursor != 20 {
		t.Errorf("expected cursor 20, got %d", log.Cursor)
	}
}
