package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testArticleHTML = `<!DOCTYPE html>
<html>
<head><title>Test Article Title</title></head>
<body>
  <article>
    <h1>Test Article Title</h1>
    <p>This is a comprehensive test article with enough content to be extracted by readability.
    It contains multiple paragraphs to ensure the algorithm picks it up properly.</p>
    <p>The second paragraph adds more substance to the article, helping the readability
    algorithm determine this is the main content of the page rather than navigation or ads.</p>
    <p>A third paragraph with even more details about the topic at hand. This should
    give the extraction algorithm plenty of text to work with.</p>
  </article>
</body>
</html>`

func TestExtractContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testArticleHTML))
	}))
	defer server.Close()

	result, err := ExtractContent(server.URL, server.Client())
	if err != nil {
		t.Fatalf("ExtractContent returned error: %v", err)
	}

	if result.Title == "" {
		t.Error("Title should not be empty")
	}
	if result.Content == "" {
		t.Error("Content should not be empty")
	}
	if !strings.Contains(result.Content, "comprehensive test article") {
		t.Error("Content should contain article text")
	}
}

func TestExtractContent_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := ExtractContent(server.URL, server.Client())
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestExtractContent_InvalidURL(t *testing.T) {
	_, err := ExtractContent("://invalid", &http.Client{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestExtractContentFromHTML(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		sourceURL   string
		wantContent bool
	}{
		{
			name:        "extracts from valid HTML",
			html:        testArticleHTML,
			sourceURL:   "https://example.com/article",
			wantContent: true,
		},
		{
			name:        "handles minimal HTML",
			html:        "<html><body><p>Short content</p></body></html>",
			sourceURL:   "https://example.com",
			wantContent: false, // readability may not extract from minimal HTML
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractContentFromHTML(tt.html, tt.sourceURL)
			if tt.wantContent && result.Content == "" {
				t.Error("expected content to be extracted")
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
		{"whitespace trimmed", "  hello  ", 20, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}
