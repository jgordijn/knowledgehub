package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

var (
	recentDate1 = time.Now().AddDate(0, 0, -2).UTC().Format(time.RFC1123Z)
	recentDate2 = time.Now().AddDate(0, 0, -1).UTC().Format(time.RFC1123Z)

	testRSSFeed = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Article One</title>
      <link>https://example.com/article-one</link>
      <guid>guid-1</guid>
      <description>First article description</description>
      <pubDate>%s</pubDate>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/article-two</link>
      <guid>guid-2</guid>
      <description>Second article description</description>
      <pubDate>%s</pubDate>
    </item>
  </channel>
</rss>`, recentDate1, recentDate2)

	testAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Feed</title>
  <entry>
    <title>Atom Article</title>
    <link href="https://example.com/atom-article"/>
    <id>atom-guid-1</id>
    <summary>Atom description</summary>
  </entry>
</feed>`
)

func TestFetchRSS(t *testing.T) {
	tests := []struct {
		name          string
		feedContent   string
		existingGUIDs []string
		expectedCount int
	}{
		{
			name:          "parses RSS feed with two new items",
			feedContent:   testRSSFeed,
			existingGUIDs: nil,
			expectedCount: 2,
		},
		{
			name:          "deduplicates by GUID",
			feedContent:   testRSSFeed,
			existingGUIDs: []string{"guid-1"},
			expectedCount: 1,
		},
		{
			name:          "all items already exist",
			feedContent:   testRSSFeed,
			existingGUIDs: []string{"guid-1", "guid-2"},
			expectedCount: 0,
		},
		{
			name:          "parses Atom feed",
			feedContent:   testAtomFeed,
			existingGUIDs: nil,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.Write([]byte(tt.feedContent))
			}))
			defer server.Close()

			resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

			// Pre-create existing entries for dedup testing
			for _, guid := range tt.existingGUIDs {
				testutil.CreateEntry(t, app, resource.Id, "existing", "https://example.com/existing", guid)
			}

			entries, err := FetchRSS(app, resource, server.Client())
			if err != nil {
				t.Fatalf("FetchRSS returned error: %v", err)
			}

			if len(entries) != tt.expectedCount {
				t.Errorf("got %d entries, want %d", len(entries), tt.expectedCount)
			}
		})
	}
}

func TestFetchRSS_InvalidFeed(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a valid feed"))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "bad", server.URL, "rss", "healthy", 0, true)

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for invalid feed, got nil")
	}
}

func TestFetchRSS_ServerError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "err", server.URL, "rss", "healthy", 0, true)

	_, err := FetchRSS(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}

func TestFetchRSS_EntryFields(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}

	if len(entries) < 1 {
		t.Fatal("expected at least 1 entry")
	}

	first := entries[0]
	if first.Title == "" {
		t.Error("entry Title should not be empty")
	}
	if first.URL == "" {
		t.Error("entry URL should not be empty")
	}
	if first.GUID == "" {
		t.Error("entry GUID should not be empty")
	}
	if first.PublishedAt == nil {
		t.Error("entry PublishedAt should be set")
	}
}

func TestFetchRSS_ContentField(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Feed with content:encoded (not just description)
	feedWithContent := `<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel><title>Test</title>
    <item>
      <title>Content Item</title>
      <link>https://example.com/content</link>
      <guid>content-guid</guid>
      <content:encoded><![CDATA[<p>Rich HTML content</p>]]></content:encoded>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(feedWithContent))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Content == "" {
		t.Error("expected content to be extracted")
	}
}

func TestFetchRSS_FallbackGUID(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Feed without GUID - should use link as GUID
	feedNoGUID := `<?xml version="1.0"?>
<rss version="2.0">
  <channel><title>Test</title>
    <item>
      <title>No GUID Item</title>
      <link>https://example.com/no-guid</link>
      <description>Description only</description>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(feedNoGUID))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

	entries, err := FetchRSS(app, resource, server.Client())
	if err != nil {
		t.Fatalf("FetchRSS returned error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// GUID should fall back to link
	if entries[0].GUID != "https://example.com/no-guid" {
		t.Errorf("GUID = %q, want link URL", entries[0].GUID)
	}
	// Content should fall back to description
	if entries[0].Content != "Description only" {
		t.Errorf("Content = %q, want description", entries[0].Content)
	}
}
