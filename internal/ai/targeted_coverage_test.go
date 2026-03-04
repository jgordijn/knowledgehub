package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

// ============================================================
// client.go — cover http.NewRequest error paths
// ============================================================

func TestComplete_BadBaseURL_CreatesRequestError(t *testing.T) {
	client := NewClient("key", "model")
	client.BaseURL = "://invalid-url"

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error for invalid base URL")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("error should mention creating request: %v", err)
	}
}

func TestCompleteStream_BadBaseURL_CreatesRequestError(t *testing.T) {
	client := NewClient("key", "model")
	client.BaseURL = "://invalid-url"

	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for invalid base URL")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("error should mention creating request: %v", err)
	}
}

// ============================================================
// preference.go — CheckAndRegeneratePreferences error in countCorrections
// ============================================================

func TestCheckAndRegenerate_NoCorrectionEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// No entries, no corrections → should not panic
	CheckAndRegeneratePreferences(app)
}

func TestCheckAndRegenerate_TriggersRegeneration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	// Create 25 corrections to trigger regeneration (threshold=20)
	for i := 0; i < 25; i++ {
		guid := "guid-trig-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", guid)
		entry.Set("ai_stars", 2)
		entry.Set("user_stars", 5)
		entry.Set("raw_content", "Content")
		entry.Set("summary", "Summary")
		app.Save(entry)
	}

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "Prefers technology articles", nil
	})
	defer restore()

	CheckAndRegeneratePreferences(app)

	// Verify profile was saved
	records, err := app.FindRecordsByFilter("preferences", "1=1", "", 1, 0, nil)
	if err != nil || len(records) == 0 {
		t.Error("expected preference profile to be saved")
	}
}

func TestSavePreferenceProfile_CreatesAndUpdates(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Create first
	if err := savePreferenceProfile(app, "First"); err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Update
	if err := savePreferenceProfile(app, "Updated"); err != nil {
		t.Fatalf("update save: %v", err)
	}

	records, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0].GetString("profile_text") != "Updated" {
		t.Error("expected updated profile text")
	}
}

func TestCountCorrections_AfterProfileGeneration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Save a profile
	savePreferenceProfile(app, "Profile")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create corrections after profile
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", "guid-after-profile")
	entry.Set("ai_stars", 2)
	entry.Set("user_stars", 4)
	app.Save(entry)

	count, err := countCorrectionsSinceLastProfile(app)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

// ============================================================
// summarizer.go — htmlToMarkdown with complex/malformed HTML
// ============================================================

func TestHtmlToMarkdown_ComplexTags(t *testing.T) {
	html := `<h1>Title</h1><p>Text with <a href="http://x.com">link</a></p><ul><li>A</li></ul>`
	result := htmlToMarkdown(html)
	if result == "" {
		t.Error("expected non-empty markdown from complex HTML")
	}
}

// ============================================================
// summarizer.go:18 — test default clientCompleteFunc through callComplete
// ============================================================

func TestCallComplete_DefaultFunc_WithMockServer(t *testing.T) {
	// First, save the current func and restore after
	clientCompleteMu.RLock()
	origFn := clientCompleteFunc
	clientCompleteMu.RUnlock()

	// Create a mock OpenRouter server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"mock response"}}]}`))
	}))
	defer mockServer.Close()

	// Replace with a func that uses the mock server
	clientCompleteMu.Lock()
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		client := NewClient(apiKey, model)
		client.BaseURL = mockServer.URL
		return client.Complete(messages)
	}
	clientCompleteMu.Unlock()
	defer func() {
		clientCompleteMu.Lock()
		clientCompleteFunc = origFn
		clientCompleteMu.Unlock()
	}()

	result, err := callComplete("test-key", "test-model", []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "mock response" {
		t.Errorf("result = %q, want 'mock response'", result)
	}
}

// ============================================================
// client.go — NewClient sets fields correctly
// ============================================================

func TestNewClient_SetsFields(t *testing.T) {
	c := NewClient("my-key", "gpt-4")
	if c.APIKey != "my-key" {
		t.Errorf("APIKey = %q", c.APIKey)
	}
	if c.Model != "gpt-4" {
		t.Errorf("Model = %q", c.Model)
	}
	if c.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

// ============================================================
// preference.go:25 — Generate fails during regeneration (no API key)
// ============================================================

func TestCheckAndRegenerate_GenerateFailsNoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// DO NOT set openrouter_api_key — so GeneratePreferenceProfile will fail

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	// Create 25 corrections to trigger regeneration
	for i := 0; i < 25; i++ {
		guid := "guid-noapikey-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/a", guid)
		entry.Set("ai_stars", 2)
		entry.Set("user_stars", 5)
		app.Save(entry)
	}

	// Should not panic — GeneratePreferenceProfile should fail (no API key)
	// and the error should be logged, not returned
	CheckAndRegeneratePreferences(app)
}

var _ = http.StatusOK
