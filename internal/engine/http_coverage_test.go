package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBrowserTransport_RoundTrip_SetsDefaultHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the transport set browser-like headers
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("expected User-Agent to be set")
		}
		if ua != "Mozilla/5.0 (compatible; KnowledgeHub/1.0; +https://github.com/jgordijn/knowledgehub)" {
			t.Errorf("unexpected User-Agent: %s", ua)
		}

		accept := r.Header.Get("Accept")
		if accept == "" {
			t.Error("expected Accept to be set")
		}
		if accept != "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" {
			t.Errorf("unexpected Accept: %s", accept)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &browserTransport{base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestBrowserTransport_RoundTrip_PreservesExistingHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "CustomBot/1.0" {
			t.Errorf("User-Agent should not be overwritten: got %q", ua)
		}
		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("Accept should not be overwritten: got %q", accept)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &browserTransport{base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "CustomBot/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
}
