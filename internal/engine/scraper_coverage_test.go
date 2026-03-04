package engine

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestResolveURL_ValidRelative(t *testing.T) {
	base, _ := url.Parse("https://example.com/page")
	got := resolveURL(base, "/article/one")
	if got != "https://example.com/article/one" {
		t.Errorf("resolveURL = %q, want 'https://example.com/article/one'", got)
	}
}

func TestResolveURL_AbsolutePassthrough(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	got := resolveURL(base, "https://other.com/page")
	if got != "https://other.com/page" {
		t.Errorf("resolveURL = %q", got)
	}
}

func TestResolveURL_NonHTTPScheme(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	got := resolveURL(base, "mailto:test@example.com")
	if got != "" {
		t.Errorf("expected empty for mailto scheme, got %q", got)
	}
}

func TestResolveURL_JavascriptScheme(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	got := resolveURL(base, "javascript:void(0)")
	if got != "" {
		t.Errorf("expected empty for javascript scheme, got %q", got)
	}
}

func TestIsArticleLink_Coverage(t *testing.T) {
	tests := []struct {
		name    string
		linkURL string
		pageURL string
		want    bool
	}{
		{"same page", "https://example.com/", "https://example.com/", false},
		{"same page slash", "https://example.com/blog/", "https://example.com/blog", false},
		{"different host", "https://other.com/article", "https://example.com", false},
		{"tag path", "https://example.com/tag/go", "https://example.com", false},
		{"category", "https://example.com/category/tech", "https://example.com", false},
		{"author", "https://example.com/author/john", "https://example.com", false},
		{"page pagination", "https://example.com/page/2", "https://example.com", false},
		{"wp-content", "https://example.com/wp-content/uploads/img.jpg", "https://example.com", false},
		{"wp-admin", "https://example.com/wp-admin/edit.php", "https://example.com", false},
		{"feed path", "https://example.com/feed", "https://example.com", false},
		{"rss path", "https://example.com/rss", "https://example.com", false},
		{"root only", "https://example.com/", "https://example.com", false},
		{"valid article", "https://example.com/articles/go-guide", "https://example.com", true},
		{"valid post", "https://example.com/2024/01/post", "https://example.com", true},
		{"hash fragment", "https://example.com/#section", "https://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArticleLink(tt.linkURL, tt.pageURL)
			if got != tt.want {
				t.Errorf("isArticleLink(%q, %q) = %v, want %v", tt.linkURL, tt.pageURL, got, tt.want)
			}
		})
	}
}

func TestScrapeArticleLinks_WithSelectorFilter(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<a class="post" href="/articles/one">Article One</a>
<a class="post" href="/articles/two">Article Two</a>
<a href="/about">About</a>
</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)
	resource.Set("article_selector", "a.post")
	app.Save(resource)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

func TestScrapeArticleLinks_HeuristicFiltering(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<a href="/articles/one">Article One</a>
<a href="/articles/two">Article Two</a>
<a href="/">Home</a>
<a href="/tag/go">Go Tag</a>
</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

func TestScrapeArticleLinks_DuplicateURLs(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<a href="/articles/one">Article One</a>
<a href="/articles/one">Article One Again</a>
</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link after dedup, got %d", len(links))
	}
}

func TestScrapeArticleLinks_ExistingEntryFiltered(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<a href="/articles/old">Old Article</a>
<a href="/articles/new">New Article</a>
</body></html>`))
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "blog", server.URL, "watchlist", "healthy", 0, true)
	testutil.CreateEntry(t, app, resource.Id, "Old Article", server.URL+"/articles/old", "old-guid")

	links, err := ScrapeArticleLinks(app, resource, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 new link, got %d", len(links))
	}
}

func TestScrapeArticleLinks_HTTPError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "broken", server.URL, "watchlist", "healthy", 0, true)

	_, err := ScrapeArticleLinks(app, resource, server.Client())
	if err == nil {
		t.Error("expected error for 404 status")
	}
}

func TestDeduplicateLinks_NilInput(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "watchlist", "healthy", 0, true)

	links, err := deduplicateLinks(app, resource.Id, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}
