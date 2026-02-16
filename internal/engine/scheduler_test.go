package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestNewScheduler(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	s := NewScheduler(app)
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.interval != defaultInterval {
		t.Errorf("interval = %v, want %v", s.interval, defaultInterval)
	}
}

func TestNewSchedulerWithInterval(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	interval := 5 * time.Minute
	s := NewSchedulerWithInterval(app, interval)
	if s.interval != interval {
		t.Errorf("interval = %v, want %v", s.interval, interval)
	}
}

func TestSchedulerStartStop(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	s := NewSchedulerWithInterval(app, 1*time.Hour) // long interval so ticker doesn't fire

	done := make(chan struct{})
	go func() {
		s.Start()
		close(done)
	}()

	// Give the initial fetchAll a moment to run
	time.Sleep(200 * time.Millisecond)

	s.Stop()

	select {
	case <-done:
		// OK - scheduler stopped
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop within timeout")
	}
}

func TestSchedulerFetchAll_NoResources(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	s := NewScheduler(app)
	// Should not panic with no resources
	s.fetchAll()
}

func TestSchedulerFetchAll_SkipsQuarantined(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateResource(t, app, "quarantined", "https://example.com/feed", "rss", "quarantined", 5, true)

	s := NewScheduler(app)
	// The fetch filter excludes quarantined resources, so this should not attempt to fetch
	s.fetchAll()
}

func TestSchedulerFetchAll_SkipsInactive(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateResource(t, app, "inactive", "https://example.com/feed", "rss", "healthy", 0, false)

	s := NewScheduler(app)
	s.fetchAll()
}

func TestSchedulerRetryFailedEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", "healthy", 0, true)

	// Create entries in failed/pending state
	col, _ := app.FindCollectionByNameOrId("entries")

	for _, status := range []string{"failed", "pending"} {
		r := core.NewRecord(col)
		r.Set("resource", resource.Id)
		r.Set("title", "test-"+status)
		r.Set("url", "https://example.com/"+status)
		r.Set("guid", "guid-"+status)
		r.Set("processing_status", status)
		app.Save(r)
	}

	s := NewScheduler(app)
	// Should not panic
	s.retryFailedEntries()

	time.Sleep(100 * time.Millisecond)
}

func TestSchedulerFetchAll_WithActiveResource(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Test Item</title><link>https://example.com/item</link><guid>g1</guid></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "active", feedServer.URL, "rss", "healthy", 0, true)

	// Override the DefaultHTTPClient for this test
	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	s := NewScheduler(app)
	s.fetchAll()

	time.Sleep(200 * time.Millisecond)

	// Should have fetched and created entries
	entries, _ := app.FindRecordsByFilter("entries", "resource = {:id}", "", 0, 0, map[string]any{"id": resource.Id})
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Resource should be marked as successful
	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusHealthy {
		t.Errorf("status = %q, want %q", got, StatusHealthy)
	}
}

func TestSchedulerFetchAll_RecordsFailure(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resource := testutil.CreateResource(t, app, "broken", server.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = server.Client()
	defer func() { DefaultHTTPClient = origClient }()

	s := NewScheduler(app)
	s.fetchAll()

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusFailing {
		t.Errorf("status = %q, want %q", got, StatusFailing)
	}
	if got := updated.GetInt("consecutive_failures"); got != 1 {
		t.Errorf("consecutive_failures = %d, want 1", got)
	}
}