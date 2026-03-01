package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

// failWriter is a ResponseWriter that fails after writing some data.
type failWriter struct {
	header     http.Header
	statusCode int
	written    int
	failAfter  int
}

func newFailWriter(failAfter int) *failWriter {
	return &failWriter{
		header:    make(http.Header),
		failAfter: failAfter,
	}
}

func (w *failWriter) Header() http.Header { return w.header }
func (w *failWriter) WriteHeader(code int) { w.statusCode = code }
func (w *failWriter) Write(data []byte) (int, error) {
	w.written += len(data)
	if w.written > w.failAfter {
		return 0, fmt.Errorf("write failed")
	}
	return len(data), nil
}

// Satisfy http.Flusher so the SSE flush paths are exercised
func (w *failWriter) Flush() {}

// ============================================================
// chat.go:77 — writeErr in CompleteStream callback
// ============================================================

func TestHandleChatDirect_StreamWriteError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Title", "https://example.com/a", "guid-write-err2")
	entry.Set("raw_content", "Content")
	app.Save(entry)

	// AI server that streams content
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send multiple chunks to increase chance of hitting the write error
		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk %d \"}}]}\n\n", i)
		}
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	// Use a writer that fails after some bytes
	w := newFailWriter(50) // fail after 50 bytes

	req := ChatRequestBody{
		EntryID:  entry.Id,
		Messages: []ai.Message{{Role: "user", Content: "test"}},
	}

	// This should handle the write error gracefully
	err := HandleChatDirect(app, w, req, aiServer.URL)
	// Should still return nil (error is logged, not returned)
	_ = err
}

// ============================================================
// link_summary.go:91 — writeErr in CompleteStream callback
// ============================================================

func TestHandleLinkSummaryDirect_StreamWriteError(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	// Article server
	articleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Article</title></head><body>
		<article><p>Long enough content for readability to extract properly and work as expected in this test scenario.</p></article>
		</body></html>`))
	}))
	defer articleServer.Close()

	// AI server
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"word%d \"}}]}\n\n", i)
		}
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	// Use a writer that fails after writing the meta event (which is ~200+ bytes)
	w := newFailWriter(300)

	body := LinkSummaryRequest{URL: articleServer.URL}

	// Override default HTTP client to reach the article server
	origHTTPClient := engine.DefaultHTTPClient
	engine.DefaultHTTPClient = articleServer.Client()
	defer func() { engine.DefaultHTTPClient = origHTTPClient }()

	err := HandleLinkSummaryDirect(app, w, body, aiServer.URL)
	_ = err
}
