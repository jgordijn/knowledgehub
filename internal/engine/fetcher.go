package engine

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/pocketbase/pocketbase/core"
)

// maxConcurrentAI limits the number of concurrent AI processing goroutines.
var maxConcurrentAI = make(chan struct{}, 5)

// PanicCount tracks the number of recovered panics in processEntry (for monitoring).
var PanicCount atomic.Int64

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

	isFragment := resource.GetBool("fragment_feed")

	// For fragment feeds, pre-load state for per-fragment dedup and time detection
	var existingFragGUIDs map[string]bool
	var existingFrags []existingFragEntry
	var fragNow time.Time
	var fragLastChecked time.Time
	var storedHashes map[string]string
	if isFragment {
		existingFragGUIDs, _ = loadExistingGUIDs(app, resource.Id)
		existingFrags, _ = loadExistingFragEntries(app, resource.Id)
		fragNow = time.Now().UTC()
		fragLastChecked = resource.GetDateTime("last_checked").Time().UTC()
		storedHashes = loadFragmentHashes(resource)
	}

	hashesChanged := false

	for _, entry := range entries {
		// Resource may have been deleted while we were fetching content
		if _, err := app.FindRecordById("resources", resource.Id); err != nil {
			log.Printf("Resource %s was deleted during fetch, stopping", resource.Id)
			return nil
		}

		content := entry.Content

		// Fragment feeds: resolve relative links, then split into individual fragments
		if isFragment {
			// Skip re-processing when the parent entry content hasn't changed.
			// This prevents duplicate/phantom fragments from re-splitting unchanged content.
			contentHash := contentSHA256(content)
			if storedHashes[entry.GUID] == contentHash {
				continue
			}
			storedHashes[entry.GUID] = contentHash
			hashesChanged = true

			content = resolveContentLinks(content, entry.URL)
			apiKey, _ := ai.GetAPIKey(app)
			model := ai.GetModel(app)

			var fragments []Fragment
			if apiKey != "" {
				fragments = SplitFragmentsWithAI(content, apiKey, model)
			} else {
				fragments = SplitFragments(content)
			}

			for _, frag := range fragments {
				guid := FragmentGUID(entry.GUID, frag.HTML)
				if existingFragGUIDs[guid] {
					continue
				}
				publishedAt := fragmentPublishedAt(entry.PublishedAt, fragLastChecked, fragNow)

				// Check for a similar existing fragment from the same day.
				// If found, update it in place instead of creating a duplicate.
				if similar := findSimilarFragEntry(existingFrags, frag.Title, publishedAt); similar != nil {
					if err := updateFragEntry(app, similar.id, frag.Title, guid, frag.HTML); err != nil {
						log.Printf("Failed to update similar fragment entry %s: %v", similar.id, err)
					}
					continue
				}

				if err := createEntry(app, resource.Id, frag.Title, entry.URL, guid, frag.HTML, publishedAt, true); err != nil {
					log.Printf("Failed to create fragment entry: %v", err)
				}
			}
			continue
		}

		// If the feed provided no meaningful content, fetch the article directly
		if isThinContent(content) && entry.URL != "" {
			extracted, err := extractWithBrowserFallback(app, resource, entry.URL, client)
			if err != nil {
				log.Printf("Failed to extract content for %s: %v", entry.URL, err)
			} else if extracted.Content != "" {
				content = extracted.Content
			}
		}

		if err := createEntry(app, resource.Id, entry.Title, entry.URL, entry.GUID, content, entry.PublishedAt, false); err != nil {
			log.Printf("Failed to create entry %s: %v", entry.URL, err)
		}
	}

	// Persist updated content hashes for fragment feeds
	if isFragment && hashesChanged {
		if err := saveFragmentHashes(app, resource, storedHashes); err != nil {
			log.Printf("Failed to save fragment hashes for resource %s: %v", resource.Id, err)
		}
	}

	return nil
}

// isThinContent returns true when RSS feed content is too minimal to summarize.
func isThinContent(content string) bool {
	return len(strings.TrimSpace(content)) < 200
}

// fragmentPublishedAt computes the published_at for a newly discovered fragment.
// Fragments from today's entry get the current time; fragments from yesterday's
// entry get the last-checked time (they were added between the last check and
// midnight). Older entries keep their original published_at.
// When lastChecked is zero (first run), the original published_at is used.
func fragmentPublishedAt(entryPublishedAt *time.Time, lastChecked, now time.Time) *time.Time {
	if entryPublishedAt == nil {
		t := now
		return &t
	}

	if lastChecked.IsZero() {
		return entryPublishedAt
	}

	today := now.Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	entryDate := entryPublishedAt.UTC().Truncate(24 * time.Hour)

	if entryDate.Equal(today) {
		t := now
		return &t
	}

	if entryDate.Equal(yesterday) {
		t := lastChecked
		return &t
	}

	return entryPublishedAt
}

