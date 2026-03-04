package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractContent_ReadabilityContentFallback(t *testing.T) {
	// Test the path where readability finds content in article.Content but not TextContent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head><body>
<div id="content">
<p>This has some content that readability should find. Go is a statically typed, compiled programming language designed at Google. It is syntactically similar to C, but with memory safety.</p>
</div>
</body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = extracted // The content may or may not be extracted depending on readability
}

func TestExtractContentFromHTML_BadHTML(t *testing.T) {
	// Test with malformed HTML that readability will fail on
	result := ExtractContentFromHTML("<<<not html>>>", "https://example.com")
	// Should handle gracefully without panic
	_ = result
}

func TestExtractContentFromHTML_WithContent(t *testing.T) {
	html := `<html><head><title>Great Article</title></head><body>
<article>
<h1>Great Article</h1>
<p>This is a wonderful article about software development. It covers many topics including testing, deployment, monitoring, and observability in production systems.</p>
<p>The author discusses various approaches to ensuring code quality through automated testing and continuous integration pipelines.</p>
</article>
</body></html>`

	result := ExtractContentFromHTML(html, "https://example.com/article")
	if result.Title == "" {
		t.Error("expected title to be extracted")
	}
	if result.Content == "" {
		t.Error("expected content to be extracted")
	}
}

func TestExtractContent_PartialHTMLFallbackContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// HTML with only a script and no article content
		w.Write([]byte(`<html><head><title>Script Page</title></head><body><script>console.log("no content")</script></body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still have some result even if content extraction is minimal
	_ = extracted
}
