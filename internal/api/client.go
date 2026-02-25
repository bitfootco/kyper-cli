package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Response types

type DeviceGrant struct {
	Code            string `json:"code"`
	VerificationURI string `json:"verification_uri"`
}

type TokenResponse struct {
	APIToken string `json:"api_token"`
	Pending  bool   `json:"pending"`
}

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type App struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Tagline     string `json:"tagline"`
	Status      string `json:"status"`
	PricingType string `json:"pricing_type"`
	Description string `json:"description"`
	Category    string `json:"category"`
	TechStack   string `json:"tech_stack"`
}

type VersionInfo struct {
	ID          int    `json:"id"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	ReviewNotes string `json:"review_notes"`
}

type AppStatus struct {
	App           App          `json:"app"`
	Status        string       `json:"status"`
	LatestVersion *VersionInfo `json:"latest_version"`
}

type VersionResponse struct {
	ID      int    `json:"id"`
	App     string `json:"app"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type BuildLog struct {
	Status   string `json:"status"`
	Log      string `json:"log"`
	Cursor   int    `json:"cursor"`
	Complete bool   `json:"complete"`
}

type MessageResponse struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}

// Client is the Kyper API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client. Use token="" for unauthenticated calls.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Transport: &Transport{Token: token},
			Timeout:   30 * time.Second,
		},
	}
}

// NewClientWithHTTP creates a client with a custom http.Client (for testing).
func NewClientWithHTTP(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
}

// doJSON makes a request and decodes the JSON response.
func (c *Client) doJSON(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}

	return nil
}

func parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode}

	// Try {"errors": [...]}
	var multiErr struct {
		Errors []string `json:"errors"`
	}
	if json.Unmarshal(body, &multiErr) == nil && len(multiErr.Errors) > 0 {
		apiErr.Messages = multiErr.Errors
		return apiErr
	}

	// Try {"error": "..."}
	var singleErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &singleErr) == nil && singleErr.Error != "" {
		apiErr.Message = singleErr.Error
		return apiErr
	}

	apiErr.Message = string(body)
	return apiErr
}

// Device Auth

func (c *Client) DeviceAuthorize() (*DeviceGrant, error) {
	var grant DeviceGrant
	err := c.doJSON("POST", "/api/v1/device/authorize", nil, &grant)
	return &grant, err
}

func (c *Client) DeviceToken(code string) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doJSON("GET", "/api/v1/device/token?code="+code, nil, &resp)
	return &resp, err
}

// User

func (c *Client) GetMe() (*User, error) {
	var user User
	err := c.doJSON("GET", "/api/v1/me", nil, &user)
	return &user, err
}

// Apps

func (c *Client) GetApp(slug string) (*App, error) {
	var app App
	err := c.doJSON("GET", "/api/v1/apps/"+slug, nil, &app)
	return &app, err
}

func (c *Client) CreateApp(params map[string]interface{}) (*App, error) {
	var app App
	err := c.doJSON("POST", "/api/v1/apps", map[string]interface{}{"app": params}, &app)
	return &app, err
}

func (c *Client) UpdateApp(slug string, params map[string]interface{}) (*App, error) {
	var app App
	err := c.doJSON("PATCH", "/api/v1/apps/"+slug, map[string]interface{}{"app": params}, &app)
	return &app, err
}

func (c *Client) GetAppStatus(slug string) (*AppStatus, error) {
	var status AppStatus
	err := c.doJSON("GET", "/api/v1/apps/"+slug+"/status", nil, &status)
	return &status, err
}

// Versions

func (c *Client) CreateVersion(slug, kyperYml, zipPath string) (*VersionResponse, error) {
	// Build multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add kyper_yml field
	if err := writer.WriteField("kyper_yml", kyperYml); err != nil {
		return nil, fmt.Errorf("writing kyper_yml field: %w", err)
	}

	// Add source_zip file
	file, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("opening zip file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("source_zip", filepath.Base(zipPath))
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copying zip to form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalizing multipart form: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/apps/"+slug+"/versions", body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uploading version: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var vr VersionResponse
	if err := json.Unmarshal(respBody, &vr); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &vr, nil
}

func (c *Client) GetBuildLog(versionID, cursor int) (*BuildLog, error) {
	var log BuildLog
	path := fmt.Sprintf("/api/v1/versions/%d/build_log?cursor=%d", versionID, cursor)
	err := c.doJSON("GET", path, nil, &log)
	return &log, err
}

func (c *Client) RetryVersion(versionID int) (*MessageResponse, error) {
	var resp MessageResponse
	path := fmt.Sprintf("/api/v1/versions/%d/retry", versionID)
	err := c.doJSON("POST", path, nil, &resp)
	return &resp, err
}

func (c *Client) CancelVersion(versionID int) (*MessageResponse, error) {
	var resp MessageResponse
	path := fmt.Sprintf("/api/v1/versions/%d/cancel", versionID)
	err := c.doJSON("POST", path, nil, &resp)
	return &resp, err
}

func (c *Client) DeleteVersion(versionID int) (*MessageResponse, error) {
	var resp MessageResponse
	path := fmt.Sprintf("/api/v1/versions/%d", versionID)
	err := c.doJSON("DELETE", path, nil, &resp)
	return &resp, err
}
