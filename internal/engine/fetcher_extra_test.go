package engine

import (
	"fmt"
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

	err := createEntry(app, resource.Id, "No Date", "https://example.com/no-date", "guid-no-date", "content", nil, false)
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


func TestCreateEntry_Fragment(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	err := createEntry(app, resource.Id, "Fragment Title", "https://example.com/fragment", "guid-frag", "Short fragment content", nil, true)
	if err != nil {
		t.Fatalf("createEntry returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, _ := app.FindRecordsByFilter("entries", "guid = 'guid-frag'", "", 1, 0, nil)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].GetBool("is_fragment") {
		t.Error("expected is_fragment to be true")
	}
}

func TestFetchRSSResource_FragmentFeed(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "fragment-rss", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("FetchResource returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	entries, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if err != nil {
		t.Fatalf("failed to find entries: %v", err)
	}

	for _, entry := range entries {
		if !entry.GetBool("is_fragment") {
			t.Errorf("entry %q should have is_fragment=true", entry.GetString("title"))
		}
	}
}

func TestFragmentPublishedAt(t *testing.T) {
	now := time.Date(2026, 2, 18, 0, 5, 0, 0, time.UTC)           // 0:05 today
	lastChecked := time.Date(2026, 2, 17, 23, 35, 0, 0, time.UTC) // 23:35 yesterday

	today := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)    // today's entry
	yesterday := time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC) // yesterday's entry
	older := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)     // older entry

	tests := []struct {
		name        string
		publishedAt *time.Time
		lastChecked time.Time
		now         time.Time
		want        time.Time
	}{
		{
			name:        "today's entry uses now",
			publishedAt: &today,
			lastChecked: lastChecked,
			now:         now,
			want:        now,
		},
		{
			name:        "yesterday's entry uses lastChecked",
			publishedAt: &yesterday,
			lastChecked: lastChecked,
			now:         now,
			want:        lastChecked,
		},
		{
			name:        "older entry uses original publishedAt",
			publishedAt: &older,
			lastChecked: lastChecked,
			now:         now,
			want:        older,
		},
		{
			name:        "nil publishedAt uses now",
			publishedAt: nil,
			lastChecked: lastChecked,
			now:         now,
			want:        now,
		},
		{
			name:        "no lastChecked uses original publishedAt",
			publishedAt: &yesterday,
			lastChecked: time.Time{},
			now:         now,
			want:        yesterday,
		},
		{
			name:        "no lastChecked with today entry uses original publishedAt",
			publishedAt: &today,
			lastChecked: time.Time{},
			now:         now,
			want:        today,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fragmentPublishedAt(tt.publishedAt, tt.lastChecked, tt.now)
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchRSSResource_FragmentFeed_Dedup(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	todayDate := time.Now().UTC().Format(time.RFC1123Z)

	feedV1 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Fragment A</p><p>Fragment B</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	feedV2 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Fragment A</p><p>Fragment B</p><p>Fragment C new</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	callCount := 0
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		callCount++
		if callCount == 1 {
			w.Write([]byte(feedV1))
		} else {
			w.Write([]byte(feedV2))
		}
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "fragment-rss", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	// First fetch — creates 2 fragments
	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("First FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries1, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries1) != 2 {
		t.Fatalf("first fetch: got %d entries, want 2", len(entries1))
	}

	// Simulate successful check (sets last_checked)
	RecordSuccess(app, resource)

	// Second fetch — same entry but with 1 new fragment
	err = FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("Second FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries2, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries2) != 3 {
		t.Fatalf("second fetch: got %d entries, want 3 (2 existing + 1 new)", len(entries2))
	}
}

func TestFindSimilarFragEntry(t *testing.T) {
	today := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	yesterday := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)

	existing := []existingFragEntry{
		{id: "e1", title: "Sonnet 4.6 is here. Link", publishedAt: today},
		{id: "e2", title: "Need to read: Harness Engineering", publishedAt: today},
		{id: "e3", title: "Something from yesterday", publishedAt: yesterday},
	}

	tests := []struct {
		name        string
		title       string
		publishedAt *time.Time
		wantID      string
	}{
		{
			name:        "case difference matches",
			title:       "Sonnet 4.6 is here. link",
			publishedAt: &today,
			wantID:      "e1",
		},
		{
			name:        "extra words still matches",
			title:       "Need to read: Harness Engineering, via Martin Fowler.",
			publishedAt: &today,
			wantID:      "e2",
		},
		{
			name:        "completely different no match",
			title:       "Context engineering pattern",
			publishedAt: &today,
			wantID:      "",
		},
		{
			name:        "similar but different day no match",
			title:       "Something from yesterday",
			publishedAt: &today,
			wantID:      "",
		},
		{
			name:        "nil publishedAt no match",
			title:       "Sonnet 4.6 is here. Link",
			publishedAt: nil,
			wantID:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSimilarFragEntry(existing, tt.title, tt.publishedAt)
			if tt.wantID == "" {
				if got != nil {
					t.Errorf("expected nil, got entry %q", got.id)
				}
			} else {
				if got == nil {
					t.Fatal("expected a match, got nil")
				}
				if got.id != tt.wantID {
					t.Errorf("got entry %q, want %q", got.id, tt.wantID)
				}
			}
		})
	}
}

