package ai

import (
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestSummarizeAndScore_EmptyContent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Empty Article", "https://example.com/empty", "guid-empty")
	entry.Set("raw_content", "")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	origComplete := clientCompleteFunc
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		return `{"summary":"Minimal article.","stars":2}`, nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	err := SummarizeAndScore(app, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("summary"); got != "Minimal article." {
		t.Errorf("summary = %q", got)
	}
}

func TestSummarizeAndScore_WithProfileAndCorrections(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	testutil.CreatePreference(t, app, "User loves distributed systems and Kotlin", "2024-01-01 00:00:00.000Z")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	testutil.CreateEntryWithStars(t, app, resource.Id, "CRDT Article", "https://example.com/crdt", 3, 5)

	entry := testutil.CreateEntry(t, app, resource.Id, "Kotlin Guide", "https://example.com/kotlin", "guid-kotlin")
	entry.Set("raw_content", "A comprehensive guide to Kotlin programming.")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	var capturedPrompt string
	origComplete := clientCompleteFunc
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		for _, m := range messages {
			if m.Role == "user" {
				capturedPrompt = m.Content
			}
		}
		return `{"summary":"Great Kotlin guide.","stars":5}`, nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	err := SummarizeAndScore(app, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedPrompt, "distributed systems") {
		t.Error("prompt should include preference profile")
	}
	if !strings.Contains(capturedPrompt, "CRDT Article") {
		t.Error("prompt should include recent corrections")
	}
}

func TestSummarizeAndScore_AIReturnsBadJSON(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test", "https://example.com/test", "guid-bad")
	entry.Set("raw_content", "Content")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	origComplete := clientCompleteFunc
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		return "This is not JSON at all", nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	err := SummarizeAndScore(app, entry)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}
