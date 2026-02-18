package engine

import (
	"net/http"
	"time"
)

// browserTransport wraps an http.RoundTripper to inject browser-like headers.
// Many servers reject or return empty responses to non-browser clients.
type browserTransport struct {
	base http.RoundTripper
}

func (t *browserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; KnowledgeHub/1.0; +https://github.com/jgordijn/knowledgehub)")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}
	return t.base.RoundTrip(req)
}

// DefaultHTTPClient is the shared HTTP client with a 30-second timeout and browser-like headers.
var DefaultHTTPClient = &http.Client{
	Timeout:   30 * time.Second,
	Transport: &browserTransport{base: http.DefaultTransport},
}
