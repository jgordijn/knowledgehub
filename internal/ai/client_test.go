package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComplete(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		wantContains string
	}{
		{
			name: "successful completion",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
					t.Errorf("unexpected auth header: %s", auth)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("unexpected content-type: %s", ct)
				}

				body, _ := io.ReadAll(r.Body)
				var req ChatRequest
				json.Unmarshal(body, &req)
				if req.Stream {
					t.Error("expected stream=false")
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"choices":[{"message":{"content":"Hello from AI"}}]}`))
			},
			wantContains: "Hello from AI",
		},
		{
			name: "API error response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limited"}`))
			},
			wantErr: true,
		},
		{
			name: "empty choices",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"choices":[]}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("test-key", "test-model")
			client.BaseURL = server.URL

			result, err := client.Complete([]Message{
				{Role: "user", Content: "test prompt"},
			})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("result %q does not contain %q", result, tt.wantContains)
			}
		})
	}
}

func TestCompleteStream(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		wantChunks   int
		wantContains string
	}{
		{
			name: "successful streaming",
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var req ChatRequest
				json.Unmarshal(body, &req)
				if !req.Stream {
					t.Error("expected stream=true")
				}

				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Hello "}}]}`)
				fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"World"}}]}`)
				fmt.Fprintln(w, "data: [DONE]")
			},
			wantChunks:   2,
			wantContains: "Hello World",
		},
		{
			name: "API error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
			},
			wantErr: true,
		},
		{
			name: "empty stream",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprintln(w, "data: [DONE]")
			},
			wantChunks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("test-key", "test-model")
			client.BaseURL = server.URL

			var chunks []string
			err := client.CompleteStream(
				[]Message{{Role: "user", Content: "test"}},
				func(chunk string) error {
					chunks = append(chunks, chunk)
					return nil
				},
			)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(chunks) != tt.wantChunks {
				t.Errorf("got %d chunks, want %d", len(chunks), tt.wantChunks)
			}

			if tt.wantContains != "" {
				combined := strings.Join(chunks, "")
				if !strings.Contains(combined, tt.wantContains) {
					t.Errorf("combined chunks %q does not contain %q", combined, tt.wantContains)
				}
			}
		})
	}
}

func TestCompleteStream_CallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"chunk"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	client := NewClient("test-key", "test-model")
	client.BaseURL = server.URL

	err := client.CompleteStream(
		[]Message{{Role: "user", Content: "test"}},
		func(chunk string) error {
			return fmt.Errorf("callback error")
		},
	)

	if err == nil || !strings.Contains(err.Error(), "callback error") {
		t.Errorf("expected callback error, got: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("my-key", "my-model")
	if c.APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "my-key")
	}
	if c.Model != "my-model" {
		t.Errorf("Model = %q, want %q", c.Model, "my-model")
	}
	if c.BaseURL != openRouterURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, openRouterURL)
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}
