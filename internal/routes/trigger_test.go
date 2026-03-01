package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestHandleTriggerAll(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Use a local server so the background goroutine completes quickly
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = feedServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	HandleTriggerAll(app, recorder)

	// Wait for the background goroutine to finish
	time.Sleep(500 * time.Millisecond)

	if recorder.Code != 200 {
		t.Errorf("status = %d, want 200", recorder.Code)
	}

	var resp map[string]string
	json.NewDecoder(recorder.Body).Decode(&resp)
	if !strings.Contains(resp["message"], "Fetch started") {
		t.Errorf("message = %q", resp["message"])
	}
}

func TestHandleTriggerSingle_Found(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	// Use a local server so the background goroutine completes quickly
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>T</title></channel></rss>`))
	}))
	defer feedServer.Close()

	resource := testutil.CreateResource(t, app, "test-feed", feedServer.URL, "rss", "healthy", 0, true)

	origClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = feedServer.Client()
	defer func() { engine.DefaultHTTPClient = origClient }()

	recorder := httptest.NewRecorder()
	HandleTriggerSingle(app, recorder, resource.Id)

	// Wait for the background goroutine to finish
	time.Sleep(500 * time.Millisecond)

	if recorder.Code != 200 {
		t.Errorf("status = %d, want 200", recorder.Code)
	}

	var resp map[string]string
	json.NewDecoder(recorder.Body).Decode(&resp)
	if !strings.Contains(resp["message"], "test-feed") {
		t.Errorf("message should contain resource name: %q", resp["message"])
	}
}

func TestHandleTriggerSingle_NotFound(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	recorder := httptest.NewRecorder()
	HandleTriggerSingle(app, recorder, "nonexistent-id")

	if recorder.Code != 404 {
		t.Errorf("status = %d, want 404", recorder.Code)
	}

	var resp map[string]string
	json.NewDecoder(recorder.Body).Decode(&resp)
	if resp["error"] != "Resource not found." {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestRegisterTriggerRoutes_Type(t *testing.T) {
	var _ func(*core.ServeEvent) = RegisterTriggerRoutes
}
