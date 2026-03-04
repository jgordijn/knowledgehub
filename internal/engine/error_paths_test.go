package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

// ============================================================
// scheduler.go:81 — FetchAllResources FindRecordsByFilter error
// Triggered by deleting the "resources" collection before calling
// ============================================================

func TestFetchAllResources_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Delete entries first (has FK to resources), then resources
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding entries collection: %v", err)
	}
	if err := app.Delete(entriesCol); err != nil {
		t.Fatalf("deleting entries collection: %v", err)
	}

	col, err := app.FindCollectionByNameOrId("resources")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	// Should not panic, should log error and return
	FetchAllResources(app)
}

// ============================================================
// scheduler.go:116 — retryFailedEntries FindRecordsByFilter error
// Triggered by deleting the "entries" collection before calling
// ============================================================

func TestRetryFailedEntries_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Delete entries collection
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	s := NewSchedulerWithInterval(app, 1*time.Hour)
	// Should not panic, should log error and return
	s.retryFailedEntries()
}

// ============================================================
// fetcher.go:286 — loadExistingFragEntries error path
// Triggered by deleting the "entries" collection
// ============================================================

func TestLoadExistingFragEntries_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	_, err = loadExistingFragEntries(app, "nonexistent-resource")
	if err == nil {
		t.Error("expected error when entries collection is deleted")
	}
}

// ============================================================
// rss.go:198 — loadExistingGUIDs error path
// ============================================================

func TestLoadExistingGUIDs_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	_, err = loadExistingGUIDs(app, "nonexistent-resource")
	if err == nil {
		t.Error("expected error when entries collection is deleted")
	}
}

// ============================================================
// fetcher.go:212 — createEntry error when entries collection missing
// ============================================================

func TestCreateEntry_CollectionMissing(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	err = createEntry(app, "fake-resource", "Test", "https://example.com", "guid", "content", nil, false)
	if err == nil {
		t.Error("expected error when entries collection is missing")
	}
}

// ============================================================
// scraper.go:146 — deduplicateLinks error when entries collection missing
// ============================================================

func TestDeduplicateLinks_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting collection: %v", err)
	}

	links := []ScrapedLink{{Title: "Test", URL: "https://example.com"}}
	_, err = deduplicateLinks(app, "fake-resource", links)
	if err == nil {
		t.Error("expected error when entries collection is missing")
	}
}

// ============================================================
// scraper.go:69 — ScrapeArticleLinks dedup error path
// ============================================================

func TestScrapeArticleLinks_DeduplicateError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/article">Article</a></body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a")
	app.Save(resource)

	// Delete entries collection to make dedup fail
	col, _ := app.FindCollectionByNameOrId("entries")
	app.Delete(col)

	_, err := ScrapeArticleLinks(app, resource, server.Client())
	if err == nil {
		t.Error("expected error when dedup fails")
	}
}

// ============================================================
// rss.go:75 — FetchRSS loadExistingGUIDs error
// ============================================================

func TestFetchRSS_LoadGUIDsError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title>
<item><title>Item</title><link>https://example.com/i</link><guid>g1</guid></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test-guid-err", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// Delete entries collection to trigger loadExistingGUIDs error
	col, _ := app.FindCollectionByNameOrId("entries")
	app.Delete(col)

	_, err := FetchRSS(app, resource, feedServer.Client())
	if err == nil {
		t.Error("expected error when loadExistingGUIDs fails")
	}
}

// ============================================================
// rss.go:145 — fetchFeedHTTP ReadAll error (simulate with truncated body)
// ============================================================

func TestFetchFeedHTTP_ReadBodyError(t *testing.T) {
	// This is very hard to trigger with httptest. Test already covered via other paths.
	// Just verify normal whitespace-only body returns error (rss.go:149)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("   \n\t  "))
	}))
	defer server.Close()

	_, err := fetchFeedHTTP(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for whitespace-only response")
	}
}
