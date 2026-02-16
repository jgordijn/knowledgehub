package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pocketbase/pocketbase/core"
)

// ScrapedLink holds a discovered article link.
type ScrapedLink struct {
	Title string
	URL   string
}

// ScrapeArticleLinks fetches a page and extracts article links using the
// resource's article_selector CSS selector, or a heuristic if none is set.
func ScrapeArticleLinks(app core.App, resource *core.Record, client *http.Client) ([]ScrapedLink, error) {
	pageURL := resource.GetString("url")
	selector := resource.GetString("article_selector")
	resourceID := resource.Id

	resp, err := client.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, pageURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	baseURL, _ := url.Parse(pageURL)
	var links []ScrapedLink

	if selector != "" {
		doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
			link := extractLink(s, baseURL)
			if link.URL != "" {
				links = append(links, link)
			}
		})
	} else {
		// Heuristic: find all <a> tags with href containing path segments
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			link := extractLink(s, baseURL)
			if link.URL != "" && isArticleLink(link.URL, pageURL) {
				links = append(links, link)
			}
		})
	}

	// Deduplicate by URL against existing entries
	links, err = deduplicateLinks(app, resourceID, links)
	if err != nil {
		return nil, fmt.Errorf("deduplicating: %w", err)
	}

	return links, nil
}

func extractLink(s *goquery.Selection, baseURL *url.URL) ScrapedLink {
	href, exists := s.Attr("href")

	// If this element isn't an anchor, look for a nested one
	if (!exists || href == "") && !s.Is("a") {
		anchor := s.Find("a[href]").First()
		if anchor.Length() > 0 {
			href, exists = anchor.Attr("href")
		}
	}

	if !exists || href == "" {
		return ScrapedLink{}
	}

	resolved := resolveURL(baseURL, href)
	if resolved == "" {
		return ScrapedLink{}
	}

	title := strings.TrimSpace(s.Text())
	if title == "" {
		title = resolved
	}
	return ScrapedLink{Title: title, URL: resolved}
}

func resolveURL(base *url.URL, href string) string {
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	// Only http/https
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

// isArticleLink applies heuristics to filter article-like links.
func isArticleLink(linkURL, pageURL string) bool {
	// Skip links that point to the same page
	if linkURL == pageURL || linkURL == pageURL+"/" {
		return false
	}
	// Skip fragment-only links
	parsed, err := url.Parse(linkURL)
	if err != nil {
		return false
	}
	// Must be on the same host or a subpath
	baseParsed, _ := url.Parse(pageURL)
	if parsed.Host != baseParsed.Host {
		return false
	}
	// Skip common non-article paths
	path := strings.ToLower(parsed.Path)
	skipPrefixes := []string{"/tag", "/category", "/author", "/page/", "/wp-content", "/wp-admin", "/feed", "/rss", "#"}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	// Must have a path beyond /
	return len(parsed.Path) > 1
}

func deduplicateLinks(app core.App, resourceID string, links []ScrapedLink) ([]ScrapedLink, error) {
	records, err := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resourceID})
	if err != nil {
		return nil, err
	}
	existingURLs := make(map[string]bool, len(records))
	for _, r := range records {
		existingURLs[r.GetString("url")] = true
	}

	var deduped []ScrapedLink
	seen := make(map[string]bool)
	for _, l := range links {
		if existingURLs[l.URL] || seen[l.URL] {
			continue
		}
		seen[l.URL] = true
		deduped = append(deduped, l)
	}
	return deduped, nil
}
