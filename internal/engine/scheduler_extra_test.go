package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestSchedulerStartWithTick(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	// Feed server
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>Test</title></channel></rss>`))
	}))
	defer feedServer.Close()

	testutil.CreateResource(t, app, "test", feedServer.URL, "rss", "healthy", 0, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	// Use very short interval to test tick
	s := NewSchedulerWithInterval(app, 100*time.Millisecond)

	done := make(chan struct{})
	go func() {
		s.Start()
		close(done)
	}()

	// Wait for at least one tick
	time.Sleep(300 * time.Millisecond)
	s.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop")
	}
}

func TestSchedulerFetchAll_SuccessResetsStatus(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "failing", feedServer.URL, "rss", "failing", 3, true)

	origClient := DefaultHTTPClient
	DefaultHTTPClient = feedServer.Client()
	defer func() { DefaultHTTPClient = origClient }()

	s := NewScheduler(app)
	s.fetchAll()

	updated, _ := app.FindRecordById("resources", resource.Id)
	if got := updated.GetString("status"); got != StatusHealthy {
		t.Errorf("status = %q, want healthy after success", got)
	}
	if got := updated.GetInt("consecutive_failures"); got != 0 {
		t.Errorf("consecutive_failures = %d, want 0", got)
	}
}

func TestRetryFailedEntries_NoEntries(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	s := NewScheduler(app)
	// Should not panic
	s.retryFailedEntries()
}
