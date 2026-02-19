package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/pocketbase/pocketbase/core"
)

// ChatRequestBody is the expected JSON body for the chat endpoint.
type ChatRequestBody struct {
	EntryID      string       `json:"entry_id"`
	Messages     []ai.Message `json:"messages"`
	ExtraContext string       `json:"extra_context,omitempty"`
}

// RegisterChatRoute adds the POST /api/chat streaming endpoint.
func RegisterChatRoute(se *core.ServeEvent) {
	se.Router.POST("/api/chat", func(re *core.RequestEvent) error {
		return handleChat(re)
	})
}

func handleChat(re *core.RequestEvent) error {
	var body ChatRequestBody
	if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := ValidateChatRequest(body); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	app := re.App
	w := re.Response

	return HandleChatDirect(app, w, body, "")
}

// HandleChatDirect is the testable core logic of the chat endpoint.
// If baseURL is empty, it uses the default OpenRouter URL.
func HandleChatDirect(app core.App, w http.ResponseWriter, body ChatRequestBody, baseURL string) error {
	if err := ValidateChatRequest(body); err != nil {
		return err
	}

	entry, err := app.FindRecordById("entries", body.EntryID)
	if err != nil {
		return fmt.Errorf("entry not found: %w", err)
	}

	rawContent := entry.GetString("raw_content")
	title := entry.GetString("title")

	apiKey, err := ai.GetAPIKey(app)
	if err != nil {
		return fmt.Errorf("API key not configured: %w", err)
	}
	model := ai.GetModel(app)

	messages := BuildChatMessages(title, rawContent, body.Messages, body.ExtraContext)
	client := ai.NewClient(apiKey, model)
	if baseURL != "" {
		client.BaseURL = baseURL
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	err = client.CompleteStream(messages, func(chunk string) error {
		data, _ := json.Marshal(map[string]string{"content": chunk})
		_, writeErr := fmt.Fprintf(w, "data: %s\n\n", data)
		if writeErr != nil {
			return writeErr
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return nil
	})

	if err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errData)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// NewChatHandler returns an http.Handler for testing the chat endpoint outside PocketBase.
func NewChatHandler(app core.App, baseURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body ChatRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		if err := HandleChatDirect(app, w, body, baseURL); err != nil {
			if w.Header().Get("Content-Type") == "" {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})
}

// ValidateChatRequest checks required fields in the request body.
func ValidateChatRequest(body ChatRequestBody) error {
	if body.EntryID == "" {
		return fmt.Errorf("entry_id is required")
	}
	if len(body.Messages) == 0 {
		return fmt.Errorf("messages are required")
	}
	return nil
}

// BuildChatMessages constructs the message list for the AI, including the system prompt.
func BuildChatMessages(title, rawContent string, userMessages []ai.Message, extraContext ...string) []ai.Message {
	extra := ""
	if len(extraContext) > 0 {
		extra = extraContext[0]
	}
	systemPrompt := buildChatSystemPrompt(title, rawContent, extra)
	messages := make([]ai.Message, 0, len(userMessages)+1)
	messages = append(messages, ai.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	messages = append(messages, userMessages...)
	return messages
}

func buildChatSystemPrompt(title, content, extraContext string) string {
	if len(content) > 8000 {
		content = content[:8000] + "..."
	}
	if extraContext != "" {
		if len(extraContext) > 8000 {
			extraContext = extraContext[:8000] + "..."
		}
		return fmt.Sprintf(
			"Answer ONLY based on the articles below. If the answer is not in the articles, say so.\n\nMain Article: %s\n\n%s\n\nLinked Article:\n%s",
			title, content, extraContext,
		)
	}
	return fmt.Sprintf(
		"Answer ONLY based on the article below. If the answer is not in the article, say so.\n\nArticle: %s\n\n%s",
		title, content,
	)
}
