package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/pocketbase/pocketbase/core"
)

// LinkSummaryRequest is the expected JSON body for the link-summary endpoint.
type LinkSummaryRequest struct {
	URL string `json:"url"`
}

// RegisterLinkSummaryRoute adds the POST /api/link-summary streaming endpoint.
func RegisterLinkSummaryRoute(se *core.ServeEvent) {
	se.Router.POST("/api/link-summary", func(re *core.RequestEvent) error {
		return handleLinkSummary(re)
	})
}

func handleLinkSummary(re *core.RequestEvent) error {
	var body LinkSummaryRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if body.URL == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}

	return HandleLinkSummaryDirect(re.App, re.Response, body, "")
}

// HandleLinkSummaryDirect is the testable core logic of the link-summary endpoint.
func HandleLinkSummaryDirect(app core.App, w http.ResponseWriter, body LinkSummaryRequest, baseURL string) error {
	if body.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Fetch and extract content
	extracted, err := engine.ExtractContent(body.URL, engine.DefaultHTTPClient)
	if err != nil {
		return writeSSEError(w, fmt.Sprintf("failed to fetch article: %v", err))
	}

	if extracted.Content == "" {
		return writeSSEError(w, "could not extract content from the linked article")
	}

	apiKey, err := ai.GetAPIKey(app)
	if err != nil {
		return writeSSEError(w, "API key not configured")
	}
	model := ai.GetModel(app)

	messages := buildLinkSummaryMessages(extracted.Title, extracted.Content)

	client := ai.NewClient(apiKey, model)
	if baseURL != "" {
		client.BaseURL = baseURL
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send metadata (title + content for subsequent chat use)
	content := extracted.Content
	if len(content) > 12000 {
		content = content[:12000] + "..."
	}
	meta, _ := json.Marshal(map[string]string{
		"type":    "meta",
		"title":   extracted.Title,
		"content": content,
	})
	fmt.Fprintf(w, "data: %s\n\n", meta)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream the summary
	err = client.CompleteStream(messages, func(chunk string) error {
		data, _ := json.Marshal(map[string]string{"content": chunk})
		_, writeErr := fmt.Fprintf(w, "data: %s\n\n", data)
		if writeErr != nil {
			return writeErr
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return nil
	})

	if err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errData)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

func buildLinkSummaryMessages(title, content string) []ai.Message {
	if len(content) > 8000 {
		content = content[:8000] + "..."
	}
	prompt := fmt.Sprintf(
		"Summarize the following article in 3-5 concise sentences.\n\nArticle: %s\n\n%s",
		title, content,
	)
	return []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that summarizes articles concisely."},
		{Role: "user", Content: prompt},
	}
}

func writeSSEError(w http.ResponseWriter, msg string) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	errData, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "data: %s\n\n", errData)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}
