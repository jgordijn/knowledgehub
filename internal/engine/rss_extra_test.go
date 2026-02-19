package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/mmcdole/gofeed"
)

func TestItemLink_Empty(t *testing.T) {
	item := &gofeed.Item{Link: ""}
	if got := itemLink(item); got != "" {
		t.Errorf("itemLink = %q, want empty", got)
	}
}

func TestItemLink_WithLink(t *testing.T) {
	item := &gofeed.Item{Link: "https://example.com/article"}
	if got := itemLink(item); got != "https://example.com/article" {
		t.Errorf("itemLink = %q", got)
	}
}

func TestItemGUID_Empty(t *testing.T) {
	item := &gofeed.Item{}
	if got := itemGUID(item); got != "" {
		t.Errorf("itemGUID = %q, want empty", got)
	}
}

func TestItemContent_DescriptionFallback(t *testing.T) {
	item := &gofeed.Item{Content: "", Description: "fallback desc"}
	if got := itemContent(item); got != "fallback desc" {
		t.Errorf("itemContent = %q, want 'fallback desc'", got)
	}
}

func TestItemContent_WithContent(t *testing.T) {
	item := &gofeed.Item{Content: "rich content", Description: "desc"}
	if got := itemContent(item); got != "rich content" {
		t.Errorf("itemContent = %q, want 'rich content'", got)
	}
}

func TestLoadExistingGUIDs(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	testutil.CreateEntry(t, app, resource.Id, "A", "https://example.com/a", "guid-a")
	testutil.CreateEntry(t, app, resource.Id, "B", "https://example.com/b", "guid-b")

	guids, err := loadExistingGUIDs(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(guids) != 2 {
		t.Errorf("expected 2 GUIDs, got %d", len(guids))
	}
	if !guids["guid-a"] {
		t.Error("missing guid-a")
	}
	if !guids["guid-b"] {
		t.Error("missing guid-b")
	}
}

func TestLoadExistingGUIDs_EmptyGUID(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "A", "https://example.com/a", "")
	entry.Set("guid", "")
	app.Save(entry)

	guids, err := loadExistingGUIDs(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty GUIDs should not be included
	if len(guids) != 0 {
		t.Errorf("expected 0 GUIDs for empty guid entries, got %d", len(guids))
	}
}

func TestLoadExistingGUIDs_FragmentGUIDsIncludeParent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	// Simulate fragment entries whose GUIDs include the parent GUID prefix
	testutil.CreateEntry(t, app, resource.Id, "Frag 1", "https://example.com/a", "parent-guid-1#frag-abc123")
	testutil.CreateEntry(t, app, resource.Id, "Frag 2", "https://example.com/a", "parent-guid-1#frag-def456")

	guids, err := loadExistingGUIDs(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain both fragment GUIDs and the extracted parent GUID
	if !guids["parent-guid-1#frag-abc123"] {
		t.Error("missing fragment GUID parent-guid-1#frag-abc123")
	}
	if !guids["parent-guid-1#frag-def456"] {
		t.Error("missing fragment GUID parent-guid-1#frag-def456")
	}
	if !guids["parent-guid-1"] {
		t.Error("missing parent GUID parent-guid-1 (should be extracted from fragment GUIDs)")
	}
}


func TestFetchRSS_SkipsItemsWithNoGUID(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Feed where items have no GUID and no link
	feed := `<?xml version="1.0"?>
<rss version="2.0">
  <channel><title>Test</title>
    <item>
      <title>No ID</title>
      <description>Item with no GUID and no link</description>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(feed))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Items without GUID or link should be skipped
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for items without GUID, got %d", len(entries))
	}
}

func TestFetchRSS_EmptyResponse_NoFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns empty body (hard block — browser can't help either)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "nitter", server.URL, "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		t.Error("browser should not be called for empty responses")
		return "", nil
	}
	defer func() { BrowserFetchBodyFunc = oldBrowserFunc }()

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestFetchRSS_UseBrowser_SkipsHTTP(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "nitter", "https://nitter.example/rss", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	oldBrowserFunc := BrowserFetchBodyFunc
	called := false
	BrowserFetchBodyFunc = func(url string) (string, error) {
		called = true
		return testRSSFeed, nil
	}
	defer func() { BrowserFetchBodyFunc = oldBrowserFunc }()

	// Pass nil client — if plain HTTP were attempted, this would panic
	entries, err := FetchRSS(app, resource, nil)
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}
	if !called {
		t.Error("browser should be called directly when use_browser is true")
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestFetchRSS_BotProtection403_BrowserFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "protected", server.URL, "rss", "healthy", 0, true)

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
}

func TestLooksLikeFeedProtection(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"HTTP 403", fmt.Errorf("HTTP 403 for feed"), true},
		{"HTTP 429", fmt.Errorf("HTTP 429 for feed"), true},
		{"HTTP 503", fmt.Errorf("HTTP 503 for feed"), true},
		{"empty response", fmt.Errorf("feed https://nitter.net/x/rss returned an empty response"), false},
		{"HTTP 404", fmt.Errorf("HTTP 404 for feed"), false},
		{"network error", fmt.Errorf("dial tcp: connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeFeedProtection(tt.err)
			if got != tt.want {
				t.Errorf("looksLikeFeedProtection(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}


func TestLooksLikeFeedBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"empty", "", false},
		{"rss xml", `<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`, true},
		{"atom feed", `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`, true},
		{"rss no prolog", `<rss version="2.0"><channel></channel></rss>`, true},
		{"json feed", `{"version":"https://jsonfeed.org/version/1","title":"Test"}`, true},
		{"html doctype", `<!DOCTYPE html><html><body>Challenge</body></html>`, false},
		{"html tag", `<html><head></head><body>Cloudflare</body></html>`, false},
		{"cloudflare challenge", `<!DOCTYPE html><html><head><title>Just a moment...</title></head><body>Checking your browser</body></html>`, false},
		{"whitespace then xml", `  <?xml version="1.0"?><rss></rss>`, true},
		{"random text", `Hello world`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeFeedBody([]byte(tt.body))
			if got != tt.want {
				t.Errorf("looksLikeFeedBody(%q) = %v, want %v", tt.body[:min(50, len(tt.body))], got, tt.want)
			}
		})
	}
}
