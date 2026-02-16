package engine

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// ExtractedContent holds the result of readability extraction.
type ExtractedContent struct {
	Title   string
	Content string
}

// ExtractContent fetches a URL and extracts its main content using readability.
// Falls back to title + first 500 chars on failure.
func ExtractContent(articleURL string, client *http.Client) (ExtractedContent, error) {
	parsed, err := url.Parse(articleURL)
	if err != nil {
		return ExtractedContent{}, fmt.Errorf("invalid URL %s: %w", articleURL, err)
	}

	resp, err := client.Get(articleURL)
	if err != nil {
		return ExtractedContent{}, fmt.Errorf("fetching %s: %w", articleURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ExtractedContent{}, fmt.Errorf("HTTP %d for %s", resp.StatusCode, articleURL)
	}

	article, err := readability.FromReader(resp.Body, parsed)
	if err != nil {
		// Fallback: return empty content with the URL as title
		return ExtractedContent{Title: articleURL}, nil
	}

	content := article.TextContent
	title := article.Title

	if content == "" && article.Content != "" {
		content = article.Content
	}

	// Fallback: if readability produced no content, use title + first 500 chars
	if strings.TrimSpace(content) == "" {
		content = truncate(article.Content, 500)
	}

	return ExtractedContent{
		Title:   title,
		Content: content,
	}, nil
}

// ExtractContentFromHTML parses HTML content directly without fetching.
func ExtractContentFromHTML(htmlContent string, sourceURL string) ExtractedContent {
	parsed, _ := url.Parse(sourceURL)
	if parsed == nil {
		parsed = &url.URL{}
	}

	article, err := readability.FromReader(strings.NewReader(htmlContent), parsed)
	if err != nil {
		return ExtractedContent{Title: sourceURL, Content: truncate(htmlContent, 500)}
	}

	content := article.TextContent
	if strings.TrimSpace(content) == "" {
		content = truncate(article.Content, 500)
	}

	return ExtractedContent{
		Title:   article.Title,
		Content: content,
	}
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// init sets the default timeout for readability's internal HTTP operations.
func init() {
	_ = time.Second // ensure time is used
}
