package ai

import (
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestCheckAndRegeneratePreferences_BelowThreshold(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Only 2 corrections — below threshold of 20
	testutil.CreateEntryWithStars(t, app, resource.Id, "A1", "https://example.com/a1", 3, 5)
	testutil.CreateEntryWithStars(t, app, resource.Id, "A2", "https://example.com/a2", 4, 1)

	// Should not regenerate
	CheckAndRegeneratePreferences(app)

	// Verify no profile was created
	profiles, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(profiles) != 0 {
		t.Errorf("expected no profile (below threshold), got %d", len(profiles))
	}
}

func TestCheckAndRegeneratePreferences_ExceedsThreshold(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create enough corrections to exceed threshold
	for i := 0; i < 21; i++ {
		testutil.CreateEntryWithStars(t, app, resource.Id,
			"Article "+string(rune('A'+i)),
			"https://example.com/"+string(rune('a'+i)),
			3, 5)
	}

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "User prefers technical articles about programming.", nil
	})
	defer restore()

	CheckAndRegeneratePreferences(app)

	profiles, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].GetString("profile_text") != "User prefers technical articles about programming." {
		t.Errorf("profile_text = %q", profiles[0].GetString("profile_text"))
	}
}

func TestCountCorrectionsSinceLastProfile_WithProfile(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Create a profile with a past timestamp
	testutil.CreatePreference(t, app, "Old profile", "2024-01-01 00:00:00.000Z")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	// These corrections were created "now", which is after the profile
	testutil.CreateEntryWithStars(t, app, resource.Id, "A1", "https://example.com/cov-a1", 3, 5)
	testutil.CreateEntryWithStars(t, app, resource.Id, "A2", "https://example.com/cov-a2", 4, 1)

	count, err := countCorrectionsSinceLastProfile(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestSavePreferenceProfile_UpdatesExistingRecord(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// First save
	err := savePreferenceProfile(app, "First profile")
	if err != nil {
		t.Fatalf("first save error: %v", err)
	}

	// Second save should update
	err = savePreferenceProfile(app, "Updated profile")
	if err != nil {
		t.Fatalf("second save error: %v", err)
	}

	profiles, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile (updated), got %d", len(profiles))
	}
	if profiles[0].GetString("profile_text") != "Updated profile" {
		t.Errorf("profile_text = %q, want 'Updated profile'", profiles[0].GetString("profile_text"))
	}
}

func TestGeneratePreferenceProfile_Succeeds(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	testutil.CreateEntryWithStars(t, app, resource.Id, "Go Article", "https://example.com/cov-go", 2, 5)
	testutil.CreateEntryWithStars(t, app, resource.Id, "JS Article", "https://example.com/cov-js", 5, 1)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "User strongly prefers Go content over JavaScript.", nil
	})
	defer restore()

	err := GeneratePreferenceProfile(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].GetString("profile_text") != "User strongly prefers Go content over JavaScript." {
		t.Errorf("profile_text = %q", profiles[0].GetString("profile_text"))
	}
}

func TestBuildPreferencePrompt_ContainsCorrections(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	r1 := testutil.CreateEntryWithStars(t, app, resource.Id, "Go Cov Article", "https://example.com/cov-go2", 2, 5)

	records := []*core.Record{r1}
	prompt := buildPreferencePrompt(records)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "Go Cov Article") {
		t.Error("prompt should contain article titles")
	}
	if !strings.Contains(prompt, "preference profile") {
		t.Error("prompt should ask for preference profile")
	}
}

func TestTruncateText_Coverage(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"short stays", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"gets truncated", "hello world", 5, "hello..."},
		{"empty stays empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
