package api

import (
	"net/http"
	"time"

	"github.com/bitfootco/kyper-cli/internal/version"
)

// Transport is a custom RoundTripper that injects auth and user-agent headers,
// and retries on 5xx errors.
type Transport struct {
	Token string
	Base  http.RoundTripper
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	r := req.Clone(req.Context())

	// Set headers
	if t.Token != "" {
		r.Header.Set("Authorization", "Bearer "+t.Token)
	}
	r.Header.Set("User-Agent", "kyper-cli/"+version.Version)

	// Only retry idempotent methods (GET, HEAD) on 5xx errors
	if r.Method != "GET" && r.Method != "HEAD" {
		return t.base().RoundTrip(r)
	}

	retryDelays := []time.Duration{1 * time.Second, 2 * time.Second}

	for attempt := 0; ; attempt++ {
		resp, err := t.base().RoundTrip(r)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 500 || attempt >= len(retryDelays) {
			return resp, nil
		}
		_ = resp.Body.Close()
		time.Sleep(retryDelays[attempt])
		// Re-clone for retry
		r = req.Clone(req.Context())
		if t.Token != "" {
			r.Header.Set("Authorization", "Bearer "+t.Token)
		}
		r.Header.Set("User-Agent", "kyper-cli/"+version.Version)
	}
}
