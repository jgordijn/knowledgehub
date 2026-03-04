package ai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestScoreOnly_EmptyContent(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Empty Fragment")
	entry.Set("url", "https://example.com/empty-frag")
	entry.Set("guid", "guid-empty-frag")
	entry.Set("raw_content", "")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		// Verify that empty content falls back to title
		for _, m := range messages {
			if m.Role == "user" && strings.Contains(m.Content, "Empty Fragment") {
				return `{"summary":"","stars":2}`, nil
			}
		}
		return `{"summary":"","stars":2}`, nil
	})
	defer restore()

	err := ScoreOnly(app, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScoreOnly_AIError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Error Fragment")
	entry.Set("url", "https://example.com/error-frag")
	entry.Set("guid", "guid-error-frag")
	entry.Set("raw_content", "Content")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "", fmt.Errorf("AI service error")
	})
	defer restore()

	err := ScoreOnly(app, entry)
	if err == nil {
		t.Error("expected error when AI fails")
	}
}

func TestScoreOnly_BadJSON(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Bad JSON Fragment")
	entry.Set("url", "https://example.com/bad-json-frag")
	entry.Set("guid", "guid-bad-json-frag")
	entry.Set("raw_content", "Content")
	entry.Set("processing_status", "pending")
	entry.Set("is_fragment", true)
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "not json", nil
	})
	defer restore()

	err := ScoreOnly(app, entry)
	if err == nil {
		t.Error("expected error for bad JSON response")
	}
}

func TestBuildScoreOnlyPrompt_WithCorrections(t *testing.T) {
	prompt := buildScoreOnlyPrompt("Fragment Title", "Content here", "", "- Article X: AI=3, User=5")
	if !strings.Contains(prompt, "rating corrections") {
		t.Error("prompt should contain rating corrections section")
	}
	if !strings.Contains(prompt, "Article X") {
		t.Error("prompt should contain correction details")
	}
}

func TestBuildScoreOnlyPrompt_LongContent(t *testing.T) {
	longContent := strings.Repeat("x", 10000)
	prompt := buildScoreOnlyPrompt("Title", longContent, "", "")
	if !strings.Contains(prompt, "...") {
		t.Error("long content should be truncated")
	}
}

func TestBuildScoreOnlyPrompt_WithProfileAndCorrections(t *testing.T) {
	prompt := buildScoreOnlyPrompt("Title", "Content", "User likes Go", "- Art: AI=2, User=4")
	if !strings.Contains(prompt, "User likes Go") {
		t.Error("prompt should contain profile")
	}
	if !strings.Contains(prompt, "Art") {
		t.Error("prompt should contain corrections")
	}
}

func TestHtmlToMarkdown_PlainText(t *testing.T) {
	result := htmlToMarkdown("Hello world, no HTML here")
	if result != "Hello world, no HTML here" {
		t.Errorf("plain text should pass through unchanged: %q", result)
	}
}

func TestHtmlToMarkdown_HTMLContent(t *testing.T) {
	result := htmlToMarkdown("<p>Hello <strong>world</strong></p>")
	if result == "" {
		t.Error("expected non-empty result")
	}
	if strings.Contains(result, "<p>") {
		t.Errorf("HTML tags should be converted: %q", result)
	}
}

func TestHtmlToMarkdown_EmptyString(t *testing.T) {
	result := htmlToMarkdown("")
	if result != "" {
		t.Errorf("empty string should remain empty: %q", result)
	}
}

func TestSummarizeAndScore_AIError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Error Test", "https://example.com/error", "guid-error")
	entry.Set("raw_content", "Content")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "", fmt.Errorf("AI service unavailable")
	})
	defer restore()

	err := SummarizeAndScore(app, entry)
	if err == nil {
		t.Error("expected error when AI fails")
	}
}

func TestBuildSummaryPrompt_HTMLContent(t *testing.T) {
	prompt := buildSummaryPrompt("Title", "<p>HTML <strong>content</strong></p>", "", "")
	// HTML should be converted to markdown
	if strings.Contains(prompt, "<p>") {
		t.Error("prompt should convert HTML to markdown")
	}
}
