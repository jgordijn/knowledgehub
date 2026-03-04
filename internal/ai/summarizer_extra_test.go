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

func TestParseSummaryResult_WithTakeaways(t *testing.T) {
	input := `{"summary":"A deep dive into CRDTs.","stars":4,"takeaways":["CRDTs enable conflict-free replication","Operation-based and state-based variants exist","Useful for collaborative editing"]}`
	result, err := parseSummaryResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "A deep dive into CRDTs." {
		t.Errorf("Summary = %q", result.Summary)
	}
	if result.Stars != 4 {
		t.Errorf("Stars = %d, want 4", result.Stars)
	}
	if len(result.Takeaways) != 3 {
		t.Fatalf("Takeaways length = %d, want 3", len(result.Takeaways))
	}
	if result.Takeaways[0] != "CRDTs enable conflict-free replication" {
		t.Errorf("Takeaways[0] = %q", result.Takeaways[0])
	}
}

func TestParseSummaryResult_WithoutTakeaways(t *testing.T) {
	input := `{"summary":"Short article.","stars":3}`
	result, err := parseSummaryResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "Short article." {
		t.Errorf("Summary = %q", result.Summary)
	}
	if result.Stars != 3 {
		t.Errorf("Stars = %d, want 3", result.Stars)
	}
	if len(result.Takeaways) != 0 {
		t.Errorf("Takeaways should be empty, got %v", result.Takeaways)
	}
}

func TestSummarizeAndScore_StoresTakeaways(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Long Article", "https://example.com/long", "guid-long")
	entry.Set("raw_content", strings.Repeat("Detailed content. ", 200))
	entry.Set("processing_status", "pending")
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return `{"summary":"A comprehensive overview.","stars":4,"takeaways":["Point one","Point two","Point three"]}`, nil
	})
	defer restore()

	err := SummarizeAndScore(app, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("summary"); got != "A comprehensive overview." {
		t.Errorf("summary = %q", got)
	}

	// PocketBase stores JSON fields; retrieve as raw and check
	raw := updated.Get("takeaways")
	if raw == nil {
		t.Fatal("takeaways should not be nil")
	}
}

func TestSummarizeAndScore_NullTakeaways(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Short Article", "https://example.com/short", "guid-short")
	entry.Set("raw_content", "A brief note.")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return `{"summary":"Brief note about Go.","stars":3}`, nil
	})
	defer restore()

	err := SummarizeAndScore(app, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("summary"); got != "Brief note about Go." {
		t.Errorf("summary = %q", got)
	}

	// Takeaways should be empty/null when not provided
	raw := updated.Get("takeaways")
	// PocketBase JSON fields return nil or empty for unset values
	if raw != nil {
		// Check it's an empty value (empty string, empty array, etc.)
		switch v := raw.(type) {
		case string:
			if v != "" && v != "null" && v != "[]" {
				t.Errorf("takeaways should be empty, got %q", v)
			}
		case []interface{}:
			if len(v) != 0 {
				t.Errorf("takeaways should be empty array, got %v", v)
			}
		}
	}
}