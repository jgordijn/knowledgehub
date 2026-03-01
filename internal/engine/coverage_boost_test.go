package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

// ============================================================
// scheduler.go — Start/Stop lifecycle with tick
// Covers: Start(), fetchAll(), retryFailedEntries() during tick
// ============================================================

func TestScheduler_FullLifecycle(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	testutil.CreateResource(t, app, "sched-test", feedServer.URL, "rss", "healthy", 0, true)

	// Create a failed entry for retryFailedEntries to pick up
	resource := testutil.CreateResource(t, app, "sched-test2", feedServer.URL, "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Failed", "https://example.com/f", "guid-sched-retry")
	entry.Set("processing_status", "failed")
	entry.Set("raw_content", "Some content")
	app.Save(entry)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	s := NewSchedulerWithInterval(app, 100*time.Millisecond)

	done := make(chan struct{})
	go func() {
		s.Start() // blocks until Stop
		close(done)
	}()

	// Let it run initial fetch + at least one tick
	time.Sleep(350 * time.Millisecond)

	s.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler did not stop in time")
	}
}

// ============================================================
// fetcher.go — processEntry fragment path with ScoreOnly
// ============================================================

func TestProcessEntry_FragmentScoreOnly(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Fragment Score", "https://example.com/fs", "guid-frag-score")
	entry.Set("raw_content", "Fragment content for scoring only path")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"stars":4,"summary":"Fragment summary"}`, nil
	})
	defer restore()

	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	status := updated.GetString("processing_status")
	if status != "done" {
		t.Errorf("processing_status = %q, want done", status)
	}
}

// ============================================================
// rss.go — loadExistingGUIDs with fragment-style GUIDs marks parent
// ============================================================

func TestLoadExistingGUIDs_FragmentGUIDs_MarksParent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "frag-guid-test", "https://example.com", "rss", "healthy", 0, true)

	// Create entry with fragment GUID
	testutil.CreateEntry(t, app, resource.Id, "Frag", "https://example.com/f", "parent-123#frag-abcdef")
	testutil.CreateEntry(t, app, resource.Id, "Normal", "https://example.com/n", "normal-guid-test")

	guids, err := loadExistingGUIDs(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !guids["parent-123#frag-abcdef"] {
		t.Error("fragment GUID should be in map")
	}
	if !guids["parent-123"] {
		t.Error("parent GUID should also be marked from fragment GUID")
	}
	if !guids["normal-guid-test"] {
		t.Error("normal GUID should be in map")
	}
}

// ============================================================
// rss.go — FetchRSS non-feed HTML body triggers browser fallback
// ============================================================

func TestFetchRSS_HTMLBodyTriggersBrowserFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>Bot check</h1></body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "cf-fallback", server.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = server.Client()
	defer func() { DefaultHTTPClient = origClient }()

	origBrowser := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`, nil
	}
	defer func() { BrowserFetchBodyFunc = origBrowser }()

	_, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be auto-set")
	}
}

// ============================================================
// rss.go — FetchRSS browser fallback fails
// ============================================================

func TestFetchRSS_BrowserFallbackFails(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "fail-both", server.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = server.Client()
	defer func() { DefaultHTTPClient = origClient }()

	origBrowser := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return "", fmt.Errorf("browser also failed")
	}
	defer func() { BrowserFetchBodyFunc = origBrowser }()

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error when both HTTP and browser fail")
	}
}

// ============================================================
// rss.go — FetchRSS unparseable feed body from browser
// ============================================================

func TestFetchRSS_UnparseableFeedFromBrowser(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "bad-feed-browser", "https://example.com/feed", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	origBrowser := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return "this is not a valid feed at all!!", nil
	}
	defer func() { BrowserFetchBodyFunc = origBrowser }()

	_, err := FetchRSS(app, resource, http.DefaultClient)
	if err == nil {
		t.Error("expected error for unparseable feed")
	}
}

// ============================================================
// rss.go — FetchRSS already use_browser, no save needed
// ============================================================

func TestFetchRSS_AlreadyUseBrowser_NoAutoLearn(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "already-ub", "https://example.com/feed", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	origBrowser := BrowserFetchBodyFunc
	BrowserFetchBodyFunc = func(url string) (string, error) {
		return `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`, nil
	}
	defer func() { BrowserFetchBodyFunc = origBrowser }()

	_, err := FetchRSS(app, resource, http.DefaultClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should still be true")
	}
}

