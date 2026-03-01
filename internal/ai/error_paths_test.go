package ai

import (
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

// ============================================================
// preference.go:18 — CheckAndRegeneratePreferences countCorrections error
// ============================================================

func TestCheckAndRegeneratePreferences_DBError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Delete entries collection to make countCorrectionsSinceLastProfile fail
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding entries collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting entries collection: %v", err)
	}

	// Should not panic, should log and return
	CheckAndRegeneratePreferences(app)
}

// ============================================================
// preference.go:100 — savePreferenceProfile preferences collection missing
// ============================================================

func TestSavePreferenceProfile_CollectionMissing(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Delete preferences collection
	col, err := app.FindCollectionByNameOrId("preferences")
	if err != nil {
		t.Fatalf("finding preferences collection: %v", err)
	}
	if err := app.Delete(col); err != nil {
		t.Fatalf("deleting preferences collection: %v", err)
	}

	err = savePreferenceProfile(app, "test profile")
	if err == nil {
		t.Error("expected error when preferences collection is missing")
	}
}

// ============================================================
// preference.go:127 — countCorrectionsSinceLastProfile entries error
// ============================================================

func TestCountCorrectionsSinceLastProfile_EntriesError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// First create a preference record so we exercise the "has profile" path
	testutil.CreatePreference(t, app, "test profile", "2024-01-01T00:00:00Z")

	// Delete entries collection
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("finding entries collection: %v", err)
	}
	if err := app.Delete(entriesCol); err != nil {
		t.Fatalf("deleting entries collection: %v", err)
	}

	_, err = countCorrectionsSinceLastProfile(app)
	if err == nil {
		t.Error("expected error when entries collection is missing")
	}
}
