package engine

import (
	"log"
	"net/http"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/pocketbase/pocketbase/core"
)

// FetchResource fetches new entries for a resource and creates them in the DB.
// It handles both RSS feeds and watchlist (scraper) resources.
func FetchResource(app core.App, resource *core.Record, client *http.Client) error {
	resourceType := resource.GetString("type")

	switch resourceType {
	case "rss":
		return fetchRSSResource(app, resource, client)
	case "watchlist":
		return fetchWatchlistResource(app, resource, client)
	default:
		log.Printf("Unknown resource type: %s for resource %s", resourceType, resource.Id)
		return nil
	}
}

func fetchRSSResource(app core.App, resource *core.Record, client *http.Client) error {
	entries, err := FetchRSS(app, resource, client)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := createEntry(app, resource.Id, entry.Title, entry.URL, entry.GUID, entry.Content, entry.PublishedAt); err != nil {
			log.Printf("Failed to create entry %s: %v", entry.URL, err)
		}
	}

	return nil
}

func fetchWatchlistResource(app core.App, resource *core.Record, client *http.Client) error {
	links, err := ScrapeArticleLinks(app, resource, client)
	if err != nil {
		return err
	}

	for _, link := range links {
		// Extract full content for each discovered article
		extracted, err := ExtractContent(link.URL, client)
		if err != nil {
			log.Printf("Failed to extract content from %s: %v", link.URL, err)
			extracted = ExtractedContent{Title: link.Title}
		}

		title := extracted.Title
		if title == "" {
			title = link.Title
		}

		if err := createEntry(app, resource.Id, title, link.URL, link.URL, extracted.Content, nil); err != nil {
			log.Printf("Failed to create entry %s: %v", link.URL, err)
		}
	}

	return nil
}

func createEntry(app core.App, resourceID, title, entryURL, guid, content string, publishedAt *time.Time) error {
	collection, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("resource", resourceID)
	record.Set("title", title)
	record.Set("url", entryURL)
	record.Set("guid", guid)
	record.Set("raw_content", content)
	record.Set("discovered_at", time.Now().UTC().Format(time.RFC3339))
	record.Set("processing_status", "pending")
	record.Set("is_read", false)

	if publishedAt != nil {
		record.Set("published_at", publishedAt.Format(time.RFC3339))
	}

	if err := app.Save(record); err != nil {
		return err
	}

	// Trigger AI summarization in the background
	go processEntry(app, record)

	return nil
}

func processEntry(app core.App, record *core.Record) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered panic in processEntry for %s: %v", record.Id, r)
		}
	}()

	err := ai.SummarizeAndScore(app, record)
	if err != nil {
		log.Printf("AI processing failed for entry %s: %v", record.Id, err)
		record.Set("processing_status", "failed")
		if saveErr := app.Save(record); saveErr != nil {
			log.Printf("Failed to update processing_status: %v", saveErr)
		}
		return
	}

	// Check if preference regeneration is needed
	ai.CheckAndRegeneratePreferences(app)
}
