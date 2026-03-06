package main

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	_ "github.com/pocketbase/pocketbase/migrations"
)

func newTestApp(t *testing.T) (core.App, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "kh_collections_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	app := core.NewBaseApp(core.BaseAppConfig{DataDir: tempDir})
	if err := app.Bootstrap(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to bootstrap app: %v", err)
	}

	cleanup := func() {
		app.ResetBootstrapState()
		os.RemoveAll(tempDir)
	}

	return app, cleanup
}

func TestRegisterCollections_ExtendsSuperuserAuthTokenDuration(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	registerCollections(app)

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("failed to find superusers collection: %v", err)
	}

	if got := superusers.AuthToken.Duration; got != rememberMeAuthTokenDurationSeconds {
		t.Fatalf("superuser auth token duration = %d, want %d", got, rememberMeAuthTokenDurationSeconds)
	}
}

func TestEnsureSuperuserAuthTokenDuration_PreservesLongerDuration(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("failed to find superusers collection: %v", err)
	}

	const customDuration int64 = rememberMeAuthTokenDurationSeconds + 3600
	superusers.AuthToken.Duration = customDuration
	if err := app.Save(superusers); err != nil {
		t.Fatalf("failed to save superusers collection: %v", err)
	}

	ensureSuperuserAuthTokenDuration(app)

	superusers, err = app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("failed to reload superusers collection: %v", err)
	}

	if got := superusers.AuthToken.Duration; got != customDuration {
		t.Fatalf("superuser auth token duration = %d, want %d", got, customDuration)
	}
}
