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

	// Retry logic for 5xx errors
	var resp *http.Response
	var err error
	backoffs := []time.Duration{0, 1 * time.Second, 2 * time.Second}

	for i, backoff := range backoffs {
		if i > 0 {
			time.Sleep(backoff)
			// Re-clone for retry (body may have been consumed)
			r = req.Clone(req.Context())
			if t.Token != "" {
				r.Header.Set("Authorization", "Bearer "+t.Token)
			}
			r.Header.Set("User-Agent", "kyper-cli/"+version.Version)
		}

		resp, err = t.base().RoundTrip(r)
		if err != nil {
			return nil, err
		}

		// Only retry on 5xx
		if resp.StatusCode < 500 || i == len(backoffs)-1 {
			return resp, nil
		}
		resp.Body.Close()
	}

	return resp, err
}
