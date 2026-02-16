package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestFetchResource_UnknownType(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "unknown", "https://example.com", "rss", "healthy", 0, true)
	resource.Set("type", "unknown_type")
	app.Save(resource)

	err := FetchResource(app, resource, http.DefaultClient)
	if err != nil {
		t.Errorf("expected no error for unknown type, got %v", err)
	}
}

func TestFetchWatchlistResource_ScrapeError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "broken", server.URL, "watchlist", "healthy", 0, true)

	err := FetchResource(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for broken watchlist, got nil")
	}
}

func TestFetchWatchlistResource_WithContent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/posts/new-article">New Article</a></body></html>`))
	})
	mux.HandleFunc("/posts/new-article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>New Article</title></head><body><article><p>Great content here</p></article></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	err := fetchWatchlistResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("fetchWatchlistResource returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestFetchWatchlistResource_ExtractFailsFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/posts/broken">Broken Article</a></body></html>`))
	})
	mux.HandleFunc("/posts/broken", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	err := fetchWatchlistResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("fetchWatchlistResource returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestCreateEntry_WithNilPublishedAt(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	err := createEntry(app, resource.Id, "No Date", "https://example.com/no-date", "guid-no-date", "content", nil)
	if err != nil {
		t.Fatalf("createEntry returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "guid = 'guid-no-date'", "", 1, 0, nil)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].GetString("published_at") != "" {
		t.Errorf("expected empty published_at, got %s", entries[0].GetString("published_at"))
	}
}

func TestProcessEntry_WithNoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/test", "guid-1")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	// processEntry should not panic even without API key
	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	status := updated.GetString("processing_status")
	if status != "failed" {
		t.Errorf("processing_status = %q, want 'failed'", status)
	}
}