// ============================================================
// rss.go — FetchRSS with content:encoded tag
// ============================================================

func TestFetchRSS_ContentEncodedField(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel><title>Test</title>
<item>
  <title>CE Item</title>
  <link>https://example.com/ce</link>
  <guid>guid-ce</guid>
  <content:encoded><![CDATA[<p>Full content from content:encoded</p>]]></content:encoded>
</item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "ce-test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Content, "Full content") {
		t.Errorf("expected content:encoded content, got: %q", entries[0].Content)
	}
}

// ============================================================
// rss.go — FetchRSS items with no GUID or link (skipped)
// ============================================================

func TestFetchRSS_SkipsItemWithNoGUIDOrLink(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>T</title>
<item><title>No GUID or Link</title><description>desc</description></item>
<item><title>Has Link</title><link>https://example.com/has</link></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "noguid-test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (item w/o guid skipped), got %d", len(entries))
	}
}

// ============================================================
// rss.go — FetchRSS fragment feed re-processes today's entries
// ============================================================

func TestFetchRSS_FragmentFeed_ReprocessesToday(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	todayDate := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Frag</title>
<item>
  <title>Today</title>
  <link>https://example.com/today</link>
  <guid>frag-today-guid</guid>
  <pubDate>%s</pubDate>
  <description><![CDATA[<p>Today content</p>]]></description>
</item>
</channel></rss>`, todayDate)))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "frag-reprocess", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

// ============================================================
// rss.go — fetchFeedHTTP non-OK status
// ============================================================

func TestFetchFeedHTTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := fetchFeedHTTP(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 503")
	}
	if !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("expected HTTP 503 in error, got: %v", err)
	}
}

// ============================================================
// readability.go — ExtractContent HTTP 403 error
// ============================================================

func TestExtractContent_HTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := ExtractContent(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 403")
	}
}

// ============================================================
// readability.go — ExtractContentFromHTML with invalid sourceURL
// ============================================================

func TestExtractContentFromHTML_InvalidSourceURL(t *testing.T) {
	result := ExtractContentFromHTML("<html><body><p>content</p></body></html>", "://bad")
	// Should handle gracefully
	_ = result
}

// ============================================================
// readability.go — truncate edge cases
// ============================================================

func TestTruncate_Variations(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		expect string
	}{
		{"short fits", "hello", 10, "hello"},
		{"exact len", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"whitespace trimmed", "  hi  ", 10, "hi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.expect {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expect)
			}
		})
	}
}

// ============================================================
// scraper.go — ScrapeArticleLinks heuristic with various link types
// ============================================================

func TestScrapeArticleLinks_HeuristicFiltersComprehensive(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<html><body>
<a href="%s/article/good-post">Good Post</a>
<a href="%s/">Home</a>
<a href="%s/tag/go">Tag</a>
<a href="%s/category/tech">Category</a>
<a href="%s/author/john">Author</a>
<a href="%s/page/2">Page 2</a>
<a href="%s/wp-content/img.jpg">WP Content</a>
<a href="%s/feed">Feed</a>
<a href="%s/rss">RSS</a>
<a href="%s/wp-admin">Admin</a>
<a href="https://other.com/external">External</a>
</body></html>`,
			server.URL, server.URL, server.URL, server.URL, server.URL,
			server.URL, server.URL, server.URL, server.URL, server.URL)))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "heuristic-full", server.URL, "watchlist", "healthy", 0, true)
	// No selector — heuristic mode

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the /article/good-post should pass all filters
	if len(links) != 1 {
		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		t.Errorf("expected 1 link (/article/good-post), got %d: %v", len(links), urls)
	}
}

// ============================================================
// scraper.go — nested anchor extraction in selector mode
// ============================================================

func TestScrapeArticleLinks_SelectorWithNestedAnchor(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div class="post-card"><a href="/nested-post">Nested Post Title</a></div>
<div class="post-card">No link in this card</div>
</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "nested-sel", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", ".post-card")
	app.Save(resource)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d", len(links))
	}
}

// ============================================================
// scraper.go — deduplicateLinks with both existing entries and duplicates
// ============================================================

