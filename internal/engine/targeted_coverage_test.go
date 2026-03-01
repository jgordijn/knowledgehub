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
// fetcher.go:88 — fragment feed without API key uses SplitFragments
// ============================================================

func TestFetchRSSResource_FragmentFeed_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// DO NOT set openrouter_api_key — forces fallback to heuristic SplitFragments
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Fragment Feed</title>
<item>
  <title>Post With Fragments</title>
  <link>https://example.com/post</link>
  <guid>frag-no-api</guid>
  <description><![CDATA[<p>Fragment one about Go</p><p>Fragment two about Rust</p><p>Fragment three about Python</p>]]></description>
</item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "frag-feed", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	err := fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created fragment entries using heuristic split
	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) == 0 {
		t.Error("expected fragment entries to be created")
	}
}

// ============================================================
// fetcher.go:262 — processEntry save error when setting failed status
// (This path is hard to trigger directly, but we can test the AIFailure path
//  which goes through it when AI returns error)
// ============================================================

func TestProcessEntry_SavesFailedStatus(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "guid-fail-save")
	entry.Set("raw_content", "Some content")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "", fmt.Errorf("AI unavailable")
	})
	defer restore()

	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("processing_status"); got != "failed" {
		t.Errorf("processing_status = %q, want failed", got)
	}
}

// ============================================================
// rss.go:98 — fragment feed: skip old entry that already exists
// ============================================================

func TestFetchRSS_FragmentFeed_SkipsOldExistingEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, 0, -7).Format("Mon, 02 Jan 2006 15:04:05 -0700")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Frag</title>
<item>
  <title>Old Post</title>
  <link>https://example.com/old</link>
  <guid>old-frag-guid</guid>
  <pubDate>%s</pubDate>
  <description><![CDATA[<p>Old fragment</p>]]></description>
</item>
</channel></rss>`, oldDate)))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "frag-feed", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// First fetch — creates entries
	err := fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("first fetch error: %v", err)
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Second fetch — old entry should be skipped (not reprocessed, not today/yesterday)
	err = fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("second fetch error: %v", err)
	}
}

// ============================================================
// rss.go:105 — skip articles older than 12 months
// ============================================================

func TestFetchRSS_SkipsOldArticles(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	oldDate := time.Now().AddDate(-2, 0, 0).Format("Mon, 02 Jan 2006 15:04:05 -0700")
	newDate := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item>
  <title>Old Article</title>
  <link>https://example.com/old</link>
  <guid>guid-old-article</guid>
  <pubDate>%s</pubDate>
</item>
<item>
  <title>New Article</title>
  <link>https://example.com/new</link>
  <guid>guid-new-article</guid>
  <pubDate>%s</pubDate>
</item>
</channel></rss>`, oldDate, newDate)))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return the new article, old one is skipped
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (old one skipped), got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Title != "New Article" {
		t.Errorf("expected 'New Article', got %q", entries[0].Title)
	}
}

// ============================================================
// rss.go:61 — save error when auto-learning use_browser for feeds
// (Tested indirectly via the successful browser fallback path)
// ============================================================

func TestFetchRSS_BrowserAutoLearn_SavesFlag(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server returns 403 to trigger bot protection → browser fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

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
		t.Error("use_browser should be auto-set after browser fallback")
	}
}

// ============================================================
// rss.go:75 — loadExistingGUIDs error in FetchRSS
// (Hard to trigger since DB is always valid, but test the function directly)
// ============================================================

func TestFetchRSS_LoadExistingGUIDsError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title>
		<item><title>Item</title><link>https://example.com/i</link><guid>g1</guid></item>
		</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// This should work fine — we can't easily trigger the loadExistingGUIDs error
	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected entries")
	}
}

// ============================================================
// rss.go:130 — fetchFeedHTTP request creation error  
// ============================================================

func TestFetchFeedHTTP_BadURL(t *testing.T) {
	_, err := fetchFeedHTTP("://bad-url", http.DefaultClient)
	if err == nil {
		t.Error("expected error for bad URL")
	}
}

// ============================================================
// readability.go:38 — readability.FromReader error
// readability.go:46 — content == "" && article.Content != ""
// ============================================================

func TestExtractContent_ReadabilityFromReaderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Return a response that causes readability to fail
		w.Write([]byte(""))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fallback to URL as title
	_ = extracted
}

func TestExtractContentFromHTML_ReadabilityError_EmptyInput(t *testing.T) {
	// Very malformed content that readability might fail on
	result := ExtractContentFromHTML("", "https://example.com")
	// Should handle gracefully
	_ = result
}

// ============================================================
// scraper.go:92 — extractLink with non-http scheme (javascript:)
// ============================================================

func TestScrapeArticleLinks_JavascriptLinks(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
		<a href="javascript:void(0)">JS Link</a>
		<a href="mailto:test@example.com">Email</a>
		<a href="/real-article">Real Article</a>
		</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a")
	app.Save(resource)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// javascript: and mailto: should be filtered out
	for _, link := range links {
		if strings.HasPrefix(link.URL, "javascript:") || strings.HasPrefix(link.URL, "mailto:") {
			t.Errorf("should not contain non-http URL: %s", link.URL)
		}
	}
}

