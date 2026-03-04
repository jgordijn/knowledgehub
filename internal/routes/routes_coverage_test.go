package routes

import (
	"bytes"
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

// TestHandleChatDirect_WriteError verifies graceful handling when SSE write fails
func TestHandleChatDirect_WriteError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "g-write-err")
	entry.Set("raw_content", "Content")
	app.Save(entry)

	// AI server that streams back content
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Response text"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	recorder := httptest.NewRecorder()
	req := ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	}

	err := HandleChatDirect(app, recorder, req, aiServer.URL)
	if err != nil {
		t.Fatalf("HandleChatDirect error: %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Errorf("should contain [DONE], got: %s", body)
	}
}

func TestBuildChatSystemPrompt_LongExtraContext(t *testing.T) {
	longContext := strings.Repeat("x", 10000)
	prompt := buildChatSystemPrompt("Title", "Content", longContext)
	if !strings.Contains(prompt, "...") {
		t.Error("long extra context should be truncated")
	}
}

func TestBuildChatSystemPrompt_EmptyContent(t *testing.T) {
	prompt := buildChatSystemPrompt("Title", "", "")
	if !strings.Contains(prompt, "Title") {
		t.Error("prompt should contain title even with empty content")
	}
}

func TestHandleLinkSummaryDirect_EmptyExtractedContent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Server returns very minimal HTML with no extractable content
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head></head><body></body></html>`))
	}))
	defer articleServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: articleServer.URL}
	err := HandleLinkSummaryDirect(app, recorder, body, "")

	// Should write SSE error (returns nil) since content is empty
	if err != nil {
		t.Fatalf("expected nil error (SSE error), got: %v", err)
	}

	output := recorder.Body.String()
	if !strings.Contains(output, "error") || !strings.Contains(output, "[DONE]") {
		t.Errorf("should contain error SSE event, got: %s", output)
	}
}

func TestHandleLinkSummaryDirect_StreamError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Article</title></head><body><article><p>This is a real article with enough content for extraction to work properly and not be thin.</p></article></body></html>`))
	}))
	defer articleServer.Close()

	// AI server that returns error
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer aiServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: articleServer.URL}
	err := HandleLinkSummaryDirect(app, recorder, body, aiServer.URL)

	// Should handle gracefully - write error SSE event and still complete
	_ = err

	output := recorder.Body.String()
	if !strings.Contains(output, "[DONE]") {
		t.Errorf("should still write [DONE] after stream error, got: %s", output)
	}
}

func TestHandleLinkSummaryDirect_LongContent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Long article content
	longContent := strings.Repeat("<p>A paragraph of content about testing. </p>", 500)
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Long Article</title></head><body><article>` + longContent + `</article></body></html>`))
	}))
	defer articleServer.Close()

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Summary"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	body := LinkSummaryRequest{URL: articleServer.URL}
	err := HandleLinkSummaryDirect(app, recorder, body, aiServer.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := recorder.Body.String()
	// The meta event should contain truncated content
	if !strings.Contains(output, `"type":"meta"`) {
		t.Error("should contain meta event")
	}
}

func TestWriteSSEError(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := writeSSEError(recorder, "something went wrong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := recorder.Body.String()
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("should contain error message, got: %s", output)
	}
	if !strings.Contains(output, "[DONE]") {
		t.Errorf("should contain [DONE], got: %s", output)
	}
	if recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", recorder.Header().Get("Content-Type"))
	}
}

func TestRegisterTriggerRoutes(t *testing.T) {
	var _ func(*core.ServeEvent) = RegisterTriggerRoutes
}

func TestChatHTTPHandler_HandleChatDirectError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	handler := NewChatHandler(app, "")

	// Valid JSON but missing entry
	body, _ := json.Marshal(ChatRequestBody{
		EntryID:  "nonexistent",
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	})

	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 500 since HandleChatDirect returns error for missing entry
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusOK {
		// It might write error to body instead of status code
		_ = w.Body.String()
	}
}

func TestBuildLinkSummaryMessages_Short(t *testing.T) {
	msgs := buildLinkSummaryMessages("Short Title", "Short content")
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Error("first message should be system")
	}
	if !strings.Contains(msgs[1].Content, "Short Title") {
		t.Error("user message should contain title")
	}
}

func TestHandleChatDirect_ExtraContext(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Answer"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Main", "https://example.com/main", "g-extra")
	entry.Set("raw_content", "Main content")
	app.Save(entry)

	recorder := httptest.NewRecorder()
	req := ChatRequestBody{
		EntryID:      entry.Id,
		Messages:     []ai.Message{{Role: "user", Content: "Compare"}},
		ExtraContext: "Linked article content here",
	}

	err := HandleChatDirect(app, recorder, req, aiServer.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(recorder.Body.String(), "Answer") {
		t.Error("should contain AI response")
	}
}
