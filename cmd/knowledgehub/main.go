package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/routes"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed all:ui/build
var uiFS embed.FS

func main() {
	dataDir := os.Getenv("KH_DATA_DIR")
	if dataDir == "" {
		dataDir = "./kh_data"
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	// Register collections on first run
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerCollections(se.App)

		// Register custom routes
		routes.RegisterChatRoute(se)
		routes.RegisterTriggerRoutes(se)
		registerSetupRoutes(se)

		// Serve embedded SvelteKit static files
		uiBuild, err := fs.Sub(uiFS, "ui/build")
		if err != nil {
			log.Printf("Warning: embedded UI not found: %v", err)
		} else {
			se.Router.GET("/{path...}", apis.Static(uiBuild, true))
		}

		return se.Next()
	})

	// Register hooks
	registerHooks(app)

	// Start the scheduler
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		scheduler := engine.NewScheduler(se.App)
		go scheduler.Start()
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
