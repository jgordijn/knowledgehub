package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// buildMux creates an HTTP mux with the custom routes registered,
// similar to what PocketBase does during OnServe.
func buildMux(t *testing.T, app core.App) http.Handler {
	t.Helper()

	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create PB router: %v", err)
	}

	se := &core.ServeEvent{
		Router: pbRouter,
	}
	se.App = app

	// Register our custom routes
	RegisterChatRoute(se)
	RegisterLinkSummaryRoute(se)
	RegisterTriggerRoutes(se)

	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatalf("failed to build mux: %v", err)
	}

	return mux
}

// createAuthToken creates a superuser auth token for testing.
func createAuthToken(t *testing.T, app core.App) string {
	t.Helper()

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("superusers collection not found: %v", err)
	}

	// Create a superuser
	su := core.NewRecord(superusers)
	su.SetEmail("test@example.com")
	su.SetPassword("testpassword123456")
	if err := app.Save(su); err != nil {
		t.Fatalf("failed to create superuser: %v", err)
	}

	token, err := su.NewStaticAuthToken(0)
	if err != nil {
		t.Fatalf("failed to create auth token: %v", err)
	}

	return token
}

// ============================================================
// RegisterTriggerRoutes — POST /api/trigger/all
// ============================================================

func TestTriggerAll_Authenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Set up a fast-responding feed server so the background goroutine completes quickly
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = feedServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("POST", "/api/trigger/all", nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Wait for background goroutine
	time.Sleep(500 * time.Millisecond)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

func TestTriggerAll_Unauthenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)

	req := httptest.NewRequest("POST", "/api/trigger/all", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("expected non-200 for unauthenticated request")
	}
}

func TestTriggerSingle_Authenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Set up a fast-responding feed server
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test-feed", feedServer.URL, "rss", "healthy", 0, true)

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = feedServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("POST", "/api/trigger/"+resource.Id, nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Wait for background goroutine
	time.Sleep(500 * time.Millisecond)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

func TestTriggerSingle_NotFound(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("POST", "/api/trigger/nonexistent", nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404, body: %s", rec.Code, rec.Body.String())
	}
}

func TestTriggerSingle_Unauthenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)

	req := httptest.NewRequest("POST", "/api/trigger/someid", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("expected non-200 for unauthenticated request")
	}
}

// ============================================================
// RegisterChatRoute — POST /api/chat
// ============================================================

func TestChat_Authenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test Article", "https://example.com/a", "chat-int-guid")
	entry.Set("raw_content", "Article content for chat")
	app.Save(entry)

	// Mock AI server
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	// Override AI client to use mock server
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "Hello", nil
	})
	defer restore()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	body, _ := json.Marshal(ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "Tell me about this article"}},
	})

	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Chat endpoint should return 200 with SSE stream
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

func TestChat_InvalidBody(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader("not json"))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK && !strings.Contains(rec.Body.String(), "error") {
		t.Error("expected error response for invalid JSON body")
	}
}

func TestChat_MissingEntryID(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	body, _ := json.Marshal(ChatRequestBody{
		Messages: []ai.Message{{Role: "user", Content: "Hello"}},
	})

	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK && !strings.Contains(rec.Body.String(), "error") {
		t.Error("expected error for missing entry_id")
	}
}

// ============================================================
// RegisterLinkSummaryRoute — POST /api/link-summary
// ============================================================

func TestLinkSummary_Authenticated(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Mock article server
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head><body><article>
		<p>` + strings.Repeat("This is article content for testing. ", 20) + `</p>
		</article></body></html>`))
	}))
	defer articleServer.Close()

	// Mock AI server
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Summary"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	body, _ := json.Marshal(LinkSummaryRequest{URL: articleServer.URL})

	req := httptest.NewRequest("POST", "/api/link-summary", bytes.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Should return 200 with SSE stream
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

func TestLinkSummary_InvalidBody(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("POST", "/api/link-summary", strings.NewReader("not json"))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK && !strings.Contains(rec.Body.String(), "error") {
		t.Error("expected error for invalid body")
	}
}

func TestLinkSummary_MissingURL(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	body, _ := json.Marshal(LinkSummaryRequest{})

	req := httptest.NewRequest("POST", "/api/link-summary", bytes.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK && !strings.Contains(rec.Body.String(), "error") {
		t.Error("expected error for missing URL")
	}
}
