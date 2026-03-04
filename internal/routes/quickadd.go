package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/pocketbase/core"
)

// QuickAddRequest is the expected JSON body for the quick-add endpoint.
type QuickAddRequest struct {
	URL string `json:"url"`
}

// QuickAddRSSArticle represents an article preview from a discovered RSS feed.
type QuickAddRSSArticle struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at,omitempty"`
}

// QuickAddRSSInfo holds discovered RSS feed information.
type QuickAddRSSInfo struct {
	FeedURL  string               `json:"feed_url"`
	SiteName string               `json:"site_name"`
	Articles []QuickAddRSSArticle `json:"articles"`
}

// QuickAddResponse is the JSON response from the quick-add endpoint.
type QuickAddResponse struct {
	Entry   QuickAddEntryInfo `json:"entry"`
	RSS     *QuickAddRSSInfo  `json:"rss,omitempty"`
	Message string            `json:"message"`
}

// QuickAddEntryInfo holds basic info about the created entry.
type QuickAddEntryInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// SubscribeRequest is the expected JSON body for the subscribe endpoint.
type SubscribeRequest struct {
	FeedURL string `json:"feed_url"`
	Name    string `json:"name"`
}

// RegisterQuickAddRoutes adds the quick-add endpoints.
func RegisterQuickAddRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/quick-add", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		return handleQuickAdd(re)
	})

	se.Router.POST("/api/quick-add/subscribe", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		return handleSubscribe(re)
	})
}

func handleQuickAdd(re *core.RequestEvent) error {
	var body QuickAddRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body."})
	}

	if body.URL == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required."})
	}

	// Validate URL
	if _, err := url.ParseRequestURI(body.URL); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL."})
	}

	resp, err := HandleQuickAddDirect(re.App, body, engine.DefaultHTTPClient)
	if err != nil {
		return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return re.JSON(http.StatusOK, resp)
}

// HandleQuickAddDirect is the testable core logic for the quick-add endpoint.
func HandleQuickAddDirect(app core.App, body QuickAddRequest, client *http.Client) (*QuickAddResponse, error) {
	// Check for duplicate URL
	existing, err := app.FindRecordsByFilter("entries", "url = {:url}", "", 1, 0,
		map[string]any{"url": body.URL})
	if err == nil && len(existing) > 0 {
		return nil, fmt.Errorf("Article already exists: %s", existing[0].GetString("title"))
	}

	// Find the Quick Add resource
	quickAddResource, err := findQuickAddResource(app)
	if err != nil {
		return nil, fmt.Errorf("Quick Add resource not found. Restart the application.")
	}

	// Extract article content
	extracted, err := engine.ExtractContent(body.URL, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch article: %v", err)
	}

	title := extracted.Title
	if title == "" {
		title = body.URL
	}

	// Create entry
	entry, err := createQuickAddEntry(app, quickAddResource.Id, title, body.URL, extracted.Content)
	if err != nil {
		return nil, fmt.Errorf("Failed to create entry: %v", err)
	}

	// Trigger AI processing in background
	go func() {
		if aiErr := ai.SummarizeAndScore(app, entry); aiErr != nil {
			log.Printf("AI processing failed for quick-add entry %s: %v", entry.Id, aiErr)
			entry.Set("processing_status", "failed")
			app.Save(entry)
			return
		}
		ai.CheckAndRegeneratePreferences(app)
	}()

	response := &QuickAddResponse{
		Entry: QuickAddEntryInfo{
			ID:    entry.Id,
			Title: title,
			URL:   body.URL,
		},
		Message: fmt.Sprintf("Added: %s", title),
	}

	// Discover RSS feeds
	feeds, err := engine.DiscoverFeeds(body.URL, client)
	if err == nil && len(feeds) > 0 {
		feedURL := feeds[0].URL

		// Fetch feed preview (last 5 articles)
		articles := fetchFeedPreview(feedURL, client)

		// Derive site name from feed or URL
		siteName := feeds[0].Title
		if siteName == "" {
			parsed, _ := url.Parse(body.URL)
			if parsed != nil {
				siteName = parsed.Host
			}
		}

		response.RSS = &QuickAddRSSInfo{
			FeedURL:  feedURL,
			SiteName: siteName,
			Articles: articles,
		}
	}

	return response, nil
}

func handleSubscribe(re *core.RequestEvent) error {
	var body SubscribeRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body."})
	}

	if body.FeedURL == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Feed URL is required."})
	}
	if body.Name == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Name is required."})
	}

	resp, err := HandleSubscribeDirect(re.App, body)
	if err != nil {
		return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return re.JSON(http.StatusOK, resp)
}

// HandleSubscribeDirect is the testable core logic for the subscribe endpoint.
func HandleSubscribeDirect(app core.App, body SubscribeRequest) (map[string]string, error) {
	collection, err := app.FindCollectionByNameOrId("resources")
	if err != nil {
		return nil, fmt.Errorf("resources collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("name", body.Name)
	record.Set("url", body.FeedURL)
	record.Set("type", "rss")
	record.Set("status", "healthy")
	record.Set("active", true)
	record.Set("consecutive_failures", 0)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("Failed to create resource: %v", err)
	}


	return map[string]string{
		"message":     fmt.Sprintf("Subscribed to %s", body.Name),
		"resource_id": record.Id,
	}, nil
}

// findQuickAddResource finds the system Quick Add resource.
func findQuickAddResource(app core.App) (*core.Record, error) {
	records, err := app.FindRecordsByFilter("resources", "type = 'quickadd'", "", 1, 0)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("Quick Add resource not found")
	}
	return records[0], nil
}

// createQuickAddEntry creates an entry under the Quick Add resource.
func createQuickAddEntry(app core.App, resourceID, title, entryURL, content string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("resource", resourceID)
	record.Set("title", title)
	record.Set("url", entryURL)
	record.Set("guid", entryURL) // use URL as GUID for one-off articles
	record.Set("raw_content", content)
	record.Set("discovered_at", time.Now().UTC().Format(time.RFC3339))
	record.Set("published_at", time.Now().UTC().Format(time.RFC3339))
	record.Set("processing_status", "pending")
	record.Set("is_read", false)

	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

// fetchFeedPreview fetches an RSS feed and returns the last 5 articles.
func fetchFeedPreview(feedURL string, client *http.Client) []QuickAddRSSArticle {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		return nil
	}

	var articles []QuickAddRSSArticle
	limit := 5
	if len(feed.Items) < limit {
		limit = len(feed.Items)
	}

	for _, item := range feed.Items[:limit] {
		article := QuickAddRSSArticle{
			Title: item.Title,
			URL:   item.Link,
		}
		if item.PublishedParsed != nil {
			article.PublishedAt = item.PublishedParsed.Format("2006-01-02")
		} else if item.UpdatedParsed != nil {
			article.PublishedAt = item.UpdatedParsed.Format("2006-01-02")
		}
		articles = append(articles, article)
	}

	return articles
}
