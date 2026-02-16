package engine

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

const testBlogPage = `<!DOCTYPE html>
<html>
<head><title>Test Blog</title></head>
<body>
  <article class="post">
    <a href="/2024/article-one">Article One</a>
  </article>
  <article class="post">
    <a href="/2024/article-two">Article Two</a>
  </article>
  <a href="/about">About</a>
  <a href="/tag/go">Go Tag</a>
</body>
</html>`

const testBlogPageWithSelector = `<!DOCTYPE html>
<html>
<body>
  <div class="article-list">
    <a class="article-link" href="/posts/alpha">Alpha Post</a>
    <a class="article-link" href="/posts/beta">Beta Post</a>
  </div>
  <a href="/other">Other Link</a>
</body>
</html>`

func TestScrapeArticleLinks_WithSelector(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testBlogPageWithSelector))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.article-link")
	if err := app.Save(resource); err != nil {
		t.Fatalf("failed to update resource: %v", err)
	}

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("ScrapeArticleLinks returned error: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("got %d links, want 2", len(links))
	}

	if len(links) >= 1 {
		if links[0].Title != "Alpha Post" {
			t.Errorf("first link title = %q, want %q", links[0].Title, "Alpha Post")
		}
	}
}

func TestScrapeArticleLinks_Heuristic(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testBlogPage))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("ScrapeArticleLinks returned error: %v", err)
	}

	// Should find article links but filter out /tag/* and /about should still pass (path > 1)
	if len(links) == 0 {
		t.Error("expected at least some links from heuristic detection")
	}

	// Verify no tag links
	for _, l := range links {
		parsed, _ := url.Parse(l.URL)
		if parsed != nil && len(parsed.Path) > 4 && parsed.Path[:4] == "/tag" {
			t.Errorf("should not include tag link: %s", l.URL)
		}
	}
}

func TestScrapeArticleLinks_Deduplication(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testBlogPageWithSelector))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.article-link")
	app.Save(resource)

	// Pre-create one entry
	testutil.CreateEntry(t, app, resource.Id, "Alpha Post", server.URL+"/posts/alpha", server.URL+"/posts/alpha")

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("ScrapeArticleLinks returned error: %v", err)
	}

	if len(links) != 1 {
		t.Errorf("got %d links after dedup, want 1", len(links))
	}
}

func TestScrapeArticleLinks_ServerError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	_, err := ScrapeArticleLinks(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestIsArticleLink(t *testing.T) {
	pageURL := "https://example.com/blog"
	tests := []struct {
		name     string
		linkURL  string
		expected bool
	}{
		{"same page", "https://example.com/blog", false},
		{"same page trailing slash", "https://example.com/blog/", false},
		{"different host", "https://other.com/article", false},
		{"tag path", "https://example.com/tag/go", false},
		{"category path", "https://example.com/category/tech", false},
		{"author path", "https://example.com/author/john", false},
		{"root path", "https://example.com/", false},
		{"valid article path", "https://example.com/2024/my-article", true},
		{"valid short path", "https://example.com/about", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArticleLink(tt.linkURL, pageURL)
			if got != tt.expected {
				t.Errorf("isArticleLink(%q, %q) = %v, want %v", tt.linkURL, pageURL, got, tt.expected)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/blog")
	tests := []struct {
		name     string
		href     string
		expected string
	}{
		{"absolute URL", "https://other.com/page", "https://other.com/page"},
		{"relative path", "/2024/article", "https://example.com/2024/article"},
		{"relative without slash", "article", "https://example.com/article"},
		{"javascript link", "javascript:void(0)", ""},
		{"mailto link", "mailto:test@example.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveURL(base, tt.href)
			if got != tt.expected {
				t.Errorf("resolveURL(%q) = %q, want %q", tt.href, got, tt.expected)
			}
		})
	}
}
