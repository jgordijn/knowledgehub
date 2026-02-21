package engine

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
// It tries plain HTTP first, falling back to browser-based fetching when
// bot protection is detected (empty responses, HTTP 403/429/503).
func FetchRSS(app core.App, resource *core.Record, client *http.Client) ([]RSSEntry, error) {
	feedURL := resource.GetString("url")
	resourceID := resource.Id
	useBrowser := resource.GetBool("use_browser")

	var feedBody string

	if !useBrowser {
		body, err := fetchFeedHTTP(feedURL, client)
		if err == nil {
			if looksLikeFeedBody(body) {
				feedBody = string(body)
			} else {
				log.Printf("Feed %s returned non-feed content (likely bot protection), trying browser", feedURL)
			}
		} else if looksLikeFeedProtection(err) {
			log.Printf("Feed protection detected for %s, trying browser", feedURL)
		} else {
			return nil, err
		}
	}

	if feedBody == "" {
		body, err := BrowserFetchBodyFunc(feedURL)
		if err != nil {
			return nil, fmt.Errorf("fetching feed %s via browser: %w", feedURL, err)
		}
		feedBody = body

		// Auto-learn: mark resource for browser fetching on future calls
		if !useBrowser {
			resource.Set("use_browser", true)
			if saveErr := app.Save(resource); saveErr != nil {
				log.Printf("Failed to set use_browser for resource %s: %v", resource.Id, saveErr)
			}
			log.Printf("Marked resource %q for browser feed extraction", resource.GetString("name"))
		}
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseString(feedBody)
	if err != nil {
		return nil, fmt.Errorf("parsing feed %s: %w", feedURL, err)
	}

	existingGUIDs, err := loadExistingGUIDs(app, resourceID)
	if err != nil {
		return nil, fmt.Errorf("loading existing GUIDs: %w", err)
	}

	isFragmentFeed := resource.GetBool("fragment_feed")

	var entries []RSSEntry
	for _, item := range feed.Items {
		guid := itemGUID(item)
		if guid == "" {
			continue
		}

		// For fragment feeds, re-process today/yesterday entries to discover
		// new fragments. Dedup for individual fragments happens in fetchRSSResource.
		if isFragmentFeed {
			reprocess := false
			if item.PublishedParsed != nil {
				entryDate := item.PublishedParsed.UTC().Truncate(24 * time.Hour)
				today := time.Now().UTC().Truncate(24 * time.Hour)
				yesterday := today.AddDate(0, 0, -1)
				reprocess = entryDate.Equal(today) || entryDate.Equal(yesterday)
			}
			if !reprocess && existingGUIDs[guid] {
				continue
			}
		} else if existingGUIDs[guid] {
			continue
		}
		// Skip articles older than 12 months (avoids importing ancient history on first fetch)
		if item.PublishedParsed != nil && time.Since(item.PublishedParsed.UTC()) > 365*24*time.Hour {
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

// fetchFeedHTTP performs a plain HTTP fetch for the feed URL.
func fetchFeedHTTP(feedURL string, client *http.Client) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", feedURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching feed %s: %w", feedURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for feed %s", resp.StatusCode, feedURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading feed %s: %w", feedURL, err)
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("feed %s returned an empty response", feedURL)
	}

	return body, nil
}

// looksLikeFeedBody checks whether the HTTP response body looks like an
// RSS/Atom/JSON feed rather than an HTML page (e.g. a Cloudflare challenge).
// When Cloudflare returns a 200 OK with HTML instead of the actual feed,
// we need to detect this early and fall through to the browser fetcher.
func looksLikeFeedBody(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		return false
	}
	// RSS/Atom feeds start with <?xml or <rss or <feed
	if trimmed[0] == '<' {
		lower := strings.ToLower(trimmed[:min(200, len(trimmed))])
		if strings.HasPrefix(lower, "<?xml") ||
			strings.Contains(lower, "<rss") ||
			strings.Contains(lower, "<feed") {
			return true
		}
		// HTML responses from Cloudflare/bot protection
		if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
			return false
		}
		// Other XML — probably a feed
		return true
	}
	// JSON feeds start with {
	if trimmed[0] == '{' {
		return true
	}
	return false
}


func loadExistingGUIDs(app core.App, resourceID string) (map[string]bool, error) {
	// Only load GUIDs from the last 13 months to bound memory usage.
	// FetchRSS already skips articles older than 12 months, so this is safe.
	cutoff := time.Now().UTC().AddDate(0, -13, 0).Format(time.RFC3339)
	records, err := app.FindRecordsByFilter(
		"entries",
		"resource = {:id} && created >= {:cutoff}",
		"", 0, 0,
		map[string]any{"id": resourceID, "cutoff": cutoff},
	)
	if err != nil {
		return nil, err
	}
	guids := make(map[string]bool, len(records))
	for _, r := range records {
		g := r.GetString("guid")
		if g != "" {
			guids[g] = true
			// For fragment GUIDs (parentGUID#frag-xxx), also mark the
			// parent GUID as seen so the parent entry is not re-fetched.
			if idx := strings.Index(g, "#frag-"); idx >= 0 {
				guids[g[:idx]] = true
			}
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
