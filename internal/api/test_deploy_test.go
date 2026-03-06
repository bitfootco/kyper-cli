package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTestDeploy(t *testing.T) {
	var gotKyperYml string
	var gotZipName string

	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps/my-app/test_deploy" {
			t.Errorf("unexpected path: %s", r.URL.Path)
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
		_ = json.NewEncoder(w).Encode(TestDeployResponse{
			VersionID: 99,
			Message:   "Test build queued.",
			Warnings:  []string{"no on_deploy hook set"},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "source.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}

	resp, err := client.CreateTestDeploy("my-app", "name: my-app\n", zipPath, nil)
	if err != nil {
		t.Fatalf("CreateTestDeploy failed: %v", err)
	}
	if resp.VersionID != 99 {
		t.Errorf("expected VersionID 99, got %d", resp.VersionID)
	}
	if resp.Message != "Test build queued." {
		t.Errorf("unexpected message: %q", resp.Message)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0] != "no on_deploy hook set" {
		t.Errorf("unexpected warnings: %v", resp.Warnings)
	}
	if gotKyperYml != "name: my-app\n" {
		t.Errorf("unexpected kyper_yml: %q", gotKyperYml)
	}
	if gotZipName != "source.zip" {
		t.Errorf("unexpected zip name: %q", gotZipName)
	}
}

func TestCreateTestDeployError(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "3 daily builds reached"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "source.zip")
	_ = os.WriteFile(zipPath, []byte("fake"), 0644)

	_, err := client.CreateTestDeploy("my-app", "name: my-app\n", zipPath, nil)
	if err == nil {
		t.Fatal("expected error on 429")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
}

func TestCreateTestDeployWithEnvVars(t *testing.T) {
	var gotEnvVars string

	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm failed: %v", err)
			return
		}
		gotEnvVars = r.FormValue("env_vars")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(TestDeployResponse{VersionID: 1, Message: "queued."})
	}))
	defer srv.Close()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "source.zip")
	_ = os.WriteFile(zipPath, []byte("fake"), 0644)

	envVars := map[string]string{"MY_KEY": "hello", "OTHER": "world"}
	_, err := client.CreateTestDeploy("my-app", "name: my-app\n", zipPath, envVars)
	if err != nil {
		t.Fatalf("CreateTestDeploy failed: %v", err)
	}
	if gotEnvVars == "" {
		t.Fatal("expected env_vars field to be set")
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(gotEnvVars), &parsed); err != nil {
		t.Fatalf("env_vars not valid JSON: %v", err)
	}
	if parsed["MY_KEY"] != "hello" {
		t.Errorf("expected MY_KEY=hello, got %q", parsed["MY_KEY"])
	}
}

func TestGetTestDeploy(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apps/my-app/test_deploy" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("provision_log_cursor") != "42" {
			t.Errorf("unexpected cursor: %q", r.URL.Query().Get("provision_log_cursor"))
		}
		_ = json.NewEncoder(w).Encode(TestDeployStatus{
			VersionID:   99,
			BuildStatus: "built",
			Deployment: &TestDeployment{
				ID:                 1,
				Status:             "running",
				URL:                "https://test-myapp.apps.kyper.shop",
				ExpiresAt:          "2026-03-06T17:00:00Z",
				ProvisionLog:       "ready\n",
				ProvisionLogCursor: 6,
			},
		})
	}))
	defer srv.Close()

	status, err := client.GetTestDeploy("my-app", 42)
	if err != nil {
		t.Fatalf("GetTestDeploy failed: %v", err)
	}
	if status.VersionID != 99 {
		t.Errorf("expected VersionID 99, got %d", status.VersionID)
	}
	if status.BuildStatus != "built" {
		t.Errorf("expected build_status 'built', got %q", status.BuildStatus)
	}
	if status.Deployment == nil {
		t.Fatal("expected non-nil deployment")
	}
	if status.Deployment.Status != "running" {
		t.Errorf("expected deployment status 'running', got %q", status.Deployment.Status)
	}
	if status.Deployment.URL != "https://test-myapp.apps.kyper.shop" {
		t.Errorf("unexpected URL: %q", status.Deployment.URL)
	}
}

func TestGetTestDeployNullDeployment(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version_id":   99,
			"build_status": "building",
			"deployment":   nil,
		})
	}))
	defer srv.Close()

	status, err := client.GetTestDeploy("my-app", 0)
	if err != nil {
		t.Fatalf("GetTestDeploy failed: %v", err)
	}
	if status.Deployment != nil {
		t.Errorf("expected nil deployment, got %+v", status.Deployment)
	}
}

func TestGetTestDeploy404(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer srv.Close()

	_, err := client.GetTestDeploy("my-app", 0)
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

func TestDeleteTestDeploy(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps/my-app/test_deploy" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Test deployment is being torn down"})
	}))
	defer srv.Close()

	resp, err := client.DeleteTestDeploy("my-app")
	if err != nil {
		t.Fatalf("DeleteTestDeploy failed: %v", err)
	}
	if resp.Message != "Test deployment is being torn down" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestDeleteTestDeploy404(t *testing.T) {
	client, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no active test deployment"})
	}))
	defer srv.Close()

	_, err := client.DeleteTestDeploy("my-app")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