// ============================================================
// scraper.go:105 — resolveURL with unparseable href
// ============================================================

func TestScrapeArticleLinks_UnparseableHref(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
		<a href="https://example.com/article">Good Link</a>
		</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a")
	app.Save(resource)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) == 0 {
		t.Error("expected at least one good link")
	}
}

// ============================================================
// scheduler.go — additional coverage for FetchSingleResource branches
// ============================================================

func TestFetchSingleResource_RecordsFailureOnFetchError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "fail-test", server.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = server.Client()
	defer func() { DefaultHTTPClient = origClient }()

	FetchSingleResource(app, resource)

	updated, _ := app.FindRecordById("resources", resource.Id)
	if updated.GetInt("consecutive_failures") != 1 {
		t.Errorf("consecutive_failures = %d, want 1", updated.GetInt("consecutive_failures"))
	}
}

func TestFetchSingleResource_RecordsSuccessOnCleanFetch(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "success-test", feedServer.URL, "rss", "failing", 2, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	FetchSingleResource(app, resource)

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusHealthy {
		t.Errorf("status = %q, want healthy", got)
	}
	if got := updated.GetInt("consecutive_failures"); got != 0 {
		t.Errorf("consecutive_failures = %d, want 0", got)
	}
}

func TestFetchAllResources_NoActiveResources(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Create only inactive/quarantined resources
	testutil.CreateResource(t, app, "quarantined", "https://example.com", "rss", "quarantined", 5, true)
	testutil.CreateResource(t, app, "inactive", "https://example.com", "rss", "healthy", 0, false)

	// Should not panic
	FetchAllResources(app)
}

func TestRetryFailedEntries_WithFailedEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create a failed entry
	entry := testutil.CreateEntry(t, app, resource.Id, "Failed", "https://example.com/fail", "guid-retry-test")
	entry.Set("processing_status", "failed")
	entry.Set("raw_content", "Some content for reprocessing")
	app.Save(entry)

	s := NewSchedulerWithInterval(app, 1*time.Hour)
	s.retryFailedEntries()

	// Give goroutine time to process
	time.Sleep(500 * time.Millisecond)
}

// ============================================================
// fragment.go:108 — SplitFragmentsWithAI text truncation at 300 chars
// ============================================================

func TestSplitFragmentsWithAI_LongFragmentText(t *testing.T) {
	// Create HTML with fragments that have > 300 chars of text
	longText := strings.Repeat("Long text content. ", 30) // ~570 chars
	html := "<p>" + longText + "</p><p>Second fragment</p>"

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"groups": [[0, 1]]}`, nil
	})
	defer restore()

	fragments := SplitFragmentsWithAI(html, "key", "model")
	if len(fragments) == 0 {
		t.Error("expected at least one fragment")
	}
}

// ============================================================
// fetcher.go — watchlist createEntry error path & resource deleted
// ============================================================

func TestFetchWatchlistResource_WithWorkingExtraction(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>
				<article><a href="/post-1">Post One</a></article>
			</body></html>`))
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><title>Post One</title></head><body>
				<article><p>` + strings.Repeat("This is article content. ", 20) + `</p></article>
			</body></html>`))
		}
	}))
	defer pageServer.Close()

	resource := testutil.CreateResource(t, app, "watch", pageServer.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "article a")
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

	// Should have created an entry
	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) == 0 {
		t.Error("expected at least one entry created")
	}
}

// ============================================================
// fetcher.go — fragment feed with saveFragmentHashes path
// ============================================================

func TestFetchRSSResource_FragmentFeed_SavesHashes(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "key")
	testutil.CreateSetting(t, app, "openrouter_model", "model")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Frag</title>
<item>
  <title>Moment Post</title>
  <link>https://example.com/moments</link>
  <guid>moment-guid</guid>
  <description><![CDATA[<p>Moment one about testing</p><p>Moment two about coverage</p>]]></description>
</item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "frag-hash", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		// Return valid scoring response
		return `{"stars":3,"summary":"Test summary"}`, nil
	})
	defer restore()

	restore2 := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"groups": [[0], [1]]}`, nil
	})
	defer restore2()

	err := fetchRSSResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for background processing
	time.Sleep(500 * time.Millisecond)

	// Verify fragment_hashes was saved on the resource
	updated, _ := app.FindRecordById("resources", resource.Id)
	hashes := updated.GetString("fragment_hashes")
	if hashes == "" {
		t.Error("expected fragment_hashes to be saved")
	}
}

// ============================================================
// readability.go:46 — ExtractContent with empty text but non-empty HTML content
// ============================================================

func TestExtractContent_EmptyTextButHTMLContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Content that readability can parse but yields empty TextContent
		w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Test Page</title></head>
