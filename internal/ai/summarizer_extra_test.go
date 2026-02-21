package ai

import (
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
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

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return `{"summary":"Minimal article.","stars":2}`, nil
	})
	defer restore()

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
	
	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		for _, m := range messages {
			if m.Role == "user" {
				capturedPrompt = m.Content
			}
		}
		return `{"summary":"Great Kotlin guide.","stars":5}`, nil
	})
	defer restore()

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

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "This is not JSON at all", nil
	})
	defer restore()

	err := SummarizeAndScore(app, entry)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestScoreOnly(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Short Fragment")
	entry.Set("url", "https://example.com/fragment")
	entry.Set("guid", "guid-frag")
	entry.Set("raw_content", "<p>A quick note about Go generics.</p>")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return `{"summary":"","stars":4}`, nil
	})
	defer restore()

	err := ScoreOnly(app, entry)
	if err != nil {
		t.Fatalf("ScoreOnly returned error: %v", err)
	}

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetInt("ai_stars"); got != 4 {
		t.Errorf("ai_stars = %d, want 4", got)
	}
	if got := updated.GetString("summary"); got != "" {
		t.Errorf("summary = %q, want empty for fragment", got)
	}
	if got := updated.GetString("processing_status"); got != "done" {
		t.Errorf("processing_status = %q, want done", got)
	}
}

func TestScoreOnly_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Fragment")
	entry.Set("url", "https://example.com/frag")
	entry.Set("guid", "guid-frag-nokey")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	err := ScoreOnly(app, entry)
	if err == nil {
		t.Error("expected error when no API key, got nil")
	}
}

func TestBuildScoreOnlyPrompt(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		content     string
		profile     string
		corrections string
		wantParts   []string
		noParts     []string
	}{
		{
			name:    "basic score prompt",
			title:   "Quick Note",
			content: "Fragment content here",
			wantParts: []string{
				"Rate the relevance",
				"Do NOT summarize",
				"Quick Note",
				"Fragment content here",
			},
			noParts: []string{
				"Summarize",
			},
		},
		{
			name:    "with profile",
			title:   "Note",
			content: "Content",
			profile: "User likes Go",
			wantParts: []string{
				"User likes Go",
				"interest profile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildScoreOnlyPrompt(tt.title, tt.content, tt.profile, tt.corrections)
			for _, part := range tt.wantParts {
				if !strings.Contains(prompt, part) {
					t.Errorf("prompt does not contain %q", part)
				}
			}
			for _, part := range tt.noParts {
				if strings.Contains(prompt, part) {
					t.Errorf("prompt should not contain %q", part)
				}
			}
		})
	}
}