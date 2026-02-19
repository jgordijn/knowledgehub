package routes

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestBuildChatSystemPrompt(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		want    []string
	}{
		{
			name:    "includes title and content",
			title:   "Go Programming",
			content: "Go is a statically typed language.",
			want:    []string{"Go Programming", "Go is a statically typed language", "ONLY based on the article"},
		},
		{
			name:    "truncates long content",
			title:   "Long Article",
			content: strings.Repeat("x", 10000),
			want:    []string{"..."},
		},
		{
			name:    "empty content",
			title:   "Empty",
			content: "",
			want:    []string{"ONLY based on the article", "Empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildChatSystemPrompt(tt.title, tt.content, "")
			for _, w := range tt.want {
				if !strings.Contains(prompt, w) {
					t.Errorf("prompt does not contain %q", w)
				}
			}
		})
	}
}

func TestValidateChatRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    ChatRequestBody
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid request",
			body:    ChatRequestBody{EntryID: "abc123", Messages: []ai.Message{{Role: "user", Content: "hello"}}},
			wantErr: false,
		},
		{
			name:    "missing entry_id",
			body:    ChatRequestBody{EntryID: "", Messages: []ai.Message{{Role: "user", Content: "hello"}}},
			wantErr: true,
			errMsg:  "entry_id is required",
		},
		{
			name:    "empty messages",
			body:    ChatRequestBody{EntryID: "abc", Messages: nil},
			wantErr: true,
			errMsg:  "messages are required",
		},
		{
			name:    "empty messages slice",
			body:    ChatRequestBody{EntryID: "abc", Messages: []ai.Message{}},
			wantErr: true,
			errMsg:  "messages are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatRequest(tt.body)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildChatMessages(t *testing.T) {
	userMsgs := []ai.Message{
		{Role: "user", Content: "What is Go?"},
		{Role: "assistant", Content: "Go is a programming language."},
		{Role: "user", Content: "Tell me more."},
	}

	messages := BuildChatMessages("Test Title", "Article content", userMsgs)

	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	// First should be system prompt
	if messages[0].Role != "system" {
		t.Errorf("first message role = %q, want system", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "ONLY based on the article") {
		t.Error("system prompt should contain article constraint")
	}
	if !strings.Contains(messages[0].Content, "Test Title") {
		t.Error("system prompt should contain title")
	}
	if !strings.Contains(messages[0].Content, "Article content") {
		t.Error("system prompt should contain content")
	}

	// Rest should be user messages in order
	if messages[1].Role != "user" || messages[1].Content != "What is Go?" {
		t.Errorf("unexpected message[1]: %+v", messages[1])
	}
	if messages[2].Role != "assistant" {
		t.Errorf("message[2] role = %q, want assistant", messages[2].Role)
	}
	if messages[3].Role != "user" || messages[3].Content != "Tell me more." {
		t.Errorf("unexpected message[3]: %+v", messages[3])
	}
}

func TestChatRequestBody_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantID  string
		wantLen int
	}{
		{
			name:    "valid request",
			json:    `{"entry_id":"abc123","messages":[{"role":"user","content":"hello"}]}`,
			wantID:  "abc123",
			wantLen: 1,
		},
		{
			name:    "multiple messages",
			json:    `{"entry_id":"abc123","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"},{"role":"user","content":"tell me more"}]}`,
			wantID:  "abc123",
			wantLen: 3,
		},
		{
			name:    "invalid json",
			json:    "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body ChatRequestBody
			err := json.NewDecoder(strings.NewReader(tt.json)).Decode(&body)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if body.EntryID != tt.wantID {
				t.Errorf("EntryID = %q, want %q", body.EntryID, tt.wantID)
			}
			if len(body.Messages) != tt.wantLen {
				t.Errorf("messages len = %d, want %d", len(body.Messages), tt.wantLen)
			}
		})
	}
}

func TestChatStreamingIntegration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Test Article", "https://example.com/article", "guid-1")
	entry.Set("raw_content", "This article is about Go programming.")
	app.Save(entry)

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"The article "}}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"discusses Go."}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	client := ai.NewClient("test-key", "test-model")
	client.BaseURL = aiServer.URL

	messages := BuildChatMessages("Test Article", "This article is about Go programming.", []ai.Message{
		{Role: "user", Content: "What is this about?"},
	})

	var buf bytes.Buffer
	w := httptest.NewRecorder()
	w.Body = &buf

	err := client.CompleteStream(messages, func(chunk string) error {
		data, _ := json.Marshal(map[string]string{"content": chunk})
		fmt.Fprintf(w, "data: %s\n\n", data)
		return nil
	})

	if err != nil {
		t.Fatalf("streaming error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "The article") {
		t.Errorf("output should contain streamed content, got: %s", output)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	eventCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			eventCount++
			data := strings.TrimPrefix(line, "data: ")
			var parsed map[string]string
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				t.Errorf("invalid SSE data JSON: %s", data)
			}
		}
	}

	if eventCount < 2 {
		t.Errorf("expected at least 2 SSE events, got %d", eventCount)
	}
}

func TestHandleChatEndToEnd(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Answer from AI"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer aiServer.Close()

	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	testutil.CreateSetting(t, app, "openrouter_model", "test-model")
	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Article Title", "https://example.com/a", "g1")
	entry.Set("raw_content", "Article content about testing.")
	app.Save(entry)

	apiKey, _ := ai.GetAPIKey(app)
	model := ai.GetModel(app)

	loadedEntry, err := app.FindRecordById("entries", entry.Id)
	if err != nil {
		t.Fatalf("entry not found: %v", err)
	}

	rawContent := loadedEntry.GetString("raw_content")
	title := loadedEntry.GetString("title")
	messages := BuildChatMessages(title, rawContent, []ai.Message{
		{Role: "user", Content: "What is this about?"},
	})

	client := ai.NewClient(apiKey, model)
	client.BaseURL = aiServer.URL

	var chunks []string
	err = client.CompleteStream(messages, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})

	if err != nil {
		t.Fatalf("streaming error: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	if len(chunks) > 0 && chunks[0] != "Answer from AI" {
		t.Errorf("chunk = %q, want %q", chunks[0], "Answer from AI")
	}
}

func TestRegisterChatRoute(t *testing.T) {
	var _ func(*core.ServeEvent) = RegisterChatRoute
}
