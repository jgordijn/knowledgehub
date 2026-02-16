package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestGeneratePreferenceProfile(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create corrections
	testutil.CreateEntryWithStars(t, app, resource.Id, "Go Article", "https://example.com/go", 2, 5)
	testutil.CreateEntryWithStars(t, app, resource.Id, "JS Article", "https://example.com/js", 5, 1)

	// Mock the AI client
	origComplete := clientCompleteFunc
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		return "User prefers Go over JavaScript programming.", nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	err := GeneratePreferenceProfile(app)
	if err != nil {
		t.Fatalf("GeneratePreferenceProfile returned error: %v", err)
	}

	// Check the profile was saved
	records, err := app.FindRecordsByFilter("preferences", "1=1", "-generated_at", 1, 0, nil)
	if err != nil || len(records) == 0 {
		t.Fatal("expected a preference record to be created")
	}

	profile := records[0].GetString("profile_text")
	if !strings.Contains(profile, "prefers Go") {
		t.Errorf("profile = %q, want to contain 'prefers Go'", profile)
	}
}

func TestGeneratePreferenceProfile_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	err := GeneratePreferenceProfile(app)
	if err == nil {
		t.Error("expected error when no API key")
	}
}

func TestGeneratePreferenceProfile_NoCorrections(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	err := GeneratePreferenceProfile(app)
	if err == nil {
		t.Error("expected error when no corrections exist")
	}
}

func TestGeneratePreferenceProfile_UpdatesExisting(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	testutil.CreateEntryWithStars(t, app, resource.Id, "Article A", "https://example.com/a", 2, 5)
	testutil.CreatePreference(t, app, "Old profile", "2024-01-01 12:00:00.000Z")

	origComplete := clientCompleteFunc
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		return "Updated preference profile.", nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	err := GeneratePreferenceProfile(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(records) != 1 {
		t.Errorf("expected exactly 1 preference record, got %d", len(records))
	}
	if got := records[0].GetString("profile_text"); got != "Updated preference profile." {
		t.Errorf("profile_text = %q, want updated", got)
	}
}

func TestCountCorrectionsSinceLastProfile(t *testing.T) {
	tests := []struct {
		name          string
		hasProfile    bool
		profileTime   string
		corrections   int
		expectedCount int
	}{
		{
			name:          "no profile, counts all corrections",
			hasProfile:    false,
			corrections:   5,
			expectedCount: 5,
		},
		{
			name:          "profile exists, counts corrections since",
			hasProfile:    true,
			profileTime:   time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
			corrections:   3,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

			if tt.hasProfile {
				testutil.CreatePreference(t, app, "Old profile", tt.profileTime)
			}

			for i := 0; i < tt.corrections; i++ {
				testutil.CreateEntryWithStars(t, app, resource.Id, "Article", "https://example.com/"+string(rune('a'+i)), 2, 5)
			}

			count, err := countCorrectionsSinceLastProfile(app)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.expectedCount {
				t.Errorf("count = %d, want %d", count, tt.expectedCount)
			}
		})
	}
}

func TestCheckAndRegeneratePreferences(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create fewer corrections than threshold
	for i := 0; i < 5; i++ {
		testutil.CreateEntryWithStars(t, app, resource.Id, "Article", "https://example.com/"+string(rune('a'+i)), 2, 5)
	}

	origComplete := clientCompleteFunc
	called := false
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		called = true
		return "Profile", nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	CheckAndRegeneratePreferences(app)

	if called {
		t.Error("should not regenerate when below threshold")
	}
}

func TestCheckAndRegeneratePreferences_AboveThreshold(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create more corrections than threshold
	for i := 0; i < regenerateThreshold+1; i++ {
		testutil.CreateEntryWithStars(t, app, resource.Id, "Article", "https://example.com/"+string(rune('a'+i)), 2, 5)
	}

	origComplete := clientCompleteFunc
	called := false
	clientCompleteFunc = func(apiKey, model string, messages []Message) (string, error) {
		called = true
		return "New profile", nil
	}
	defer func() { clientCompleteFunc = origComplete }()

	CheckAndRegeneratePreferences(app)

	if !called {
		t.Error("should regenerate when above threshold")
	}
}

func TestBuildPreferencePrompt(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	entry := testutil.CreateEntryWithStars(t, app, resource.Id, "Go Article", "https://example.com/go", 2, 5)

	records, _ := app.FindRecordsByFilter("entries", "id = {:id}", "", 1, 0, map[string]any{"id": entry.Id})
	prompt := buildPreferencePrompt(records)

	if !strings.Contains(prompt, "Go Article") {
		t.Error("prompt should contain article title")
	}
	if !strings.Contains(prompt, "preference profile") {
		t.Error("prompt should mention preference profile")
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world", 5, "hello..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}