func TestUpdateFragEntry(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Old Title", "https://example.com/frag", "old-guid")

	err := updateFragEntry(app, entry.Id, "New Title", "new-guid", "<p>New content</p>")
	if err != nil {
		t.Fatalf("updateFragEntry returned error: %v", err)
	}

	updated, err := app.FindRecordById("entries", entry.Id)
	if err != nil {
		t.Fatalf("failed to find updated entry: %v", err)
	}

	if got := updated.GetString("title"); got != "New Title" {
		t.Errorf("title = %q, want %q", got, "New Title")
	}
	if got := updated.GetString("guid"); got != "new-guid" {
		t.Errorf("guid = %q, want %q", got, "new-guid")
	}
	if got := updated.GetString("raw_content"); got != "<p>New content</p>" {
		t.Errorf("raw_content = %q, want %q", got, "<p>New content</p>")
	}
}

func TestFetchRSSResource_FragmentFeed_SimilarDedup(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	todayDate := time.Now().UTC().Format(time.RFC1123Z)

	// First version: "Sonnet 4.6 is here. Link"
	feedV1 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Sonnet 4.6 is here. Link</p><p>WebMCP might be very big.</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	// Second version: minor edit "Sonnet 4.6 is here. link" (lowercase)
	feedV2 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Sonnet 4.6 is here. link</p><p>WebMCP might be very big.</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	callCount := 0
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		callCount++
		if callCount == 1 {
			w.Write([]byte(feedV1))
		} else {
			w.Write([]byte(feedV2))
		}
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "fragment-rss", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	// First fetch — creates 2 fragments
	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("First FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries1, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries1) != 2 {
		t.Fatalf("first fetch: got %d entries, want 2", len(entries1))
	}

	RecordSuccess(app, resource)

	// Second fetch — same content with minor edit, should NOT create duplicates
	err = FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("Second FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries2, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries2) != 2 {
		t.Fatalf("second fetch: got %d entries, want 2 (similar dedup), got %d", len(entries2), len(entries2))
	}
}


func TestFetchRSSResource_FragmentFeed_UnchangedContentSkipped(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	todayDate := time.Now().UTC().Format(time.RFC1123Z)

	feed := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Fragment A</p><p>Fragment B</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(feed))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "fragment-rss", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	// First fetch — creates 2 fragments and stores content hash
	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("First FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries1, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries1) != 2 {
		t.Fatalf("first fetch: got %d entries, want 2", len(entries1))
	}

	RecordSuccess(app, resource)

	// Reload resource to get stored fragment_hashes
	resource, _ = app.FindRecordById("resources", resource.Id)
	storedHashes := resource.GetString("fragment_hashes")
	if storedHashes == "" {
		t.Fatal("expected fragment_hashes to be stored after first fetch")
	}

	// Second fetch — identical content, should be skipped entirely (no new entries)
	err = FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("Second FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries2, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries2) != 2 {
		t.Fatalf("second fetch with unchanged content: got %d entries, want 2", len(entries2))
	}
}

func TestFetchRSSResource_FragmentFeed_ChangedContentProcessed(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	todayDate := time.Now().UTC().Format(time.RFC1123Z)

	feedV1 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Fragment A</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	feedV2 := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Moments Feed</title>
    <item>
      <title>Today Moments</title>
      <link>https://example.com/moments/today</link>
      <guid>moments-today</guid>
      <description><![CDATA[<p>Fragment A</p><p>Brand new fragment</p>]]></description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, todayDate)

	callCount := 0
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		callCount++
		if callCount == 1 {
			w.Write([]byte(feedV1))
		} else {
			w.Write([]byte(feedV2))
		}
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "fragment-rss", feedServer.URL, "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	// First fetch — creates 1 fragment
	err := FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("First FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries1, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries1) != 1 {
		t.Fatalf("first fetch: got %d entries, want 1", len(entries1))
	}

	RecordSuccess(app, resource)
	resource, _ = app.FindRecordById("resources", resource.Id)

	// Second fetch — content changed (new fragment added), should create the new one
	err = FetchResource(app, resource, feedServer.Client())
	if err != nil {
		t.Fatalf("Second FetchResource returned error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	entries2, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries2) != 2 {
		t.Fatalf("second fetch with changed content: got %d entries, want 2 (1 existing + 1 new)", len(entries2))
	}

	// Verify the hash was updated
	resource, _ = app.FindRecordById("resources", resource.Id)
	updatedHashes := resource.GetString("fragment_hashes")
	if updatedHashes == "" {
		t.Fatal("expected fragment_hashes to be updated")
	}
}