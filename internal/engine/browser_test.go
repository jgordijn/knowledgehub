package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestLooksLikeBotProtection(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"HTTP 403", fmt.Errorf("HTTP 403 for https://example.com"), true},
		{"HTTP 429", fmt.Errorf("HTTP 429 for https://example.com"), true},
		{"HTTP 503", fmt.Errorf("HTTP 503 for https://example.com"), true},
		{"HTTP 404", fmt.Errorf("HTTP 404 for https://example.com"), false},
		{"network error", fmt.Errorf("dial tcp: connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeBotProtection(tt.err)
			if got != tt.want {
				t.Errorf("looksLikeBotProtection(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestExtractWithBrowserFallback_PlainHTTPSuccess(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testArticleHTML))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		t.Error("browser should not be called when plain HTTP succeeds")
		return ExtractedContent{}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	extracted, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Content == "" {
		t.Error("expected content from plain HTTP")
	}

	// use_browser should not be set
	updated, _ := app.FindRecordById("resources", resource.Id)
	if updated.GetBool("use_browser") {
		t.Error("use_browser should not be set when plain HTTP succeeds")
	}
}

func TestExtractWithBrowserFallback_FallbackToBrowser(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns 403 (Cloudflare-like)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{
			Title:   "Browser Title",
			Content: "Browser extracted content with enough text to not be thin",
		}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	extracted, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Content != "Browser extracted content with enough text to not be thin" {
		t.Errorf("expected browser content, got %q", extracted.Content)
	}

	// use_browser should be auto-learned
	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be set after browser fallback succeeds")
	}
}

func TestExtractWithBrowserFallback_UseBrowserSkipsHTTP(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	oldBrowserFunc := BrowserExtractFunc
	called := false
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		called = true
		return ExtractedContent{
			Title:   "Browser Title",
			Content: "Browser content",
		}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	// Pass nil client — if plain HTTP were attempted, this would panic
	extracted, err := extractWithBrowserFallback(app, resource, "https://example.com/article", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("browser should be called directly when use_browser is true")
	}
	if extracted.Content != "Browser content" {
		t.Errorf("expected browser content, got %q", extracted.Content)
	}
}

func TestExtractWithBrowserFallback_NonBotError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns 404 (not bot protection)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		t.Error("browser should not be called for non-bot errors")
		return ExtractedContent{}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	_, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestExtractWithBrowserFallback_BrowserFails(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns 403
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{}, fmt.Errorf("browser launch failed")
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	_, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err == nil {
		t.Error("expected error when browser also fails")
	}

	// use_browser should NOT be set when browser fails
	updated, _ := app.FindRecordById("resources", resource.Id)
	if updated.GetBool("use_browser") {
		t.Error("use_browser should not be set when browser fails")
	}
}
