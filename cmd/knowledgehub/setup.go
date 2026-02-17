package main

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func registerSetupRoutes(se *core.ServeEvent) {
	// GET /api/setup-status — returns whether initial setup is needed
	se.Router.GET("/api/setup-status", func(re *core.RequestEvent) error {
		needsSetup := !hasSuperusers(re.App)
		return re.JSON(http.StatusOK, map[string]bool{"needsSetup": needsSetup})
	})

	// POST /api/setup — creates the first superuser (only when none exist)
	se.Router.POST("/api/setup", func(re *core.RequestEvent) error {
		if hasSuperusers(re.App) {
			return re.JSON(http.StatusForbidden, map[string]string{
				"error": "Setup already completed. Use the login form.",
			})
		}

		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := re.BindBody(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body."})
		}

		if body.Email == "" || body.Password == "" {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Email and password are required."})
		}

		if len(body.Password) < 8 {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Password must be at least 8 characters."})
		}

		collection, err := re.App.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not find superusers collection."})
		}

		record := core.NewRecord(collection)
		record.SetEmail(body.Email)
		record.SetPassword(body.Password)

		if err := re.App.Save(record); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create account: " + err.Error()})
		}

		return re.JSON(http.StatusOK, map[string]string{"message": "Account created. You can now sign in."})
	})
}

func hasSuperusers(app core.App) bool {
	superusers, err := app.FindRecordsByFilter(
		core.CollectionNameSuperusers,
		"email != '__pbinstaller@example.com'",
		"",
		1, 0,
		nil,
	)
	return err == nil && len(superusers) > 0
}
