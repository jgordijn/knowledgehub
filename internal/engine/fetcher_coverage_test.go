package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestProcessEntry_PanicRecovery(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Panic Test", "https://example.com/panic", "guid-panic")
	entry.Set("processing_status", "pending")
	entry.Set("raw_content", "Test content")
	app.Save(entry)

	// Make AI call panic
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		panic("intentional test panic")
	})
	defer restore()

	before := PanicCount.Load()

	// processEntry should recover from panic without propagating it
	processEntry(app, entry)

	after := PanicCount.Load()
	if after != before+1 {
		t.Errorf("PanicCount = %d, want %d", after, before+1)
	}
}

func TestProcessEntry_Fragment(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Fragment Test", "https://example.com/frag", "guid-frag-process")
	entry.Set("processing_status", "pending")
	entry.Set("raw_content", "<p>Short fragment</p>")
	entry.Set("is_fragment", true)
	app.Save(entry)

	// Mock ScoreOnly via SetCompleteFunc
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"summary":"","stars":3}`, nil
	})
	defer restore()

	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("processing_status"); got != "done" {
		t.Errorf("processing_status = %q, want done", got)
	}
	if got := updated.GetInt("ai_stars"); got != 3 {
		t.Errorf("ai_stars = %d, want 3", got)
	}
}

func TestProcessEntry_NonFragment(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Article Test", "https://example.com/article", "guid-article-process")
	entry.Set("processing_status", "pending")
	entry.Set("raw_content", "A full article about Go programming.")
	app.Save(entry)

	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"summary":"Go programming article.","stars":4}`, nil
	})
	defer restore()

	processEntry(app, entry)

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("processing_status"); got != "done" {
		t.Errorf("processing_status = %q, want done", got)
	}
	if got := updated.GetString("summary"); got != "Go programming article." {
		t.Errorf("summary = %q", got)
	}
}

func TestFetchRSSResource_ThinContent_ExtractsFromURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// Feed with thin content (less than 200 chars) and absolute article URL
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Thin Item</title><link>` + server.URL + `/article</link><guid>thin-1</guid><description>Short.</description></item>
</channel></rss>`))
	})
	mux.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testArticleHTML))
	})

	defer server.Close()

	// Override BrowserExtractFunc to avoid browser
	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{}, fmt.Errorf("should not be called")
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	resource := testutil.CreateResource(t, app, "test", server.URL+"/feed", "rss", "healthy", 0, true)

	err := FetchResource(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchResource returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestFetchRSSResource_ResourceDeletedMidFetch(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Item 1</title><link>https://example.com/1</link><guid>g1</guid><description>Content 1</description></item>
<item><title>Item 2</title><link>https://example.com/2</link><guid>g2</guid><description>Content 2</description></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "deleteme", feedServer.URL, "rss", "healthy", 0, true)
	resourceID := resource.Id

	// Delete the resource before fetching
	app.Delete(resource)

	entries, err := FetchRSS(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}

	// Now try to run fetchRSSResource with the deleted resource
	// This simulates the resource being deleted mid-fetch
	resource2 := testutil.CreateResource(t, app, "deleteme2", feedServer.URL, "rss", "healthy", 0, true)
	_ = entries
	_ = resourceID

	// Delete resource after RSS fetch but before entry creation
	go func() {
		time.Sleep(10 * time.Millisecond)
		app.Delete(resource2)
	}()

	err = fetchRSSResource(app, resource2, feedServer.Client())
	// Should handle gracefully (either succeed partially or return nil)
	_ = err
}

func TestIsThinContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"empty", "", true},
		{"short", "Hello world", true},
		{"just under threshold", string(make([]byte, 199)), true},
		{"at threshold", string(make([]byte, 200)), false},
		{"long", string(make([]byte, 500)), false},
		{"whitespace only", "     \n\n\t\t   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isThinContent(tt.content); got != tt.want {
				t.Errorf("isThinContent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadExistingFragEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	entry := testutil.CreateEntry(t, app, resource.Id, "Fragment 1", "https://example.com/frag", "frag-guid-1")
	entry.Set("is_fragment", true)
	entry.Set("published_at", "2026-02-18 10:00:00.000Z")
	app.Save(entry)

	entries, err := loadExistingFragEntries(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].title != "Fragment 1" {
		t.Errorf("title = %q, want 'Fragment 1'", entries[0].title)
	}
}

func TestLoadExistingFragEntries_NoFragments(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	// Regular entry, not a fragment
	testutil.CreateEntry(t, app, resource.Id, "Regular", "https://example.com/reg", "reg-guid")

	entries, err := loadExistingFragEntries(app, resource.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 fragment entries, got %d", len(entries))
	}
}

func TestUpdateFragEntry_NotFound(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	err := updateFragEntry(app, "nonexistent", "Title", "guid", "<p>Content</p>")
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
}

func TestFetchWatchlistResource_ResourceDeletedDuringFetch(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/posts/one">Article One</a></body></html>`))
	})
	mux.HandleFunc("/posts/one", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testArticleHTML))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource := testutil.CreateResource(t, app, "deleteme", server.URL, "watchlist", "healthy", 0, true)

	// Delete after scrape but before entry creation
	go func() {
		time.Sleep(50 * time.Millisecond)
		app.Delete(resource)
	}()

	// Should handle gracefully
	_ = fetchWatchlistResource(app, resource, server.Client())
}
