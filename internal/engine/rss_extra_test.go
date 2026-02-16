package engine

import (
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
