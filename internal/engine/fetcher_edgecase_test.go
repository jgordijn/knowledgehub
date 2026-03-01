package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestFetchWatchlistResource_ExtractionFails_TitleFallback(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a class="post" href="/posts/failing">Fallback Title</a></body></html>`))
	})
	mux.HandleFunc("/posts/failing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog-fallback", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.post")
	app.Save(resource)

	err := fetchWatchlistResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Should have used the link title as fallback
	title := entries[0].GetString("title")
	if title != "Fallback Title" {
		t.Errorf("title = %q, want 'Fallback Title'", title)
	}
}

func TestFetchWatchlistResource_EmptyExtractedTitle(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a class="post" href="/posts/untitled">Link Text</a></body></html>`))
	})
	mux.HandleFunc("/posts/untitled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Minimal HTML where readability might not extract a title
		w.Write([]byte(`<html><body><p>Just some content without a title tag.</p></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog-untitled", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.post")
	app.Save(resource)

	err := fetchWatchlistResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestProcessEntry_CheckPreferences(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Pref Test", "https://example.com/pref", "guid-pref-test")
	entry.Set("processing_status", "pending")
	entry.Set("raw_content", "Article about preference testing")
	app.Save(entry)

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"summary":"Test summary.","stars":4}`, nil
	})
	defer restore()

	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("processing_status"); got != "done" {
		t.Errorf("processing_status = %q, want done", got)
	}
	// CheckAndRegeneratePreferences should have been called (but won't regenerate since no corrections)
}

func TestCreateEntry_SetsIsFragment(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	now := time.Now()
	err := createEntry(app, resource.Id, "Fragment Title", "https://example.com/frag-test", "guid-frag-test", "<p>Fragment</p>", &now, true)
	if err != nil {
		t.Fatalf("createEntry error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "guid = 'guid-frag-test'", "", 1, 0, nil)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].GetBool("is_fragment") {
		t.Error("expected is_fragment = true")
	}
}
