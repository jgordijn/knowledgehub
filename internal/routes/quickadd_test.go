package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestHandleQuickAddDirect_Success(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Create Quick Add resource
	testutil.CreateResource(t, app, "Quick Add", "https://quickadd.local", "quickadd", "healthy", 0, true)

	// Mock AI
	restore := ai.SetCompleteFunc(func(_, _ string, _ []ai.Message) (string, error) {
		return `{"summary":"Test summary","stars":4,"takeaways":["point 1"]}`, nil
	})
	defer restore()

	// Serve article content
	articleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Article</title></head><body><p>This is a great article about testing in Go with lots of content to extract.</p></body></html>`))
	}))
	defer articleSrv.Close()

	body := QuickAddRequest{URL: articleSrv.URL + "/article"}
	resp, err := HandleQuickAddDirect(app, body, &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Entry.ID == "" {
		t.Error("expected entry ID")
	}
	if resp.Entry.URL != articleSrv.URL+"/article" {
		t.Errorf("expected URL %s, got %s", articleSrv.URL+"/article", resp.Entry.URL)
	}
	if resp.Message == "" {
		t.Error("expected message")
	}

	// Verify entry was created in DB
	entry, err := app.FindRecordById("entries", resp.Entry.ID)
	if err != nil {
		t.Fatalf("entry not found in DB: %v", err)
	}
	// Status may be "pending" or "failed" depending on AI goroutine timing
	status := entry.GetString("processing_status")
	if status != "pending" && status != "done" && status != "failed" {
		t.Errorf("unexpected processing_status: %s", status)
	}
}

func TestHandleQuickAddDirect_DuplicateURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	res := testutil.CreateResource(t, app, "Quick Add", "https://quickadd.local", "quickadd", "healthy", 0, true)
	testutil.CreateEntry(t, app, res.Id, "Existing Article", "https://example.com/article", "guid1")

	body := QuickAddRequest{URL: "https://example.com/article"}
	_, err := HandleQuickAddDirect(app, body, &http.Client{})
	if err == nil {
		t.Error("expected error for duplicate URL")
	}
}

func TestHandleQuickAddDirect_WithRSSDiscovery(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateResource(t, app, "Quick Add", "https://quickadd.local", "quickadd", "healthy", 0, true)

	// Mock AI
	restore := ai.SetCompleteFunc(func(_, _ string, _ []ai.Message) (string, error) {
		return `{"summary":"Test summary","stars":3}`, nil
	})
	defer restore()

	// Serve article page with RSS link, and RSS feed
	mux := http.NewServeMux()
	mux.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<title>Test Article</title>
			<link rel="alternate" type="application/rss+xml" title="My Blog" href="/feed.xml">
		</head><body><p>Article content here with enough text to extract.</p></body></html>`))
	})
	mux.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<rss version="2.0">
			<channel>
				<title>My Blog</title>
				<item><title>Post 1</title><link>https://example.com/post1</link><pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate></item>
				<item><title>Post 2</title><link>https://example.com/post2</link><pubDate>Tue, 02 Jan 2024 00:00:00 GMT</pubDate></item>
				<item><title>Post 3</title><link>https://example.com/post3</link><pubDate>Wed, 03 Jan 2024 00:00:00 GMT</pubDate></item>
			</channel>
		</rss>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	body := QuickAddRequest{URL: srv.URL + "/article"}
	resp, err := HandleQuickAddDirect(app, body, &http.Client{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.RSS == nil {
		t.Fatal("expected RSS info in response")
	}
	if resp.RSS.SiteName != "My Blog" {
		t.Errorf("expected site name 'My Blog', got %s", resp.RSS.SiteName)
	}
	if len(resp.RSS.Articles) != 3 {
		t.Errorf("expected 3 articles, got %d", len(resp.RSS.Articles))
	}
	if resp.RSS.Articles[0].Title != "Post 1" {
		t.Errorf("expected first article 'Post 1', got %s", resp.RSS.Articles[0].Title)
	}
}

func TestHandleQuickAddDirect_InvalidURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	body := QuickAddRequest{URL: ""}
	_, err := HandleQuickAddDirect(app, body, &http.Client{})
	// Empty URL should fail at duplicate check or fetch stage
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestHandleQuickAddDirect_NoQuickAddResource(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Don't create Quick Add resource
	articleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head><body><p>Content</p></body></html>`))
	}))
	defer articleSrv.Close()

	body := QuickAddRequest{URL: articleSrv.URL + "/article"}
	_, err := HandleQuickAddDirect(app, body, &http.Client{})
	if err == nil {
		t.Error("expected error when Quick Add resource is missing")
	}
}

func TestHandleSubscribeDirect_Success(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	body := SubscribeRequest{
		FeedURL: "https://example.com/feed.xml",
		Name:    "Example Blog",
	}

	resp, err := HandleSubscribeDirect(app, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp["resource_id"] == "" {
		t.Error("expected resource_id in response")
	}

	// Verify resource was created
	record, err := app.FindRecordById("resources", resp["resource_id"])
	if err != nil {
		t.Fatalf("resource not found: %v", err)
	}
	if record.GetString("type") != "rss" {
		t.Errorf("expected type 'rss', got %s", record.GetString("type"))
	}
	if record.GetString("url") != "https://example.com/feed.xml" {
		t.Errorf("expected feed URL, got %s", record.GetString("url"))
	}
}

func TestHandleSubscribeDirect_MissingURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// PocketBase requires URL field, so empty feed_url should fail
	body := SubscribeRequest{FeedURL: "", Name: "Test"}
	_, err := HandleSubscribeDirect(app, body)
	if err == nil {
		t.Error("expected error for missing feed URL")
	}
}

func TestHandleQuickAddDirect_FetchFailure(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateResource(t, app, "Quick Add", "https://quickadd.local", "quickadd", "healthy", 0, true)

	// Server that returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	body := QuickAddRequest{URL: srv.URL + "/article"}
	_, err := HandleQuickAddDirect(app, body, &http.Client{})
	if err == nil {
		t.Error("expected error for fetch failure")
	}
}
