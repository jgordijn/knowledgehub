package main

import (
	"os"
	"strings"
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

func TestRegisterCollections_CreatesDailyNewsCollections(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	registerCollections(app)

	settings, err := app.FindCollectionByNameOrId("daily_news_settings")
	if err != nil {
		t.Fatalf("daily_news_settings collection not found: %v", err)
	}
	assertFieldExists(t, settings, "user")
	assertFieldExists(t, settings, "enabled")
	assertFieldExists(t, settings, "generation_time")
	assertFieldExists(t, settings, "timezone")
	assertFieldExists(t, settings, "extra_instructions")
	assertRule(t, "settings list", settings.ListRule, "user = @request.auth.id")
	assertRule(t, "settings view", settings.ViewRule, "user = @request.auth.id")
	assertDeniedRule(t, "settings create", settings.CreateRule)
	assertDeniedRule(t, "settings update", settings.UpdateRule)
	assertDeniedRule(t, "settings delete", settings.DeleteRule)
	assertIndexContains(t, settings, "unique", "user")

	digests, err := app.FindCollectionByNameOrId("daily_digests")
	if err != nil {
		t.Fatalf("daily_digests collection not found: %v", err)
	}
	for _, field := range []string{"user", "local_date", "period_start", "period_end", "status", "trigger", "title", "body_markdown", "referenced_entry_ids", "candidate_count", "included_count", "used_subset", "has_successful_snapshot", "last_success_at", "error_message", "queued_at", "started_at", "heartbeat_at", "attempt_finished_at", "window_key", "active_window_key", "scheduled_day_key", "active_scheduled_day_key", "successful_scheduled_day_key"} {
		assertFieldExists(t, digests, field)
	}
	assertRule(t, "digests list", digests.ListRule, "user = @request.auth.id")
	assertRule(t, "digests view", digests.ViewRule, "user = @request.auth.id")
	assertDeniedRule(t, "digests create", digests.CreateRule)
	assertDeniedRule(t, "digests update", digests.UpdateRule)
	assertDeniedRule(t, "digests delete", digests.DeleteRule)
	assertIndexContains(t, digests, "active_window_key", "where active_window_key != ''")
	assertIndexContains(t, digests, "active_scheduled_day_key", "where active_scheduled_day_key != ''")
	assertIndexContains(t, digests, "successful_scheduled_day_key", "where successful_scheduled_day_key != ''")
}

func assertFieldExists(t *testing.T, collection *core.Collection, name string) {
	t.Helper()
	if collection.Fields.GetByName(name) == nil {
		t.Fatalf("%s missing field %s", collection.Name, name)
	}
}

func assertRule(t *testing.T, label string, rule *string, want string) {
	t.Helper()
	if rule == nil || !strings.Contains(*rule, want) {
		t.Fatalf("%s rule = %v, want to contain %q", label, rule, want)
	}
}

func assertDeniedRule(t *testing.T, label string, rule *string) {
	t.Helper()
	if rule == nil || strings.TrimSpace(*rule) != "" {
		t.Fatalf("%s rule = %v, want denied empty rule", label, rule)
	}
}

func assertIndexContains(t *testing.T, collection *core.Collection, parts ...string) {
	t.Helper()
	for _, idx := range collection.Indexes {
		lower := strings.ToLower(idx)
		matched := true
		for _, part := range parts {
			if !strings.Contains(lower, strings.ToLower(part)) {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("%s indexes %v do not contain all parts %v", collection.Name, collection.Indexes, parts)
}

func TestEnsureDailyNewsDefaultSettingsForSuperusers(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()
	registerCollections(app)

	user1 := createTestSuperuser(t, app, "daily1@example.com")
	ensureDailyNewsDefaultSettings(app)
	settings := findDailyNewsSettingsForUser(t, app, user1.Id)
	if len(settings) != 1 {
		t.Fatalf("settings for user1 = %d, want 1", len(settings))
	}
	if !settings[0].GetBool("enabled") || settings[0].GetString("generation_time") != "08:00" || settings[0].GetString("timezone") != "Europe/Amsterdam" {
		t.Fatalf("unexpected defaults: enabled=%v time=%q timezone=%q", settings[0].GetBool("enabled"), settings[0].GetString("generation_time"), settings[0].GetString("timezone"))
	}

	settings[0].Set("generation_time", "09:30")
	if err := app.Save(settings[0]); err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}
	ensureDailyNewsDefaultSettings(app)
	settings = findDailyNewsSettingsForUser(t, app, user1.Id)
	if len(settings) != 1 || settings[0].GetString("generation_time") != "09:30" {
		t.Fatalf("default materialization was not idempotent; got %d records time %q", len(settings), settings[0].GetString("generation_time"))
	}

	user2 := createTestSuperuser(t, app, "daily2@example.com")
	ensureDailyNewsDefaultSettings(app)
	if got := len(findDailyNewsSettingsForUser(t, app, user2.Id)); got != 1 {
		t.Fatalf("settings for user2 after later pass = %d, want 1", got)
	}
}

func createTestSuperuser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("superusers collection not found: %v", err)
	}
	record := core.NewRecord(col)
	record.SetEmail(email)
	record.SetPassword("testpassword123456")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to create superuser: %v", err)
	}
	return record
}

func findDailyNewsSettingsForUser(t *testing.T, app core.App, userID string) []*core.Record {
	t.Helper()
	records, err := app.FindRecordsByFilter("daily_news_settings", "user = {:user}", "", 10, 0, map[string]any{"user": userID})
	if err != nil {
		t.Fatalf("failed to query daily news settings: %v", err)
	}
	return records
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
