package engine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/pocketbase/core"
)

// RSSEntry holds a parsed feed item ready for persistence.
type RSSEntry struct {
	Title       string
	URL         string
	GUID        string
	Content     string
	PublishedAt *time.Time
}

// FetchRSS fetches and parses an RSS/Atom/JSON feed, returning new entries
// that don't already exist in the database (deduped by GUID).
func FetchRSS(app core.App, resource *core.Record, client *http.Client) ([]RSSEntry, error) {
	feedURL := resource.GetString("url")
	resourceID := resource.Id

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	fp.Client = client
	feed, err := fp.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parsing feed %s: %w", feedURL, err)
	}

	existingGUIDs, err := loadExistingGUIDs(app, resourceID)
	if err != nil {
		return nil, fmt.Errorf("loading existing GUIDs: %w", err)
	}

	var entries []RSSEntry
	for _, item := range feed.Items {
		guid := itemGUID(item)
		if guid == "" {
			continue
		}
		if existingGUIDs[guid] {
			continue
		}

		entry := RSSEntry{
			Title:   item.Title,
			URL:     itemLink(item),
			GUID:    guid,
			Content: itemContent(item),
		}
		if item.PublishedParsed != nil {
			t := item.PublishedParsed.UTC()
			entry.PublishedAt = &t
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func loadExistingGUIDs(app core.App, resourceID string) (map[string]bool, error) {
	records, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resourceID})
	if err != nil {
		return nil, err
	}
	guids := make(map[string]bool, len(records))
	for _, r := range records {
		g := r.GetString("guid")
		if g != "" {
			guids[g] = true
		}
	}
	return guids, nil
}

func itemGUID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	if item.Link != "" {
		return item.Link
	}
	return ""
}

func itemLink(item *gofeed.Item) string {
	if item.Link != "" {
		return item.Link
	}
	return ""
}

func itemContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	return item.Description
}
