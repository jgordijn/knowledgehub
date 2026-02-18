package engine

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestExtractLink_AnchorTag(t *testing.T) {
	base, _ := url.Parse("https://example.com")

	html := `<a href="/article/one">Article One</a>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link := extractLink(s, base)
		if link.URL != "https://example.com/article/one" {
			t.Errorf("URL = %q, want %q", link.URL, "https://example.com/article/one")
		}
		if link.Title != "Article One" {
			t.Errorf("Title = %q, want %q", link.Title, "Article One")
		}
	})
}

func TestExtractLink_NestedAnchor(t *testing.T) {
	base, _ := url.Parse("https://example.com")

	html := `<div class="card"><a href="/post/2">Nested Link</a></div>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("div.card").Each(func(_ int, s *goquery.Selection) {
		link := extractLink(s, base)
		if link.URL != "https://example.com/post/2" {
			t.Errorf("URL = %q, want %q", link.URL, "https://example.com/post/2")
		}
	})
}

func TestExtractLink_NoHref(t *testing.T) {
	base, _ := url.Parse("https://example.com")

	html := `<a>No Link</a>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link := extractLink(s, base)
		if link.URL != "" {
			t.Errorf("expected empty URL, got %q", link.URL)
		}
	})
}

func TestExtractLink_EmptyTitle(t *testing.T) {
	base, _ := url.Parse("https://example.com")

	html := `<a href="/page"></a>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link := extractLink(s, base)
		if link.Title == "" {
			t.Error("Title should not be empty (should fallback to URL)")
		}
	})
}

func TestScrapeArticleLinks_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "broken", server.URL, "watchlist", "healthy", 0, true)

	_, err := ScrapeArticleLinks(app, resource, &http.Client{})
	if err == nil {
		t.Error("expected error for closed server")
	}
}
