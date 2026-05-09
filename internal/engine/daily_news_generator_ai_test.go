package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestGenerateDailyNewsDigestStructuredJSONAndReferences(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	resource := testutil.CreateResource(t, app, "Source", "https://example.com/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "A", "https://example.com/a", "a")

	var captured []ai.Message
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		captured = messages
		return `{"title":"Daily Brief","body_markdown":"# News [[kh-entry:` + entry.Id + `]]","referenced_entry_ids":["` + entry.Id + `","` + entry.Id + `","missing"]}`, nil
	})
	defer restore()

	result, err := GenerateDailyNewsDigest(app, DailyNewsGenerateInput{APIKey: "key", Model: "model", Candidates: []*core.Record{entry}})
	if err != nil {
		t.Fatalf("GenerateDailyNewsDigest error: %v", err)
	}
	if result.Title != "Daily Brief" || result.BodyMarkdown == "" || len(result.ReferencedEntryIDs) != 1 || result.ReferencedEntryIDs[0] != entry.Id {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(captured) != 2 || captured[0].Role != "system" || captured[1].Role != "user" || !strings.Contains(captured[0].Content, "structured JSON") {
		t.Fatalf("unexpected AI messages: %+v", captured)
	}
}

func TestGenerateDailyNewsDigestRejectsMalformedResponse(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	resource := testutil.CreateResource(t, app, "Source", "https://example.com/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "A", "https://example.com/a", "a")
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `not json`, nil
	})
	defer restore()

	if _, err := GenerateDailyNewsDigest(app, DailyNewsGenerateInput{APIKey: "key", Model: "model", Candidates: []*core.Record{entry}}); err == nil {
		t.Fatalf("expected malformed response error")
	}
}

func TestGenerateDailyNewsDigestEmptyWindow(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	result, err := GenerateDailyNewsDigest(app, DailyNewsGenerateInput{APIKey: "", Model: "", Candidates: nil})
	if err != nil {
		t.Fatalf("empty window should succeed: %v", err)
	}
	if result.Title != "No articles today" || !strings.Contains(result.BodyMarkdown, "No articles today") || result.CandidateCount != 0 || result.IncludedCount != 0 {
		t.Fatalf("unexpected empty digest: %+v", result)
	}
}

func TestRecordDailyNewsFailureSanitizesMessage(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-failure@example.com")
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "running", "automatic")

	if err := RecordDailyNewsFailure(app, digest.Id, errors.New("provider failed with sk-secret stack trace")); err != nil {
		t.Fatalf("RecordDailyNewsFailure: %v", err)
	}
	updated, _ := app.FindRecordById("daily_digests", digest.Id)
	if updated.GetString("status") != "failed" || strings.Contains(updated.GetString("error_message"), "sk-secret") {
		t.Fatalf("failure not sanitized: status=%s message=%q", updated.GetString("status"), updated.GetString("error_message"))
	}
}