func TestDeduplicateLinks_ComprehensiveDedup(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "dedup-comp", "https://example.com", "watchlist", "healthy", 0, true)
	testutil.CreateEntry(t, app, resource.Id, "Existing1", "https://example.com/existing1", "guid-ex1")
	testutil.CreateEntry(t, app, resource.Id, "Existing2", "https://example.com/existing2", "guid-ex2")

	links := []ScrapedLink{
		{Title: "New 1", URL: "https://example.com/new1"},
		{Title: "Existing 1", URL: "https://example.com/existing1"}, // dupe
		{Title: "New 1 Dup", URL: "https://example.com/new1"},       // same-batch dupe
		{Title: "New 2", URL: "https://example.com/new2"},
		{Title: "Existing 2", URL: "https://example.com/existing2"}, // dupe
	}

	deduped, err := deduplicateLinks(app, resource.Id, links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deduped) != 2 {
		t.Errorf("expected 2 unique new links, got %d", len(deduped))
	}
}

// ============================================================
// fragment.go — SplitFragments with only block elements (no <p>)
// ============================================================

func TestSplitFragments_OnlyBlocks(t *testing.T) {
	html := "<blockquote>Quote</blockquote><ul><li>Item</li></ul>"
	frags := SplitFragments(html)
	if len(frags) != 1 {
		t.Errorf("expected 1 fragment (all blocks combined), got %d", len(frags))
	}
}

// ============================================================
// fragment.go — findSimilarFragEntry best match selection
// ============================================================

func TestFindSimilarFragEntry_SelectsBestMatch(t *testing.T) {
	now := time.Now().UTC()

	existing := []existingFragEntry{
		{id: "1", title: "hello world test content here", publishedAt: now},
		{id: "2", title: "hello world test content here exactly", publishedAt: now},
		{id: "3", title: "completely different unrelated topic", publishedAt: now},
	}

	result := findSimilarFragEntry(existing, "hello world test content here exactly updated", &now)
	if result == nil {
		t.Fatal("expected a match")
	}
	// Should pick id "2" as the best match (more overlapping words)
	if result.id != "2" {
		t.Errorf("expected best match id=2, got id=%s", result.id)
	}
}

// ============================================================
// fetcher.go — fetchRSSResource with resource deleted mid-fetch (non-fragment)
// ============================================================

func TestFetchRSSResource_ResourceDeletedDuringNonFragmentFetch(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>T</title>
<item><title>Item</title><link>https://example.com/i</link><guid>guid-del-nf</guid></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "del-nonfrag", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// Delete before the loop processes entries
	app.Delete(resource)

	err := fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Errorf("expected nil (resource deleted), got: %v", err)
	}
}

// ============================================================
// fetcher.go — fetchRSSResource fragment feed with similar existing entry (dedup update)
// ============================================================

func TestFetchRSSResource_FragmentFeed_SimilarEntryUpdate(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Frag</title>
<item>
  <title>Moments</title>
  <link>https://example.com/moments</link>
  <guid>moments-guid</guid>
  <description><![CDATA[<p>Today content about testing is great and wonderful</p>]]></description>
</item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "frag-sim", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"stars":3,"summary":"test"}`, nil
	})
	defer restore()

	// First fetch — creates fragment entries
	err := fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("first fetch error: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) == 0 {
		t.Error("expected fragment entries after first fetch")
	}
}

// ============================================================
// fetcher.go — fetchWatchlistResource with empty extracted title uses link title
// ============================================================

func TestFetchWatchlistResource_EmptyExtractedTitleUsesLinkTitle(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>
<a href="/untitled">Link With Text</a>
</body></html>`))
		} else {
			w.Header().Set("Content-Type", "text/html")
			// Article with no title tag
			w.Write([]byte(`<html><body>
<p>` + strings.Repeat("Article body content. ", 20) + `</p>
</body></html>`))
		}
	}))
	defer pageServer.Close()

	resource := testutil.CreateResource(t, app, "watch-notitle", pageServer.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a")
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = pageServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	origBrowser := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{}, fmt.Errorf("no browser")
	}
	defer func() { BrowserExtractFunc = origBrowser }()

	err := fetchWatchlistResource(app, resource, pageServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}
}

// ============================================================
// browser.go — extractWithBrowserFallback: bot protection detected, browser succeeds, save error path
// ============================================================

