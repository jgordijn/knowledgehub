package testutil

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	_ "github.com/pocketbase/pocketbase/migrations"
)

// NewTestApp creates a fresh PocketBase app in a temp dir with all
// KnowledgeHub collections registered. Returns app and cleanup func.
func NewTestApp(t *testing.T) (core.App, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "kh_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir: tempDir,
	})

	if err := app.Bootstrap(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to bootstrap app: %v", err)
	}

	registerCollections(t, app)

	cleanup := func() {
		app.ResetBootstrapState()
		os.RemoveAll(tempDir)
	}

	return app, cleanup
}

func registerCollections(t *testing.T, app core.App) {
	t.Helper()

	// resources
	resources := core.NewBaseCollection("resources")
	addAutodateFields(resources)
	resources.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
	resources.Fields.Add(&core.URLField{Name: "url", Required: true})
	resources.Fields.Add(&core.SelectField{Name: "type", Required: true, Values: []string{"rss", "watchlist"}, MaxSelect: 1})
	resources.Fields.Add(&core.TextField{Name: "article_selector"})
	resources.Fields.Add(&core.TextField{Name: "content_selector"})
	resources.Fields.Add(&core.SelectField{Name: "status", Required: true, Values: []string{"healthy", "failing", "quarantined"}, MaxSelect: 1})
	resources.Fields.Add(&core.NumberField{Name: "consecutive_failures"})
	resources.Fields.Add(&core.TextField{Name: "last_error"})
	resources.Fields.Add(&core.DateField{Name: "quarantined_at"})
	resources.Fields.Add(&core.BoolField{Name: "active"})
	resources.Fields.Add(&core.BoolField{Name: "fragment_feed"})
	resources.Fields.Add(&core.BoolField{Name: "use_browser"})
	resources.Fields.Add(&core.NumberField{Name: "check_interval"})
	resources.Fields.Add(&core.DateField{Name: "last_checked"})
	resources.Fields.Add(&core.TextField{Name: "fragment_hashes"})
	resources.ListRule = types.Pointer("")
	resources.ViewRule = types.Pointer("")
	resources.CreateRule = types.Pointer("")
	resources.UpdateRule = types.Pointer("")
	resources.DeleteRule = types.Pointer("")
	if err := app.Save(resources); err != nil {
		t.Fatalf("failed to create resources collection: %v", err)
	}

	// entries
	entries := core.NewBaseCollection("entries")
	addAutodateFields(entries)
	entries.Fields.Add(&core.RelationField{Name: "resource", CollectionId: resources.Id, Required: true, MaxSelect: 1})
	entries.Fields.Add(&core.URLField{Name: "url", Required: true})
	entries.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 500})
	entries.Fields.Add(&core.EditorField{Name: "raw_content"})
	entries.Fields.Add(&core.TextField{Name: "summary", Max: 2000})
	fp := func(f float64) *float64 { return &f }
	entries.Fields.Add(&core.NumberField{Name: "ai_stars", Min: fp(0), Max: fp(5)})
	entries.Fields.Add(&core.NumberField{Name: "user_stars", Min: fp(0), Max: fp(5)})
	entries.Fields.Add(&core.BoolField{Name: "is_read"})
	entries.Fields.Add(&core.TextField{Name: "guid", Max: 1000})
	entries.Fields.Add(&core.DateField{Name: "discovered_at"})
	entries.Fields.Add(&core.DateField{Name: "published_at"})
	entries.Fields.Add(&core.SelectField{Name: "processing_status", Values: []string{"pending", "done", "failed"}, MaxSelect: 1})
	entries.Fields.Add(&core.BoolField{Name: "is_fragment"})
	entries.ListRule = types.Pointer("")
	entries.ViewRule = types.Pointer("")
	entries.CreateRule = types.Pointer("")
	entries.UpdateRule = types.Pointer("")
	entries.DeleteRule = types.Pointer("")
	if err := app.Save(entries); err != nil {
		t.Fatalf("failed to create entries collection: %v", err)
	}

	// preferences
	prefs := core.NewBaseCollection("preferences")
	addAutodateFields(prefs)
	prefs.Fields.Add(&core.EditorField{Name: "profile_text"})
	prefs.Fields.Add(&core.DateField{Name: "generated_at"})
	prefs.ListRule = types.Pointer("")
	prefs.ViewRule = types.Pointer("")
	prefs.CreateRule = types.Pointer("")
	prefs.UpdateRule = types.Pointer("")
	prefs.DeleteRule = types.Pointer("")
	if err := app.Save(prefs); err != nil {
		t.Fatalf("failed to create preferences collection: %v", err)
	}

	// app_settings
	settings := core.NewBaseCollection("app_settings")
	addAutodateFields(settings)
	settings.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 200})
	settings.Fields.Add(&core.TextField{Name: "value", Max: 2000})
	settings.ListRule = types.Pointer("")
	settings.ViewRule = types.Pointer("")
	settings.CreateRule = types.Pointer("")
	settings.UpdateRule = types.Pointer("")
	settings.DeleteRule = types.Pointer("")
	if err := app.Save(settings); err != nil {
		t.Fatalf("failed to create app_settings collection: %v", err)
	}
}

func addAutodateFields(col *core.Collection) {
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
}

// CreateResource is a test helper to create a resource record.
func CreateResource(t *testing.T, app core.App, name, url, rtype, status string, failures int, active bool) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("resources")
	if err != nil {
		t.Fatalf("resources collection not found: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("name", name)
	r.Set("url", url)
	r.Set("type", rtype)
	r.Set("status", status)
	r.Set("consecutive_failures", failures)
	r.Set("active", active)
	if err := app.Save(r); err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}
	return r
}

// CreateEntry is a test helper to create an entry record.
func CreateEntry(t *testing.T, app core.App, resourceID, title, url, guid string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection not found: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("resource", resourceID)
	r.Set("title", title)
	r.Set("url", url)
	r.Set("guid", guid)
	r.Set("processing_status", "done")
	r.Set("is_read", false)
	if err := app.Save(r); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}
	return r
}

// CreateSetting is a test helper to create an app_settings record.
func CreateSetting(t *testing.T, app core.App, key, value string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		t.Fatalf("app_settings collection not found: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("key", key)
	r.Set("value", value)
	if err := app.Save(r); err != nil {
		t.Fatalf("failed to create setting: %v", err)
	}
	return r
}

// CreatePreference is a test helper to create a preferences record.
func CreatePreference(t *testing.T, app core.App, profileText, generatedAt string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("preferences")
	if err != nil {
		t.Fatalf("preferences collection not found: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("profile_text", profileText)
	if generatedAt != "" {
		r.Set("generated_at", generatedAt)
	}
	if err := app.Save(r); err != nil {
		t.Fatalf("failed to create preference: %v", err)
	}
	return r
}

// CreateEntryWithStars creates an entry with AI and user star ratings.
func CreateEntryWithStars(t *testing.T, app core.App, resourceID, title, url string, aiStars, userStars int) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection not found: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("resource", resourceID)
	r.Set("title", title)
	r.Set("url", url)
	r.Set("guid", url)
	r.Set("processing_status", "done")
	r.Set("is_read", false)
	r.Set("ai_stars", aiStars)
	r.Set("user_stars", userStars)
	if err := app.Save(r); err != nil {
		t.Fatalf("failed to create entry with stars: %v", err)
	}
	return r
}
