package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestParseSummaryResult(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantStars int
		wantSumm  string
	}{
		{
			name:      "valid JSON",
			input:     `{"summary":"Great article about Go.","stars":4}`,
			wantStars: 4,
			wantSumm:  "Great article about Go.",
		},
		{
			name:      "JSON in code block",
			input:     "```json\n{\"summary\":\"Code block\",\"stars\":3}\n```",
			wantStars: 3,
			wantSumm:  "Code block",
		},
		{
			name:      "JSON in plain code block",
			input:     "```\n{\"summary\":\"Plain block\",\"stars\":2}\n```",
			wantStars: 2,
			wantSumm:  "Plain block",
		},
		{
			name:      "stars clamped to minimum 1",
			input:     `{"summary":"Low","stars":0}`,
			wantStars: 1,
			wantSumm:  "Low",
		},
		{
			name:      "stars clamped to maximum 5",
			input:     `{"summary":"High","stars":10}`,
			wantStars: 5,
			wantSumm:  "High",
		},
		{
			name:    "invalid JSON",
			input:   "not json at all",
			wantErr: true,
		},
		{
			name:    "empty response",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSummaryResult(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Stars != tt.wantStars {
				t.Errorf("Stars = %d, want %d", result.Stars, tt.wantStars)
			}
			if result.Summary != tt.wantSumm {
				t.Errorf("Summary = %q, want %q", result.Summary, tt.wantSumm)
			}
		})
	}
}

func TestBuildSummaryPrompt(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		content     string
		profile     string
		corrections string
		wantParts   []string
	}{
		{
			name:    "basic prompt without profile",
			title:   "Test Article",
			content: "Article content here",
			wantParts: []string{
				"Summarize",
				"Test Article",
				"Article content here",
				"JSON",
			},
		},
		{
			name:    "prompt with profile",
			title:   "Test",
			content: "Content",
			profile: "User likes Go programming",
			wantParts: []string{
				"User likes Go programming",
				"interest profile",
			},
		},
		{
			name:        "prompt with corrections",
			title:       "Test",
			content:     "Content",
			corrections: "- Article X: AI=3, User=5",
			wantParts: []string{
				"rating corrections",
				"Article X",
			},
		},
		{
			name:    "long content is truncated",
			title:   "Test",
			content: strings.Repeat("x", 10000),
			wantParts: []string{
				"...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildSummaryPrompt(tt.title, tt.content, tt.profile, tt.corrections)
			for _, part := range tt.wantParts {
				if !strings.Contains(prompt, part) {
					t.Errorf("prompt does not contain %q", part)
				}
			}
		})
	}
}

func TestSummarizeAndScore(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Mock OpenRouter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: `{"summary":"This article discusses Go programming.","stars":4}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Go Programming Guide")
	entry.Set("url", "https://example.com/go-guide")
	entry.Set("guid", "go-guide")
	entry.Set("raw_content", "This is a comprehensive guide to Go programming.")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	// Patch the client to use our mock server
	
	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		c := NewClient(apiKey, model)
		c.BaseURL = server.URL
		return c.Complete(messages)
	})
	defer restore()

	err := SummarizeAndScore(app, entry)
	if err != nil {
		t.Fatalf("SummarizeAndScore returned error: %v", err)
	}

	// Reload entry
	updated, _ := app.FindRecordById("entries", entry.Id)
	if got := updated.GetString("summary"); got != "This article discusses Go programming." {
		t.Errorf("summary = %q, want expected summary", got)
	}
	if got := updated.GetInt("ai_stars"); got != 4 {
		t.Errorf("ai_stars = %d, want 4", got)
	}
	if got := updated.GetString("processing_status"); got != "done" {
		t.Errorf("processing_status = %q, want done", got)
	}
}

func TestSummarizeAndScore_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	col, _ := app.FindCollectionByNameOrId("entries")
	entry := core.NewRecord(col)
	entry.Set("resource", resource.Id)
	entry.Set("title", "Test")
	entry.Set("url", "https://example.com/test")
	entry.Set("guid", "test")
	entry.Set("processing_status", "pending")
	app.Save(entry)

	err := SummarizeAndScore(app, entry)
	if err == nil {
		t.Error("expected error when no API key, got nil")
	}
}

func TestLoadPreferenceProfile(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// No profile exists
	profile := loadPreferenceProfile(app)
	if profile != "" {
		t.Errorf("expected empty profile, got %q", profile)
	}

	// Create a profile
	testutil.CreatePreference(t, app, "User likes Go", "2024-01-01 12:00:00.000Z")

	profile = loadPreferenceProfile(app)
	if profile != "User likes Go" {
		t.Errorf("profile = %q, want %q", profile, "User likes Go")
	}
}

func TestLoadRecentCorrections(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// No corrections
	corrections := loadRecentCorrections(app)
	if corrections != "" {
		t.Errorf("expected empty corrections, got %q", corrections)
	}

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create entries with different AI vs user stars
	testutil.CreateEntryWithStars(t, app, resource.Id, "Article A", "https://example.com/a", 3, 5)
	testutil.CreateEntryWithStars(t, app, resource.Id, "Article B", "https://example.com/b", 4, 2)

	corrections = loadRecentCorrections(app)
	if corrections == "" {
		t.Error("expected corrections, got empty")
	}
	if !strings.Contains(corrections, "Article A") {
		t.Error("corrections should mention Article A")
	}
}