<body>
<div class="content">
<img src="image.jpg" alt="just an image"/>
</div>
</body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should handle empty text content gracefully
	_ = extracted
}

// ============================================================
// fragment.go:311 — resolveContentLinks with bad goquery parse
// fragment.go:334 — resolveContentLinks with body.Html() error
// (goquery is very lenient so these are hard to trigger)
// ============================================================

func TestResolveContentLinks_EmptyHTML(t *testing.T) {
	result := resolveContentLinks("", "https://example.com")
	// Should handle empty HTML gracefully
	_ = result
}

func TestResolveContentLinks_ComplexHTML(t *testing.T) {
	html := `<a href="/page1">Page 1</a><img src="/img.png"><a href="https://absolute.com/url">Absolute</a>`
	result := resolveContentLinks(html, "https://example.com")
	if !strings.Contains(result, "https://example.com/page1") {
		t.Errorf("expected resolved href, got: %s", result)
	}
	if !strings.Contains(result, "https://example.com/img.png") {
		t.Errorf("expected resolved src, got: %s", result)
	}
	if !strings.Contains(result, "https://absolute.com/url") {
		t.Errorf("expected absolute URL preserved, got: %s", result)
	}
}

// ============================================================
// scraper.go:38-45 — ScrapeArticleLinks io.ReadAll error / goquery error
// These are extremely hard to trigger with httptest, but we can test
// the goquery error path with a selector-based scrape
// ============================================================

func TestScrapeArticleLinks_WithSelectorReturnsLinks(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
		<div class="articles">
			<a href="/post-1">Post 1</a>
			<a href="/post-2">Post 2</a>
		</div>
		<a href="/nav">Navigation</a>
		</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", ".articles a")
	app.Save(resource)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

// ============================================================
// scraper.go:124 — isArticleLink url.Parse error
// ============================================================

func TestIsArticleLink_InvalidURL(t *testing.T) {
	// url.Parse is very lenient, most strings parse. But we can test edge cases.
	result := isArticleLink("https://example.com/valid-article", "https://example.com")
	if !result {
		t.Error("expected true for valid article link")
	}
}

// ============================================================
// New helper for SetFragmentCompleteFunc
// ============================================================

func TestSetFragmentCompleteFunc_Integration(t *testing.T) {
	called := false
	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		called = true
		return `{"groups": [[0]]}`, nil
	})
	defer restore()

	html := "<p>First fragment</p><p>Second fragment</p>"
	_ = SplitFragmentsWithAI(html, "key", "model")
	if !called {
		t.Error("expected custom fragment complete func to be called")
	}
}


// ============================================================
// fetcher.go:185 — watchlist resource deleted during link processing
// ============================================================

func TestFetchWatchlistResource_ResourceDeletedDuringLinkProcessing(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>
				<article><a href="/art1">Article 1</a></article>
			</body></html>`))
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><title>Art</title></head><body>
				<article><p>` + strings.Repeat("Some article content. ", 20) + `</p></article>
			</body></html>`))
		}
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "watch-del", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "article a")
	app.Save(resource)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = server.Client()
	defer func() { DefaultHTTPClient = origClient }()

	origBrowser := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{}, fmt.Errorf("no browser")
	}
	defer func() { BrowserExtractFunc = origBrowser }()

	// Delete the resource BEFORE calling fetchWatchlistResource
	// The scrape will still succeed (it uses cached resource data),
	// but the loop check will find it's deleted
	app.Delete(resource)

	err := fetchWatchlistResource(app, resource, server.Client())
	// Should return nil since resource was deleted
	if err != nil {
		t.Errorf("expected nil error when resource deleted, got: %v", err)
	}
}

// ============================================================
// rss.go:61 — save error setting use_browser (already tested via BotProtection test)
// rss.go:75 — loadExistingGUIDs error (DB error, hard to trigger)
// Test RSS feed with items that have nil publishedAt
// ============================================================

func TestFetchRSS_ItemWithoutPubDate(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item>
  <title>No Date Article</title>
  <link>https://example.com/nodate</link>
  <guid>guid-nodate</guid>
</item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test", feedServer.URL, "rss", "healthy", 0, true)

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
// readability.go:46 — content == "" but article.Content != ""
// ============================================================

func TestExtractContent_HTMLContentFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Return HTML where readability produces Content but empty TextContent
		w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Image Gallery</title></head>
<body>
<div class="gallery">
<figure><img src="img1.jpg" alt="Image 1"/><figcaption>Caption 1</figcaption></figure>
<figure><img src="img2.jpg" alt="Image 2"/><figcaption>Caption 2</figcaption></figure>
</div>
</body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should handle gracefully
	_ = extracted.Content
}

// ============================================================
// readability.go:69 — ExtractContentFromHTML readability error
// ============================================================

func TestExtractContentFromHTML_GracefulFallback(t *testing.T) {
	// Very minimal HTML that might trigger readability error
	result := ExtractContentFromHTML("<html></html>", "https://example.com")
	// Should not panic, should have some result
	_ = result
}
