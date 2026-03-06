package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerHooks(app *pocketbase.PocketBase) {
	// On resource update, reset health on URL changes and clear fragment parsing
	// state when fragment settings change so the next fetch can rebuild entries.
	app.OnRecordUpdate("resources").BindFunc(func(e *core.RecordEvent) error {
		oldRecord := e.Record.Original()

		if oldRecord.GetString("url") != e.Record.GetString("url") {
			e.Record.Set("consecutive_failures", 0)
			e.Record.Set("status", "healthy")
		}

		if fragmentConfigChanged(oldRecord, e.Record) {
			e.Record.Set("fragment_hashes", "")
			deleteFragmentEntries(e.App, e.Record.Id)
		}

		return e.Next()
	})

	// On resource delete, cascade delete associated entries.
	app.OnRecordDelete("resources").BindFunc(func(e *core.RecordEvent) error {
		deleteAllResourceEntries(e.App, e.Record.Id)
		return e.Next()
	})
}

func fragmentConfigChanged(oldRecord, newRecord *core.Record) bool {
	return oldRecord.GetBool("fragment_feed") != newRecord.GetBool("fragment_feed") ||
		oldRecord.GetString("fragment_mode") != newRecord.GetString("fragment_mode") ||
		oldRecord.GetString("fragment_separator") != newRecord.GetString("fragment_separator")
}

func deleteFragmentEntries(app core.App, resourceID string) {
	deleteEntries(app, resourceID, "resource = {:id} && is_fragment = true")
}

func deleteAllResourceEntries(app core.App, resourceID string) {
	deleteEntries(app, resourceID, "resource = {:id}")
}

func deleteEntries(app core.App, resourceID, filter string) {
	entries, err := app.FindRecordsByFilter("entries", filter, "", 0, 0, map[string]any{"id": resourceID})
	if err != nil {
		log.Printf("Warning: could not find entries for resource %s: %v", resourceID, err)
		return
	}
	for _, entry := range entries {
		if err := app.Delete(entry); err != nil {
			log.Printf("Warning: could not delete entry %s: %v", entry.Id, err)
		}
	}
}
