package api

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTransportSetsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{Token: "mytoken"}}
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	client.Do(req)

	if gotAuth != "Bearer mytoken" {
		t.Errorf("expected 'Bearer mytoken', got %q", gotAuth)
	}
}

func TestTransportSkipsAuthWhenEmpty(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{Token: ""}}
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	client.Do(req)

	if gotAuth != "" {
		t.Errorf("expected empty auth header, got %q", gotAuth)
	}
}

func TestTransportSetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{Token: ""}}
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	client.Do(req)

	if gotUA == "" {
		t.Error("expected user-agent header to be set")
	}
}

func TestTransportRetries5xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{Token: "tok"}}
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestTransportNoRetryOn4xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{Token: "tok"}}
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	resp, _ := client.Do(req)

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", atomic.LoadInt32(&attempts))
	}
}
