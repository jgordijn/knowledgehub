package ai

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComplete_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestComplete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error for empty choices")
	}
}

func TestComplete_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCompleteStream_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestCompleteStream_InvalidDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: not-json`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"valid"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	var chunks []string
	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "valid" {
		t.Errorf("chunks = %v, want [valid]", chunks)
	}
}

func TestCompleteStream_EmptyDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":""}}]}`)
		fmt.Fprintln(w, `data: {"choices":[]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	var chunks []string
	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected no chunks for empty content, got %v", chunks)
	}
}

func TestComplete_ConnectionError(t *testing.T) {
	client := NewClient("key", "model")
	client.BaseURL = "http://127.0.0.1:1"

	_, err := client.Complete([]Message{{Role: "user", Content: "test"}})
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestCompleteStream_ConnectionError(t *testing.T) {
	client := NewClient("key", "model")
	client.BaseURL = "http://127.0.0.1:1"

	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		return nil
	})
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestComplete_AuthorizationHeader(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	client := NewClient("my-secret-key", "model")
	client.BaseURL = server.URL

	client.Complete([]Message{{Role: "user", Content: "test"}})
	if authHeader != "Bearer my-secret-key" {
		t.Errorf("Authorization = %q, want 'Bearer my-secret-key'", authHeader)
	}
}

func TestCompleteStream_NonSSELines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Lines without "data: " prefix should be skipped
		fmt.Fprintln(w, ": comment")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"text"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("key", "model")
	client.BaseURL = server.URL

	var chunks []string
	err := client.CompleteStream([]Message{{Role: "user", Content: "test"}}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}
