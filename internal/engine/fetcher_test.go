package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestFetchResource_RSS(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Set up a fake API key so AI processing goroutines don't crash
	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "rss-test", feedServer.URL, "rss", "healthy", 0, true)

	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("FetchResource returned error: %v", err)
	}

	// Give AI goroutines a moment (they'll fail gracefully)
	time.Sleep(100 * time.Millisecond)

	entries, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if err != nil {
		t.Fatalf("failed to find entries: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestFetchResource_Watchlist(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html><body>
  <a class="post" href="/articles/one">Article One</a>
  <a class="post" href="/articles/two">Article Two</a>
</body></html>`))
	})
	mux.HandleFunc("/articles/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testArticleHTML))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog-test", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.post")
	app.Save(resource)

	err := FetchResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchResource returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if err != nil {
		t.Fatalf("failed to find entries: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestCreateEntry(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// No API key means AI will fail gracefully
	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	published := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	err := createEntry(app, resource.Id, "Test Title", "https://example.com/article", "guid-1", "Test content", &published, false)
	if err != nil {
		t.Fatalf("createEntry returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if err != nil {
		t.Fatalf("failed to find entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if got := entry.GetString("title"); got != "Test Title" {
		t.Errorf("title = %q, want %q", got, "Test Title")
	}
	// processing_status may be "pending" or "failed" depending on AI goroutine timing
	got := entry.GetString("processing_status")
	if got != "pending" && got != "failed" {
		t.Errorf("processing_status = %q, want pending or failed", got)
	}
	if got := entry.GetString("guid"); got != "guid-1" {
		t.Errorf("guid = %q, want %q", got, "guid-1")
	}
}
