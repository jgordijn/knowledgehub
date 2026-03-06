package main

import (
	"os"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase"
)

func newHooksTestApp(t *testing.T) (*pocketbase.PocketBase, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "kh_hooks_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: tempDir})
	if err := app.Bootstrap(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to bootstrap app: %v", err)
	}

	registerCollections(app)
	registerHooks(app)

	cleanup := func() {
		app.ResetBootstrapState()
		os.RemoveAll(tempDir)
	}

	return app, cleanup
}

func TestRegisterHooks_ClearsFragmentStateOnFragmentConfigChange(t *testing.T) {
	app, cleanup := newHooksTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "Moments", "https://example.com/feed.xml", "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	resource.Set("fragment_mode", "auto")
	resource.Set("fragment_hashes", `{"guid-1":"hash-1"}`)
	if err := app.Save(resource); err != nil {
		t.Fatalf("failed to enable fragment feed: %v", err)
	}

	fragEntry := testutil.CreateEntry(t, app, resource.Id, "Old fragment", "https://example.com/a", "frag-1")
	fragEntry.Set("is_fragment", true)
	if err := app.Save(fragEntry); err != nil {
		t.Fatalf("failed to save fragment entry: %v", err)
	}

	normalEntry := testutil.CreateEntry(t, app, resource.Id, "Normal entry", "https://example.com/b", "entry-1")

	resource.Set("fragment_mode", "separated")
	resource.Set("fragment_separator", "~ ~ ~")
	if err := app.Save(resource); err != nil {
		t.Fatalf("failed to update fragment config: %v", err)
	}

	updated, err := app.FindRecordById("resources", resource.Id)
	if err != nil {
		t.Fatalf("failed to reload resource: %v", err)
	}
	if got := updated.GetString("fragment_hashes"); got != "" {
		t.Fatalf("fragment_hashes = %q, want empty", got)
	}

	if _, err := app.FindRecordById("entries", fragEntry.Id); err == nil {
		t.Fatal("expected fragment entry to be deleted")
	}
	if _, err := app.FindRecordById("entries", normalEntry.Id); err != nil {
		t.Fatalf("expected normal entry to be preserved: %v", err)
	}
}
