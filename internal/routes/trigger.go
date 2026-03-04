package routes

import (
	"encoding/json"
	"net/http"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterTriggerRoutes adds endpoints to manually trigger resource fetching.
func RegisterTriggerRoutes(se *core.ServeEvent) {
	// POST /api/trigger/all — fetch all active resources
	se.Router.POST("/api/trigger/all", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}

		go engine.FetchAllResources(re.App)

		return re.JSON(http.StatusOK, map[string]string{"message": "Fetch started for all active resources."})
	})

	// POST /api/trigger/:id — fetch a single resource
	se.Router.POST("/api/trigger/{id}", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}

		id := re.Request.PathValue("id")
		resource, err := re.App.FindRecordById("resources", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "Resource not found."})
		}

		go engine.FetchSingleResource(re.App, resource)

		return re.JSON(http.StatusOK, map[string]string{
			"message": "Fetch started for " + resource.GetString("name") + ".",
		})
	})
}

// HandleTriggerAll is the testable core logic for the trigger-all endpoint.
func HandleTriggerAll(app core.App, w http.ResponseWriter) {
	go engine.FetchAllResources(app)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Fetch started for all active resources."})
}

// HandleTriggerSingle is the testable core logic for the trigger-single endpoint.
func HandleTriggerSingle(app core.App, w http.ResponseWriter, id string) {
	resource, err := app.FindRecordById("resources", id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Resource not found."})
		return
	}

	go engine.FetchSingleResource(app, resource)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Fetch started for " + resource.GetString("name") + ".",
	})
}
