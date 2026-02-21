package ai

import (
	"fmt"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestCheckAndRegeneratePreferences_NoAPIKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// No API key, should handle gracefully
	CheckAndRegeneratePreferences(app)
}

func TestSavePreferenceProfile_NewProfile(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	err := savePreferenceProfile(app, "New profile text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles, _ := app.FindRecordsByFilter("preferences", "1=1", "", 0, 0, nil)
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].GetString("profile_text") != "New profile text" {
		t.Errorf("profile_text = %q", profiles[0].GetString("profile_text"))
	}
}

func TestCountCorrectionsSinceLastProfile_NoEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	count, err := countCorrectionsSinceLastProfile(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestGeneratePreferenceProfile_AIFailure(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	testutil.CreateEntryWithStars(t, app, resource.Id, "Article", "https://example.com/a", 2, 5)

	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		return "", fmt.Errorf("AI service unavailable")
	})
	defer restore()

	err := GeneratePreferenceProfile(app)
	if err == nil {
		t.Error("expected error when AI fails")
	}
}
