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
	RegisterDailyNewsRoutes(se)

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

func TestDailyNewsRegisteredRoutesRequireAuthAndServeAuthenticatedSettings(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	mux := buildMux(t, app)

	unauth := httptest.NewRequest("GET", "/api/daily-news/settings", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated daily news settings denial, got %d body=%s", unauthRec.Code, unauthRec.Body.String())
	}

	token := createAuthToken(t, app)
	auth := httptest.NewRequest("GET", "/api/daily-news/settings", nil)
	auth.Header.Set("Authorization", token)
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, auth)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected authenticated settings success, got %d body=%s", authRec.Code, authRec.Body.String())
	}
	var settings DailyNewsSettingsDTO
	if err := json.Unmarshal(authRec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.User == "" || settings.GenerationTime != "08:00" || settings.Timezone == "" {
		t.Fatalf("expected materialized default settings, got %+v", settings)
	}
}

func TestDailyNewsRegisteredDigestDetailRouteReturnsOwnedDigest(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	settingsReq := httptest.NewRequest("GET", "/api/daily-news/settings", nil)
	settingsReq.Header.Set("Authorization", token)
	settingsRec := httptest.NewRecorder()
	mux.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("settings failed: %d body=%s", settingsRec.Code, settingsRec.Body.String())
	}
	var settings DailyNewsSettingsDTO
	if err := json.Unmarshal(settingsRec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	digest := testutil.CreateDailyDigest(t, app, settings.User, "2026-05-08", "success", "automatic")
	digest.Set("title", "Route Digest")
	digest.Set("body_markdown", "# Route Digest")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/daily-news/digests/"+digest.Id, nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected digest detail success, got %d body=%s", rec.Code, rec.Body.String())
	}
	var dto DailyNewsDigestDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode digest: %v", err)
	}
	if dto.ID != digest.Id || dto.User != settings.User || dto.Title != "Route Digest" {
		t.Fatalf("unexpected digest dto: %+v", dto)
	}
}

func TestDailyNewsRegisteredSettingsPutGenerateAndRegenerateRoutes(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	putBody := strings.NewReader(`{"enabled":false,"generation_time":"09:15","timezone":"UTC","extra_instructions":"focus on infrastructure"}`)
	putReq := httptest.NewRequest("PUT", "/api/daily-news/settings", putBody)
	putReq.Header.Set("Authorization", token)
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected settings PUT success, got %d body=%s", putRec.Code, putRec.Body.String())
	}
	var settings DailyNewsSettingsDTO
	if err := json.Unmarshal(putRec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.GenerationTime != "09:15" || settings.Timezone != "UTC" || settings.ExtraInstructions != "focus on infrastructure" || settings.Enabled {
		t.Fatalf("unexpected saved settings: %+v", settings)
	}

	generateReq := httptest.NewRequest("POST", "/api/daily-news/generate", nil)
	generateReq.Header.Set("Authorization", token)
	generateRec := httptest.NewRecorder()
	mux.ServeHTTP(generateRec, generateReq)
	if generateRec.Code != http.StatusAccepted {
		t.Fatalf("expected generate accepted, got %d body=%s", generateRec.Code, generateRec.Body.String())
	}

	digest := testutil.CreateDailyDigest(t, app, settings.User, "2026-05-08", "success", "manual")
	digest.Set("period_start", "2026-05-07T08:00:00Z")
	digest.Set("period_end", "2026-05-08T08:00:00Z")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}
	regenReq := httptest.NewRequest("POST", "/api/daily-news/digests/"+digest.Id+"/regenerate", nil)
	regenReq.Header.Set("Authorization", token)
	regenRec := httptest.NewRecorder()
	mux.ServeHTTP(regenRec, regenReq)
	if regenRec.Code != http.StatusAccepted {
		t.Fatalf("expected regenerate accepted, got %d body=%s", regenRec.Code, regenRec.Body.String())
	}
}

func TestDailyNewsRegisteredDigestRouteReturnsNullableEmptyState(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	mux := buildMux(t, app)
	token := createAuthToken(t, app)

	req := httptest.NewRequest("GET", "/api/daily-news/digests", nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected empty digest list success, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode digest list: %v", err)
	}
	if body["latest"] != nil || body["selected"] != nil {
		t.Fatalf("expected null latest/selected, got %s", rec.Body.String())
	}
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