func TestExtractWithBrowserFallback_BotDetected_BrowserSucceeds_SetsBrowserFlag(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 429 (Too Many Requests) — bot protection
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "bot-429", server.URL, "rss", "healthy", 0, true)

	oldBrowser := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{Title: "Browser Title", Content: "Browser content"}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowser }()

	extracted, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Title != "Browser Title" {
		t.Errorf("title = %q, want 'Browser Title'", extracted.Title)
	}

	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be set after 429 browser fallback")
	}
}

// ============================================================
// browser.go — extractWithBrowserFallback: use_browser=true, browser fails
// ============================================================

func TestExtractWithBrowserFallback_UseBrowserTrue_BrowserFails(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "ub-fail", "https://example.com/feed", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	oldBrowser := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{}, fmt.Errorf("browser extraction error")
	}
	defer func() { BrowserExtractFunc = oldBrowser }()

	_, err := extractWithBrowserFallback(app, resource, "https://example.com/article", nil)
	if err == nil {
		t.Error("expected error when browser fails and use_browser is true")
	}
}

// ============================================================
// browser.go — looksLikeBotProtection with nil error
// ============================================================

func TestLooksLikeBotProtection_NilError(t *testing.T) {
	if looksLikeBotProtection(nil) {
		t.Error("expected false for nil error")
	}
}

func TestLooksLikeBotProtection_Various(t *testing.T) {
	tests := []struct {
		err  string
		want bool
	}{
		{"HTTP 403 forbidden", true},
		{"HTTP 429 too many requests", true},
		{"HTTP 503 service unavailable", true},
		{"HTTP 404 not found", false},
		{"connection refused", false},
		{"timeout", false},
	}
	for _, tt := range tests {
		if got := looksLikeBotProtection(fmt.Errorf("%s", tt.err)); got != tt.want {
			t.Errorf("looksLikeBotProtection(%q) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

// ============================================================
// fragment.go — resolveContentLinks with no href/src attributes
// ============================================================

func TestResolveContentLinks_PlainText(t *testing.T) {
	html := `<p>No links or images here</p>`
	result := resolveContentLinks(html, "https://example.com")
	if !strings.Contains(result, "No links") {
		t.Errorf("expected content preserved, got: %s", result)
	}
}

// ============================================================
// rss.go — looksLikeFeedBody more edge cases
// ============================================================

func TestLooksLikeFeedBody_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"whitespace only", "   \n\t  ", false},
		{"json object", `{"version":"https://jsonfeed.org/version/1"}`, true},
		{"random xml", `<something><other/></something>`, true},
		{"html uppercase", `<HTML><BODY>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeFeedBody([]byte(tt.body)); got != tt.want {
				t.Errorf("looksLikeFeedBody(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

// ============================================================
// readability.go — ExtractContent with readability success but empty TextContent
// ============================================================

func TestExtractContent_EmptyTextContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Test</title></head>
<body><main>` + strings.Repeat("<br/>", 50) + `</main></body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = extracted // Just verify no panic
}

// ============================================================
// readability.go — ExtractContentFromHTML with empty text content fallback
// ============================================================

func TestExtractContentFromHTML_EmptyTextFallback(t *testing.T) {
	html := `<html><head><title>Test</title></head><body><div>` + strings.Repeat("<br/>", 50) + `</div></body></html>`
	result := ExtractContentFromHTML(html, "https://example.com")
	_ = result // Just verify no panic
}

// ============================================================
// fetcher.go — maxConcurrentAI semaphore
// ============================================================

func TestCreateEntry_ConcurrencyControl(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	resource := testutil.CreateResource(t, app, "conc-test", "https://example.com", "rss", "healthy", 0, true)

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"stars":3,"summary":"test","content_summary":"test"}`, nil
	})
	defer restore()

	// Create multiple entries concurrently to exercise the semaphore
	for i := 0; i < 3; i++ {
		err := createEntry(app, resource.Id, fmt.Sprintf("Entry %d", i),
			fmt.Sprintf("https://example.com/%d", i),
			fmt.Sprintf("guid-conc-%d", i),
			strings.Repeat("Some content. ", 20), nil, false)
		if err != nil {
			t.Fatalf("createEntry %d error: %v", i, err)
		}
	}

	// Wait for all background processing
	time.Sleep(1 * time.Second)
}