func fetchWatchlistResource(app core.App, resource *core.Record, client *http.Client) error {
	links, err := ScrapeArticleLinks(app, resource, client)
	if err != nil {
		return err
	}

	for _, link := range links {
		// Resource may have been deleted while we were fetching content
		if _, err := app.FindRecordById("resources", resource.Id); err != nil {
			log.Printf("Resource %s was deleted during fetch, stopping", resource.Id)
			return nil
		}

		// Extract full content for each discovered article
		extracted, err := extractWithBrowserFallback(app, resource, link.URL, client)
		if err != nil {
			log.Printf("Failed to extract content from %s: %v", link.URL, err)
			extracted = ExtractedContent{Title: link.Title}
		}

		title := extracted.Title
		if title == "" {
			title = link.Title
		}

		if err := createEntry(app, resource.Id, title, link.URL, link.URL, extracted.Content, nil, false); err != nil {
			log.Printf("Failed to create entry %s: %v", link.URL, err)
		}
	}

	return nil
}

func createEntry(app core.App, resourceID, title, entryURL, guid, content string, publishedAt *time.Time, isFragment bool) error {
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
	record.Set("is_fragment", isFragment)

	if publishedAt != nil {
		record.Set("published_at", publishedAt.Format(time.RFC3339))
	}

	if err := app.Save(record); err != nil {
		return err
	}

	// Trigger AI processing in the background (bounded concurrency)
	maxConcurrentAI <- struct{}{}
	go func() {
		defer func() { <-maxConcurrentAI }()
		processEntry(app, record)
	}()

	return nil
}

func processEntry(app core.App, record *core.Record) {
	defer func() {
		if r := recover(); r != nil {
			PanicCount.Add(1)
			log.Printf("PANIC in processEntry for %s: %v\n%s", record.Id, r, debug.Stack())
		}
	}()

	var err error
	if record.GetBool("is_fragment") {
		err = ai.ScoreOnly(app, record)
	} else {
		err = ai.SummarizeAndScore(app, record)
	}
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

// existingFragEntry holds metadata about an existing fragment entry for similarity matching.
type existingFragEntry struct {
	id          string
	title       string
	publishedAt time.Time
}

// similarityThreshold is the minimum word-level Jaccard similarity for two
// fragment titles to be considered duplicates of each other.
const similarityThreshold = 0.6

// loadExistingFragEntries loads all fragment entries for a resource.
func loadExistingFragEntries(app core.App, resourceID string) ([]existingFragEntry, error) {
	records, err := app.FindRecordsByFilter("entries", "resource = {:id} && is_fragment = true", "", 0, 0, map[string]any{"id": resourceID})
	if err != nil {
		return nil, err
	}
	result := make([]existingFragEntry, 0, len(records))
	for _, r := range records {
		result = append(result, existingFragEntry{
			id:          r.Id,
			title:       r.GetString("title"),
			publishedAt: r.GetDateTime("published_at").Time().UTC(),
		})
	}
	return result, nil
}

// findSimilarFragEntry finds an existing fragment from the same day whose title
// is similar enough to be considered a duplicate.
func findSimilarFragEntry(existing []existingFragEntry, title string, publishedAt *time.Time) *existingFragEntry {
	if publishedAt == nil {
		return nil
	}
	targetDate := publishedAt.UTC().Truncate(24 * time.Hour)

	var best *existingFragEntry
	var bestScore float64

	for i := range existing {
		existDate := existing[i].publishedAt.Truncate(24 * time.Hour)
		if !existDate.Equal(targetDate) {
			continue
		}
		score := titleSimilarity(existing[i].title, title)
		if score >= similarityThreshold && score > bestScore {
			best = &existing[i]
			bestScore = score
		}
	}
	return best
}

// updateFragEntry updates an existing fragment entry with new content.
func updateFragEntry(app core.App, id, title, guid, html string) error {
	record, err := app.FindRecordById("entries", id)
	if err != nil {
		return err
	}
	record.Set("title", title)
	record.Set("guid", guid)
	record.Set("raw_content", html)
	return app.Save(record)
}
