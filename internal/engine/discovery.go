package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FeedInfo holds a discovered feed's URL and title.
type FeedInfo struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// feedMIMETypes are the MIME types used in <link rel="alternate"> for feeds.
var feedMIMETypes = map[string]bool{
	"application/rss+xml":  true,
	"application/atom+xml": true,
	"application/feed+json": true,
	"application/json":     true, // some sites use this for JSON Feed
}

// DiscoverFeeds finds RSS/Atom/JSON Feed URLs by parsing <link rel="alternate">
// tags from the given page URL. If none are found on the page, it falls back
// to checking the site root (scheme + host).
// Returns up to the first discovered feed URL, or empty slice if none found.
func DiscoverFeeds(pageURL string, client *http.Client) ([]FeedInfo, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", pageURL, err)
	}

	// Try the article page first
	feeds, err := discoverFeedsFromURL(pageURL, parsed, client)
	if err == nil && len(feeds) > 0 {
		return feeds, nil
	}

	// Fallback: try site root
	rootURL := fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host)
	if rootURL == pageURL || rootURL == pageURL+"/" {
		return nil, nil // already tried root
	}

	feeds, err = discoverFeedsFromURL(rootURL, parsed, client)
	if err != nil {
		return nil, nil // silently fail on root fallback
	}
	return feeds, nil
}

// discoverFeedsFromURL fetches a URL and extracts feed links from its HTML.
func discoverFeedsFromURL(fetchURL string, baseURL *url.URL, client *http.Client) ([]FeedInfo, error) {
	resp, err := client.Get(fetchURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", fetchURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, fetchURL)
	}

	return extractFeedLinks(resp.Body, baseURL)
}

// extractFeedLinks parses HTML and returns feed URLs from <link rel="alternate"> tags.
func extractFeedLinks(r io.Reader, baseURL *url.URL) ([]FeedInfo, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	var feeds []FeedInfo
	doc.Find("link[rel='alternate']").Each(func(_ int, s *goquery.Selection) {
		typ, _ := s.Attr("type")
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		typ = strings.TrimSpace(strings.ToLower(typ))
		if !feedMIMETypes[typ] {
			return
		}

		// Resolve relative URLs
		resolved := resolveURL(baseURL, href)
		if resolved == "" {
			return
		}

		title, _ := s.Attr("title")

		feeds = append(feeds, FeedInfo{
			URL:   resolved,
			Title: title,
		})
	})

	return feeds, nil
}