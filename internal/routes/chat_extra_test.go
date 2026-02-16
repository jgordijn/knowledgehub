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
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestHandleChat_Integration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Set up mock AI server
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Test response"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test Article", "https://example.com/a", "g1")
	entry.Set("raw_content", "Article content about testing in Go.")
	app.Save(entry)

	// Test the HandleChatDirect function
	req := ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "What is this about?"}},
	}

	recorder := httptest.NewRecorder()

	err := HandleChatDirect(app, recorder, req, aiServer.URL)
	if err != nil {
		t.Fatalf("HandleChatDirect error: %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "Test response") {
		t.Errorf("response should contain AI output, got: %s", body)
	}
	if !strings.Contains(body, "[DONE]") {
		t.Errorf("response should contain [DONE], got: %s", body)
	}
}

func TestHandleChat_InvalidBody(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	recorder := httptest.NewRecorder()

	err := HandleChatDirect(app, recorder, ChatRequestBody{}, "")
	if err == nil {
		t.Error("expected validation error for empty body")
	}
}

func TestHandleChat_EntryNotFound(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	recorder := httptest.NewRecorder()

	req := ChatRequestBody{
		EntryID:  "nonexistent",
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	}

	err := HandleChatDirect(app, recorder, req, "")
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
}

func TestHandleChat_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "g1")

	recorder := httptest.NewRecorder()

	req := ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	}

	err := HandleChatDirect(app, recorder, req, "")
	if err == nil {
		t.Error("expected error when no API key is configured")
	}
}

func TestHandleChat_StreamError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Server that returns an error
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "g1")
	entry.Set("raw_content", "Content")
	app.Save(entry)

	recorder := httptest.NewRecorder()
	req := ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	}

	// This will write an error SSE event but still complete
	err := HandleChatDirect(app, recorder, req, aiServer.URL)
	// The function should handle the stream error gracefully
	_ = err

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Errorf("should still write [DONE] after stream error, got: %s", body)
	}
}

func TestChatRequestBody_JSON(t *testing.T) {
	body := ChatRequestBody{
		EntryID: "abc123",
		Messages: []ai.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ChatRequestBody
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.EntryID != body.EntryID {
		t.Errorf("EntryID = %q, want %q", decoded.EntryID, body.EntryID)
	}
	if len(decoded.Messages) != len(body.Messages) {
		t.Errorf("messages len = %d, want %d", len(decoded.Messages), len(body.Messages))
	}
}

func TestChatHTTPHandler(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Response"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "g1")
	entry.Set("raw_content", "Test content.")
	app.Save(entry)

	// Test the HTTP handler directly
	handler := NewChatHandler(app, aiServer.URL)

	body, _ := json.Marshal(ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	})

	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Response") {
		t.Errorf("body should contain response, got: %s", w.Body.String())
	}
}

func TestChatHTTPHandler_InvalidJSON(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	handler := NewChatHandler(app, "")

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
