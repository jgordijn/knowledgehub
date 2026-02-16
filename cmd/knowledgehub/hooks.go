package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerHooks(app *pocketbase.PocketBase) {
	// On resource URL update, reset consecutive_failures to 0 and status to "healthy"
	app.OnRecordUpdate("resources").BindFunc(func(e *core.RecordEvent) error {
		oldURL := e.Record.Original().GetString("url")
		newURL := e.Record.GetString("url")
		if oldURL != newURL {
			e.Record.Set("consecutive_failures", 0)
			e.Record.Set("status", "healthy")
		}
		return e.Next()
	})

	// On resource delete, cascade delete associated entries
	app.OnRecordDelete("resources").BindFunc(func(e *core.RecordEvent) error {
		resourceID := e.Record.Id
		entries, err := e.App.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resourceID})
		if err != nil {
			log.Printf("Warning: could not find entries for resource %s: %v", resourceID, err)
			return e.Next()
		}
		for _, entry := range entries {
			if err := e.App.Delete(entry); err != nil {
				log.Printf("Warning: could not delete entry %s: %v", entry.Id, err)
			}
		}
		return e.Next()
	})
}
