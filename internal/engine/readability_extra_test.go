package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractContent_Fallback(t *testing.T) {
	// Serve something that readability can't extract content from
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><nav>just nav</nav></body></html>`))
	}))
	defer server.Close()

	result, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should at least have a title (even if content is minimal)
	_ = result
}

func TestExtractContentFromHTML_InvalidURL(t *testing.T) {
	result := ExtractContentFromHTML("<html><body><p>content</p></body></html>", "://invalid")
	// Should not panic
	_ = result
}

func TestExtractContentFromHTML_ReadabilityError(t *testing.T) {
	// Empty HTML that may cause readability to fail
	result := ExtractContentFromHTML("", "https://example.com")
	_ = result
}

func TestExtractContentFromHTML_WithRichContent(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Rich Article</title></head>
<body>
  <header><nav>Navigation</nav></header>
  <main>
    <article>
      <h1>Rich Article</h1>
      <p>This is a comprehensive article with multiple paragraphs for readability.</p>
      <p>The second paragraph provides additional details about the topic.</p>
      <p>The third paragraph concludes with key takeaways for the reader.</p>
      <p>A fourth paragraph ensures enough text mass for extraction.</p>
    </article>
  </main>
  <footer>Copyright 2024</footer>
</body>
</html>`

	result := ExtractContentFromHTML(html, "https://example.com/rich")
	if result.Content == "" {
		t.Error("expected content from rich HTML")
	}
}

func TestExtractContent_ReadabilityFallback(t *testing.T) {
	// Return valid HTML but with minimal extractable content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Page Title</title></head><body>` + strings.Repeat("<p>x</p>", 200) + `</body></html>`))
	}))
	defer server.Close()

	result, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestExtractContent_ConnectionError(t *testing.T) {
	// Use a URL that won't connect
	_, err := ExtractContent("http://127.0.0.1:1", &http.Client{})
	if err == nil {
		t.Error("expected error for connection refused")
	}
}
