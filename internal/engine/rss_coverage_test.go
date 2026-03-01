package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestFetchFeedHTTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>Test</title></channel></rss>`))
	}))
	defer server.Close()

	body, err := fetchFeedHTTP(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "<rss") {
		t.Errorf("body should contain RSS content: %s", body)
	}
}

func TestFetchFeedHTTP_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := fetchFeedHTTP(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 403 status")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestFetchFeedHTTP_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body
	}))
	defer server.Close()

	_, err := fetchFeedHTTP(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error should mention empty response: %v", err)
	}
}

func TestFetchFeedHTTP_NetworkError(t *testing.T) {
	_, err := fetchFeedHTTP("http://invalid.test.localhost:99999/feed", http.DefaultClient)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestFetchRSS_HTMLResponse_BrowserFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns HTML (Cloudflare challenge) instead of XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>Checking your browser...</h1></body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "cf", server.URL, "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return testRSSFeed, nil
	}
	defer func() { BrowserFetchBodyFunc = oldBrowserFunc }()

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries from browser fallback, got %d", len(entries))
	}

	// Should have auto-learned use_browser
	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be set after browser fallback")
	}
}

func TestFetchRSS_BrowserFetchFails(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns HTML instead of XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>Not a feed</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "fail", server.URL, "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return "", fmt.Errorf("browser launch failed")
	}
	defer func() { BrowserFetchBodyFunc = oldBrowserFunc }()

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error when browser fallback also fails")
	}
}

func TestFetchRSS_ParseError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns invalid XML (not a feed)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><not-a-feed>invalid</not-a-feed>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "bad", server.URL, "rss", "healthy", 0, true)

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for unparseable feed")
	}
}

func TestLooksLikeFeedBody_OtherXML(t *testing.T) {
	// Other XML that's not HTML should be considered a feed
	body := []byte(`<root><item>data</item></root>`)
	if !looksLikeFeedBody(body) {
		t.Error("other XML should be considered feed-like")
	}
}

func TestFetchRSS_SaveBrowserFlagError(t *testing.T) {
	// This tests the path where use_browser auto-learn succeeds
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return HTML to trigger browser fallback
		w.Write([]byte(`<html><body>Challenge</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "auto-learn", server.URL, "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return testRSSFeed, nil
	}
	defer func() { BrowserFetchBodyFunc = oldBrowserFunc }()

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected entries from browser fallback")
	}

	// Verify use_browser was saved
	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be set")
	}
}
