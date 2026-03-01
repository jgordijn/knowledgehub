package ai

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComplete_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("definitely not json {{{"))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error for malformed JSON response")
	}
}

func TestCompleteStream_InvalidJSONChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Invalid JSON in data line — should be skipped
		fmt.Fprintln(w, `data: {broken json}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"valid"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	var chunks []string
	err := client.CompleteStream(
		[]Message{{Role: "user", Content: "test"}},
		func(chunk string) error {
			chunks = append(chunks, chunk)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "valid" {
		t.Errorf("expected [valid], got %v", chunks)
	}
}

func TestCompleteStream_EmptyChoicesSkipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":""}}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"real"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	var chunks []string
	err := client.CompleteStream(
		[]Message{{Role: "user", Content: "test"}},
		func(chunk string) error {
			chunks = append(chunks, chunk)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "real" {
		t.Errorf("expected [real], got %v", chunks)
	}
}

func TestSetCompleteFunc_RestoresOriginal(t *testing.T) {
	called := false
	restore := SetCompleteFunc(func(apiKey, model string, messages []Message) (string, error) {
		called = true
		return "mocked", nil
	})

	result, err := callComplete("key", "model", []Message{{Role: "user", Content: "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("custom function should have been called")
	}
	if result != "mocked" {
		t.Errorf("result = %q, want 'mocked'", result)
	}

	restore()
}

func TestComplete_ResponseBodyInError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid model parameter"}`))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "invalid model parameter") {
		t.Errorf("error should contain response body: %v", err)
	}
}
