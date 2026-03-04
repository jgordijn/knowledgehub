package engine

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDiscoverFeeds_FoundOnArticlePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<link rel="alternate" type="application/rss+xml" title="Main Feed" href="/feed.xml">
		</head><body><p>Article</p></body></html>`))
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/blog/article", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatal("expected at least one feed")
	}
	if !strings.HasSuffix(feeds[0].URL, "/feed.xml") {
		t.Errorf("expected feed URL ending in /feed.xml, got %s", feeds[0].URL)
	}
	if feeds[0].Title != "Main Feed" {
		t.Errorf("expected title 'Main Feed', got %s", feeds[0].Title)
	}
}

func TestDiscoverFeeds_FallbackToSiteRoot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			w.Write([]byte(`<html><head>
				<link rel="alternate" type="application/atom+xml" href="/atom.xml">
			</head><body></body></html>`))
		} else {
			w.Write([]byte(`<html><head></head><body><p>Article with no feed links</p></body></html>`))
		}
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/blog/post", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatal("expected feed from site root fallback")
	}
	if !strings.HasSuffix(feeds[0].URL, "/atom.xml") {
		t.Errorf("expected feed URL ending in /atom.xml, got %s", feeds[0].URL)
	}
}

func TestDiscoverFeeds_NoFeedFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>No Feed</title></head><body></body></html>`))
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/page", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected no feeds, got %d", len(feeds))
	}
}

func TestDiscoverFeeds_RelativeURLResolution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<link rel="alternate" type="application/rss+xml" href="../rss">
		</head><body></body></html>`))
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/blog/posts/article", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatal("expected a feed")
	}
	// "../rss" relative to /blog/posts/article should resolve to /blog/rss
	if !strings.Contains(feeds[0].URL, "/blog/rss") {
		t.Errorf("expected resolved URL containing /blog/rss, got %s", feeds[0].URL)
	}
}

func TestDiscoverFeeds_MultipleFeeds_ReturnsAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<link rel="alternate" type="application/rss+xml" title="Posts" href="/feed.xml">
			<link rel="alternate" type="application/atom+xml" title="Comments" href="/comments.xml">
		</head><body></body></html>`))
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/page", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) < 2 {
		t.Fatalf("expected at least 2 feeds, got %d", len(feeds))
	}
	if !strings.HasSuffix(feeds[0].URL, "/feed.xml") {
		t.Errorf("expected first feed /feed.xml, got %s", feeds[0].URL)
	}
}

func TestDiscoverFeeds_InvalidURL(t *testing.T) {
	_, err := DiscoverFeeds("://invalid", &http.Client{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestDiscoverFeeds_JSONFeed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<link rel="alternate" type="application/feed+json" href="/feed.json">
		</head><body></body></html>`))
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/page", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatal("expected JSON Feed to be discovered")
	}
}

func TestExtractFeedLinks_EmptyHref(t *testing.T) {
	html := `<html><head><link rel="alternate" type="application/rss+xml" href=""></head></html>`
	base, _ := url.Parse("https://example.com")
	feeds, err := extractFeedLinks(strings.NewReader(html), base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected no feeds for empty href, got %d", len(feeds))
	}
}

func TestExtractFeedLinks_NoTypeAttribute(t *testing.T) {
	html := `<html><head><link rel="alternate" href="/feed.xml"></head></html>`
	base, _ := url.Parse("https://example.com")
	feeds, err := extractFeedLinks(strings.NewReader(html), base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected no feeds when type is missing, got %d", len(feeds))
	}
}

func TestDiscoverFeeds_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	feeds, err := DiscoverFeeds(srv.URL+"/page", &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both article page and root return 500, so no feeds
	if len(feeds) != 0 {
		t.Errorf("expected no feeds on server error, got %d", len(feeds))
	}
}
