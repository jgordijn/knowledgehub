package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestFetchSingleResource_Success(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Item</title><link>https://example.com/item</link><guid>g1</guid></item>
</channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	FetchSingleResource(app, resource)

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusHealthy {
		t.Errorf("status = %q, want %q", got, StatusHealthy)
	}
	if got := updated.GetInt("consecutive_failures"); got != 0 {
		t.Errorf("consecutive_failures = %d, want 0", got)
	}
}

func TestFetchSingleResource_Failure(t *testing.T) {
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

	FetchSingleResource(app, resource)

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusFailing {
		t.Errorf("status = %q, want %q", got, StatusFailing)
	}
	if got := updated.GetInt("consecutive_failures"); got != 1 {
		t.Errorf("consecutive_failures = %d, want 1", got)
	}
}

func TestFetchAllResources_MultipleResources(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	testutil.CreateResource(t, app, "r1", feedServer.URL, "rss", "healthy", 0, true)
	testutil.CreateResource(t, app, "r2", feedServer.URL, "rss", "healthy", 0, true)
	// Quarantined — should be skipped
	testutil.CreateResource(t, app, "quarantined", feedServer.URL, "rss", "quarantined", 5, true)
	// Inactive — should be skipped
	testutil.CreateResource(t, app, "inactive", feedServer.URL, "rss", "healthy", 0, false)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// Should process only the 2 active non-quarantined resources
	FetchAllResources(app)
}

func TestRetryFailedEntries_ProcessesEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Create entries with failed/pending statuses
	for _, status := range []string{"failed", "pending"} {
		entry := testutil.CreateEntry(t, app, resource.Id, "test-"+status, "https://example.com/"+status, "guid-retry-"+status)
		entry.Set("processing_status", status)
		app.Save(entry)
	}

	s := NewSchedulerWithInterval(app, 1*time.Hour)
	s.retryFailedEntries()

	time.Sleep(200 * time.Millisecond)
}

func TestFetchSingleResource_SuccessAfterPreviousFailure(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	// Start with 3 failures
	resource := testutil.CreateResource(t, app, "recovering", feedServer.URL, "rss", "failing", 3, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	FetchSingleResource(app, resource)

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusHealthy {
		t.Errorf("status = %q, want healthy", got)
	}
	if got := updated.GetInt("consecutive_failures"); got != 0 {
		t.Errorf("consecutive_failures = %d, want 0", got)
	}
}
