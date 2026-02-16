package main

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func registerCollections(app core.App) {
	ensureResourcesCollection(app)
	ensureEntriesCollection(app)
	ensurePreferencesCollection(app)
	ensureSettingsCollection(app)
}

func ensureResourcesCollection(app core.App) {
	if _, err := app.FindCollectionByNameOrId("resources"); err == nil {
		return
	}

	collection := core.NewBaseCollection("resources")
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Max:      200,
	})
	collection.Fields.Add(&core.URLField{
		Name:     "url",
		Required: true,
	})
	collection.Fields.Add(&core.SelectField{
		Name:      "type",
		Required:  true,
		Values:    []string{"rss", "watchlist"},
		MaxSelect: 1,
	})
	collection.Fields.Add(&core.TextField{
		Name: "article_selector",
	})
	collection.Fields.Add(&core.TextField{
		Name: "content_selector",
	})
	collection.Fields.Add(&core.SelectField{
		Name:      "status",
		Required:  true,
		Values:    []string{"healthy", "failing", "quarantined"},
		MaxSelect: 1,
	})
	collection.Fields.Add(&core.NumberField{
		Name: "consecutive_failures",
	})
	collection.Fields.Add(&core.TextField{
		Name: "last_error",
	})
	collection.Fields.Add(&core.DateField{
		Name: "quarantined_at",
	})
	collection.Fields.Add(&core.BoolField{
		Name: "active",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "check_interval",
	})
	collection.Fields.Add(&core.DateField{
		Name: "last_checked",
	})

	// API rules - authenticated users only
	collection.ListRule = types.Pointer("@request.auth.id != ''")
	collection.ViewRule = types.Pointer("@request.auth.id != ''")
	collection.CreateRule = types.Pointer("@request.auth.id != ''")
	collection.UpdateRule = types.Pointer("@request.auth.id != ''")
	collection.DeleteRule = types.Pointer("@request.auth.id != ''")

	if err := app.Save(collection); err != nil {
		log.Printf("Failed to create resources collection: %v", err)
	}
}

func ensureEntriesCollection(app core.App) {
	if _, err := app.FindCollectionByNameOrId("entries"); err == nil {
		return
	}

	collection := core.NewBaseCollection("entries")
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

	collection.Fields.Add(&core.RelationField{
		Name:         "resource",
		CollectionId: getCollectionId(app, "resources"),
		Required:     true,
		MaxSelect:    1,
	})
	collection.Fields.Add(&core.URLField{
		Name:     "url",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name:     "title",
		Required: true,
		Max:      500,
	})
	collection.Fields.Add(&core.EditorField{
		Name: "raw_content",
	})
	collection.Fields.Add(&core.TextField{
		Name: "summary",
		Max:  2000,
	})
	collection.Fields.Add(&core.NumberField{
		Name: "ai_stars",
		Min:  floatPtr(0),
		Max:  floatPtr(5),
	})
	collection.Fields.Add(&core.NumberField{
		Name: "user_stars",
		Min:  floatPtr(0),
		Max:  floatPtr(5),
	})
	collection.Fields.Add(&core.BoolField{
		Name: "is_read",
	})
	collection.Fields.Add(&core.TextField{
		Name: "guid",
		Max:  1000,
	})
	collection.Fields.Add(&core.DateField{
		Name: "discovered_at",
	})
	collection.Fields.Add(&core.DateField{
		Name: "published_at",
	})
	collection.Fields.Add(&core.SelectField{
		Name:      "processing_status",
		Values:    []string{"pending", "done", "failed"},
		MaxSelect: 1,
	})

	collection.ListRule = types.Pointer("@request.auth.id != ''")
	collection.ViewRule = types.Pointer("@request.auth.id != ''")
	collection.CreateRule = types.Pointer("@request.auth.id != ''")
	collection.UpdateRule = types.Pointer("@request.auth.id != ''")
	collection.DeleteRule = types.Pointer("@request.auth.id != ''")

	if err := app.Save(collection); err != nil {
		log.Printf("Failed to create entries collection: %v", err)
	}
}

func ensurePreferencesCollection(app core.App) {
	if _, err := app.FindCollectionByNameOrId("preferences"); err == nil {
		return
	}

	collection := core.NewBaseCollection("preferences")
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

	collection.Fields.Add(&core.EditorField{
		Name: "profile_text",
	})
	collection.Fields.Add(&core.DateField{
		Name: "generated_at",
	})

	collection.ListRule = types.Pointer("@request.auth.id != ''")
	collection.ViewRule = types.Pointer("@request.auth.id != ''")
	collection.CreateRule = types.Pointer("@request.auth.id != ''")
	collection.UpdateRule = types.Pointer("@request.auth.id != ''")
	collection.DeleteRule = types.Pointer("@request.auth.id != ''")

	if err := app.Save(collection); err != nil {
		log.Printf("Failed to create preferences collection: %v", err)
	}
}

func ensureSettingsCollection(app core.App) {
	if _, err := app.FindCollectionByNameOrId("app_settings"); err == nil {
		return
	}

	collection := core.NewBaseCollection("app_settings")
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

	collection.Fields.Add(&core.TextField{
		Name:     "key",
		Required: true,
		Max:      200,
	})
	collection.Fields.Add(&core.TextField{
		Name: "value",
		Max:  2000,
	})

	collection.ListRule = types.Pointer("@request.auth.id != ''")
	collection.ViewRule = types.Pointer("@request.auth.id != ''")
	collection.CreateRule = types.Pointer("@request.auth.id != ''")
	collection.UpdateRule = types.Pointer("@request.auth.id != ''")
	collection.DeleteRule = types.Pointer("@request.auth.id != ''")

	if err := app.Save(collection); err != nil {
		log.Printf("Failed to create app_settings collection: %v", err)
	}
}

func getCollectionId(app core.App, name string) string {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return ""
	}
	return col.Id
}

func floatPtr(f float64) *float64 {
	return &f
}
