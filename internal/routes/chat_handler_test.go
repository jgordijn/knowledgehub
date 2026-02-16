package routes

import (
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestNewChatHandler_ValidationError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	handler := NewChatHandler(app, "")

	// Request with missing entry_id - should get 400
	req := makeTestRequest(t, `{"entry_id":"","messages":[{"role":"user","content":"test"}]}`)
	w := executeHandler(handler, req)

	// Handler should return error status since entry_id is empty
	if w.Code != 400 && w.Code != 500 {
		// The handler might write an error SSE event instead of HTTP status
		// depending on which validation kicks in first
		body := w.Body.String()
		if !strings.Contains(body, "error") && !strings.Contains(body, "required") {
			t.Errorf("expected error response, got status %d body: %s", w.Code, body)
		}
	}
}

func TestNewChatHandler_EntryNotFound(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	handler := NewChatHandler(app, "")

	req := makeTestRequest(t, `{"entry_id":"nonexistent_id","messages":[{"role":"user","content":"test"}]}`)
	w := executeHandler(handler, req)

	// Should fail since entry doesn't exist
	if w.Code == 200 {
		body := w.Body.String()
		if !strings.Contains(body, "error") {
			t.Error("expected error in response")
		}
	}
}
