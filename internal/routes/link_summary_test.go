package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestHandleLinkSummary_Success(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Mock article server
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Linked Article</title></head><body><p>This is the content of the linked article about Go programming and testing patterns.</p></body></html>`))
	}))
	defer articleServer.Close()

	// Mock AI server
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"This article covers Go testing."}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	// Override the HTTP client for content fetching
	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: articleServer.URL}
	err := HandleLinkSummaryDirect(app, recorder, body, aiServer.URL)
	if err != nil {
		t.Fatalf("HandleLinkSummaryDirect error: %v", err)
	}

	output := recorder.Body.String()

	// Should contain meta event
	if !strings.Contains(output, `"type":"meta"`) {
		t.Errorf("output should contain meta event, got: %s", output)
	}

	// Should contain streamed content
	if !strings.Contains(output, "Go testing") {
		t.Errorf("output should contain AI summary, got: %s", output)
	}

	// Should contain [DONE]
	if !strings.Contains(output, "[DONE]") {
		t.Errorf("output should contain [DONE], got: %s", output)
	}
}

func TestHandleLinkSummary_EmptyURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: ""}
	err := HandleLinkSummaryDirect(app, recorder, body, "")
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestHandleLinkSummary_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Mock article server
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head><body><p>Content here.</p></body></html>`))
	}))
	defer articleServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: articleServer.URL}
	err := HandleLinkSummaryDirect(app, recorder, body, "")

	// Should write error SSE event, not return error
	if err != nil {
		t.Fatalf("expected nil error (SSE error), got: %v", err)
	}

	output := recorder.Body.String()
	if !strings.Contains(output, "API key not configured") {
		t.Errorf("output should contain API key error, got: %s", output)
	}
}

func TestHandleLinkSummary_FetchError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: "http://invalid.test.localhost:99999/nonexistent"}
	err := HandleLinkSummaryDirect(app, recorder, body, "")

	// Should write error SSE event, not return error
	if err != nil {
		t.Fatalf("expected nil error (SSE error), got: %v", err)
	}

	output := recorder.Body.String()
	if !strings.Contains(output, "error") {
		t.Errorf("output should contain error event, got: %s", output)
	}
}

func TestBuildLinkSummaryMessages(t *testing.T) {
	messages := buildLinkSummaryMessages("Test Title", "Article content")

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("first message role = %q, want system", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "summarizes articles") {
		t.Error("system prompt should mention summarizing")
	}
	if !strings.Contains(messages[1].Content, "Test Title") {
		t.Error("user prompt should contain title")
	}
	if !strings.Contains(messages[1].Content, "Article content") {
		t.Error("user prompt should contain content")
	}
}

func TestBuildLinkSummaryMessages_TruncatesLongContent(t *testing.T) {
	longContent := strings.Repeat("x", 10000)
	messages := buildLinkSummaryMessages("Title", longContent)

	if !strings.Contains(messages[1].Content, "...") {
		t.Error("long content should be truncated")
	}
}

func TestBuildChatSystemPrompt_WithExtraContext(t *testing.T) {
	prompt := buildChatSystemPrompt("Main Article", "Main content", "Linked article content about Go")

	if !strings.Contains(prompt, "Main Article") {
		t.Error("prompt should contain main article title")
	}
	if !strings.Contains(prompt, "Main content") {
		t.Error("prompt should contain main article content")
	}
	if !strings.Contains(prompt, "Linked article content about Go") {
		t.Error("prompt should contain linked article context")
	}
	if !strings.Contains(prompt, "Linked Article") {
		t.Error("prompt should mention linked article section")
	}
	if !strings.Contains(prompt, "articles") {
		t.Error("prompt should reference multiple articles")
	}
}

func TestBuildChatSystemPrompt_WithoutExtraContext(t *testing.T) {
	prompt := buildChatSystemPrompt("Test", "Content", "")

	if !strings.Contains(prompt, "article below") {
		t.Error("single-article prompt should say 'article below'")
	}
	if strings.Contains(prompt, "Linked Article") {
		t.Error("single-article prompt should not mention linked article")
	}
}

func TestBuildChatMessages_WithExtraContext(t *testing.T) {
	userMsgs := []ai.Message{
		{Role: "user", Content: "Compare the articles"},
	}

	messages := BuildChatMessages("Main", "Main content", userMsgs, "Linked content")

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if !strings.Contains(messages[0].Content, "Linked content") {
		t.Error("system prompt should contain linked content")
	}
}

func TestBuildChatMessages_WithoutExtraContext(t *testing.T) {
	userMsgs := []ai.Message{
		{Role: "user", Content: "Hello"},
	}

	messages := BuildChatMessages("Title", "Content", userMsgs)

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if strings.Contains(messages[0].Content, "Linked Article") {
		t.Error("should not contain linked article section without extra context")
	}
}

func TestHandleChat_WithExtraContext(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request includes extra context
		var req struct {
			Messages []ai.Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Combined answer"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Main Article", "https://example.com/a", "g1")
	entry.Set("raw_content", "Main article content.")
	app.Save(entry)

	recorder := httptest.NewRecorder()
	req := ChatRequestBody{
		EntryID:      entry.Id,
		Messages:     []ai.Message{{Role: "user", Content: "Compare both articles"}},
		ExtraContext: "This is the linked article about testing in Go.",
	}

	err := HandleChatDirect(app, recorder, req, aiServer.URL)
	if err != nil {
		t.Fatalf("HandleChatDirect error: %v", err)
	}

	output := recorder.Body.String()
	if !strings.Contains(output, "Combined answer") {
		t.Errorf("response should contain AI output, got: %s", output)
	}
}

func TestRegisterLinkSummaryRoute(t *testing.T) {
	var _ func(*core.ServeEvent) = RegisterLinkSummaryRoute
}
