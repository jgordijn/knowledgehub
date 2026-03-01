package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractContent_NonOKStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := ExtractContent(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 404 status")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("error should mention HTTP status: %v", err)
	}
}

func TestExtractContent_EmptyHTMLBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Empty</title></head><body></body></html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = extracted
}

func TestExtractContent_RichArticle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
<head><title>Test Article</title></head>
<body>
<article>
<h1>Test Article</h1>
<p>This is a comprehensive article about Go programming. It covers various topics including concurrency, goroutines, channels, and the standard library. Go was designed at Google.</p>
<p>The language is known for its simplicity and efficiency.</p>
</article>
</body>
</html>`))
	}))
	defer server.Close()

	extracted, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestExtractContentFromHTML_ValidArticle(t *testing.T) {
	html := `<html>
<head><title>Test Page</title></head>
<body>
<article>
<h1>Test Page</h1>
<p>This is a test page with article content about programming and testing.</p>
<p>It has multiple paragraphs to ensure readability can extract it properly.</p>
</article>
</body>
</html>`

	result := ExtractContentFromHTML(html, "https://example.com/test")
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestExtractContentFromHTML_EmptyContentFallback(t *testing.T) {
	html := `<html><head><title>Empty</title></head><body></body></html>`
	result := ExtractContentFromHTML(html, "https://example.com")
	_ = result // Should handle gracefully
}

func TestTruncate_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello..."},
		{"empty string", "", 10, ""},
		{"whitespace stripped", "  hello  ", 20, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestExtractContent_NetworkFailure(t *testing.T) {
	_, err := ExtractContent("http://invalid.test.localhost:99999/nonexistent", http.DefaultClient)
	if err == nil {
		t.Error("expected error for unreachable URL")
	}
}
