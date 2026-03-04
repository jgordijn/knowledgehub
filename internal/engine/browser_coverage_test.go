package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestLooksLikeChallengePage(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{"normal page", "<html><body><h1>Article</h1></body></html>", false},
		{"verifying browser", "<html><body>Verifying your browser...</body></html>", true},
		{"checking browser", "<html><body>Checking your browser before accessing...</body></html>", true},
		{"just a moment", "<html><body>Just a moment...</body></html>", true},
		{"challenge platform", "<html><body><div class='challenge-platform'>Wait</div></body></html>", true},
		{"mixed case", "<html><body>VERIFYING YOUR BROWSER</body></html>", true},
		{"empty html", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeChallengePage(tt.html)
			if got != tt.want {
				t.Errorf("looksLikeChallengePage = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLooksLikeFeedProtection_DelegatesToBotProtection(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"403", fmt.Errorf("HTTP 403 blocked"), true},
		{"429", fmt.Errorf("HTTP 429 rate limited"), true},
		{"503", fmt.Errorf("HTTP 503 service unavailable"), true},
		{"404", fmt.Errorf("HTTP 404 not found"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeFeedProtection(tt.err)
			if got != tt.want {
				t.Errorf("looksLikeFeedProtection(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestExtractWithBrowserFallback_AlreadyUseBrowser_SaveNotCalledAgain(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)
	resource.Set("use_browser", true)
	app.Save(resource)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{
			Title:   "Browser Title",
			Content: "Browser content",
		}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	extracted, err := extractWithBrowserFallback(app, resource, "https://example.com/article", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Title != "Browser Title" {
		t.Errorf("title = %q, want 'Browser Title'", extracted.Title)
	}

	// use_browser was already true, so save shouldn't have been needed for that flag
	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should still be true")
	}
}

func TestExtractWithBrowserFallback_BotProtection_BrowserSucceeds(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "test", server.URL, "rss", "healthy", 0, true)

	oldBrowserFunc := BrowserExtractFunc
	BrowserExtractFunc = func(url string) (ExtractedContent, error) {
		return ExtractedContent{
			Title:   "Browser Title",
			Content: "Browser extracted content with enough text for thin check",
		}, nil
	}
	defer func() { BrowserExtractFunc = oldBrowserFunc }()

	extracted, err := extractWithBrowserFallback(app, resource, server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Title != "Browser Title" {
		t.Errorf("title = %q", extracted.Title)
	}

	// use_browser should be auto-learned
	updated, _ := app.FindRecordById("resources", resource.Id)
	if !updated.GetBool("use_browser") {
		t.Error("use_browser should be set after browser fallback")
	}
}
