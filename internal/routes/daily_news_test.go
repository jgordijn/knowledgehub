package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestHandleDailyNewsGenerateNowQueuesPendingJob(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")

	status, dto, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T07:30:00Z"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", status)
	}
	if dto.ID == "" || dto.Status != "pending" || dto.User != user.Id || dto.Trigger != "automatic" {
		t.Fatalf("unexpected dto: %+v", dto)
	}

	record, err := app.FindRecordById("daily_digests", dto.ID)
	if err != nil {
		t.Fatalf("pending job was not persisted: %v", err)
	}
	if record.GetString("status") != "pending" || record.GetString("user") != user.Id {
		t.Fatalf("unexpected persisted job status/user")
	}
}

func TestHandleDailyNewsGenerateNowReusesExistingActiveJob(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "active@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	now := mustTime("2026-05-08T07:30:00Z")
	_, first, err := HandleDailyNewsGenerateNow(app, user.Id, now)
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}

	status, second, err := HandleDailyNewsGenerateNow(app, user.Id, now.Add(500*time.Millisecond))
	if err != nil {
		t.Fatalf("second generate failed: %v", err)
	}
	if status != http.StatusAccepted || second.ID != first.ID {
		t.Fatalf("expected active job reuse, status=%d first=%s second=%s", status, first.ID, second.ID)
	}
}

func TestHandleDailyNewsGenerateNowReturnsSuccessfulSameDayDigest(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "success@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	status, dto, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T07:30:00Z"))
	if err != nil || status != http.StatusAccepted {
		t.Fatalf("queue failed: status=%d err=%v", status, err)
	}
	if err := engine.CompleteDailyNewsJob(app, dto.ID, "success", "", mustTime("2026-05-08T07:45:00Z")); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	status, again, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T08:00:00Z"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK || again.ID != dto.ID || again.Status != "success" {
		t.Fatalf("expected existing success, status=%d dto=%+v", status, again)
	}
}

func TestHandleDailyNewsGenerateNowRetriesAfterFailedDigest(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "retry@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	now := mustTime("2026-05-08T07:30:00Z")
	_, failed, err := HandleDailyNewsGenerateNow(app, user.Id, now)
	if err != nil {
		t.Fatalf("queue failed: %v", err)
	}
	if err := engine.CompleteDailyNewsJob(app, failed.ID, "failed", "boom", mustTime("2026-05-08T07:45:00Z")); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	status, retry, err := HandleDailyNewsGenerateNow(app, user.Id, now)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if status != http.StatusAccepted || retry.ID == failed.ID || retry.Status != "pending" {
		t.Fatalf("expected new pending retry, status=%d failed=%s retry=%+v", status, failed.ID, retry)
	}
}

func TestHandleDailyNewsRegeneratePreservesSuccessfulSnapshotWhileActive(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "regen-success@example.com")
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	digest.Set("period_start", "2026-05-07T06:00:00Z")
	digest.Set("period_end", "2026-05-08T06:00:00Z")
	digest.Set("title", "Original title")
	digest.Set("body_markdown", "# Original")
	digest.Set("referenced_entry_ids", []string{"entry1"})
	digest.Set("candidate_count", 4)
	digest.Set("included_count", 3)
	digest.Set("used_subset", true)
	digest.Set("has_successful_snapshot", true)
	digest.Set("last_success_at", "2026-05-08T06:10:00Z")
	digest.Set("period_start", "2026-05-07T06:00:00Z")
	digest.Set("period_end", "2026-05-08T06:00:00Z")
	digest.Set("window_key", user.Id+"|2026-05-08|2026-05-07T06:00:00Z|2026-05-08T06:00:00Z")
	digest.Set("successful_scheduled_day_key", user.Id+"|2026-05-08")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, dto, err := HandleDailyNewsRegenerate(app, user.Id, digest.Id, mustTime("2026-05-08T08:00:00Z"))
	if err != nil || status != http.StatusAccepted || dto.ID != digest.Id || dto.Status != "pending" {
		t.Fatalf("expected accepted regeneration on same digest, status=%d dto=%+v err=%v", status, dto, err)
	}
	reloaded, _ := app.FindRecordById("daily_digests", digest.Id)
	if reloaded.GetString("body_markdown") != "# Original" || !reloaded.GetBool("has_successful_snapshot") || reloaded.GetString("successful_scheduled_day_key") == "" {
		t.Fatalf("successful snapshot was not preserved during active regeneration")
	}
	if reloaded.GetString("active_window_key") == "" || reloaded.GetString("active_scheduled_day_key") == "" {
		t.Fatalf("expected active regeneration lock keys")
	}
}

func TestHandleDailyNewsRegenerateBlocksActiveAndCrossUserAndAuth(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "regen-owner@example.com")
	other := testutil.CreateSuperuser(t, app, "regen-other@example.com")
	active := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "running", "manual")
	active.Set("period_start", "2026-05-07T06:00:00Z")
	active.Set("period_end", "2026-05-08T06:00:00Z")
	active.Set("active_window_key", user.Id+"|2026-05-08|2026-05-07T06:00:00Z|2026-05-08T06:00:00Z")
	if err := app.Save(active); err != nil {
		t.Fatalf("save active: %v", err)
	}

	status, dto, err := HandleDailyNewsRegenerate(app, user.Id, active.Id, mustTime("2026-05-08T08:00:00Z"))
	if err != nil || status != http.StatusAccepted || dto.ID != active.Id || dto.Status != "running" {
		t.Fatalf("expected selected active state, status=%d dto=%+v err=%v", status, dto, err)
	}
	status, _, err = HandleDailyNewsRegenerate(app, other.Id, active.Id, mustTime("2026-05-08T08:00:00Z"))
	if err == nil || status != http.StatusNotFound {
		t.Fatalf("expected cross-user denial without leak, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsRegenerate(app, "", active.Id, mustTime("2026-05-08T08:00:00Z"))
	if err == nil || status != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated denial, status=%d err=%v", status, err)
	}
}

func TestCompleteDailyNewsRegenerationSuccessAndFailureSnapshots(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "regen-complete@example.com")
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	digest.Set("period_start", "2026-05-07T06:00:00Z")
	digest.Set("period_end", "2026-05-08T06:00:00Z")
	digest.Set("title", "Original")
	digest.Set("body_markdown", "# Original")
	digest.Set("has_successful_snapshot", true)
	digest.Set("successful_scheduled_day_key", user.Id+"|2026-05-08")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}
	_, _, err := HandleDailyNewsRegenerate(app, user.Id, digest.Id, mustTime("2026-05-08T08:00:00Z"))
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if err := engine.CompleteDailyNewsRegeneration(app, digest.Id, engine.DailyNewsGenerateResult{Title: "New", BodyMarkdown: "# New", ReferencedEntryIDs: []string{"e2"}, CandidateCount: 5, IncludedCount: 2, UsedSubset: true}, mustTime("2026-05-08T08:01:00Z")); err != nil {
		t.Fatalf("complete success: %v", err)
	}
	reloaded, _ := app.FindRecordById("daily_digests", digest.Id)
	if reloaded.GetString("status") != "success" || reloaded.GetString("title") != "New" || reloaded.GetString("body_markdown") != "# New" || reloaded.GetString("active_window_key") != "" {
		t.Fatalf("regeneration success did not replace content and clear active state")
	}
	_, _, err = HandleDailyNewsRegenerate(app, user.Id, digest.Id, mustTime("2026-05-08T08:02:00Z"))
	if err != nil {
		t.Fatalf("second regenerate: %v", err)
	}
	if err := engine.FailDailyNewsRegeneration(app, digest.Id, "secret sk-test stack trace", mustTime("2026-05-08T08:03:00Z")); err != nil {
		t.Fatalf("complete failure: %v", err)
	}
	reloaded, _ = app.FindRecordById("daily_digests", digest.Id)
	if reloaded.GetString("status") != "failed" || reloaded.GetString("body_markdown") != "# New" || reloaded.GetString("error_message") == "secret sk-test stack trace" || reloaded.GetString("successful_scheduled_day_key") == "" {
		t.Fatalf("failed regeneration did not preserve snapshot/sanitize error")
	}
}

func TestHandleDailyNewsGenerateNowEnforcesOwnerAndAuth(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "owner@example.com")
	other := testutil.CreateSuperuser(t, app, "other@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	testutil.CreateDailyNewsSettings(t, app, other.Id, true, "08:00", "Europe/Amsterdam", "")

	status, _, err := HandleDailyNewsGenerateNow(app, "", mustTime("2026-05-08T07:30:00Z"))
	if err == nil || status != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated denial, status=%d err=%v", status, err)
	}
	status, dto, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T07:30:00Z"))
	if err != nil || status != http.StatusAccepted || dto.User != user.Id {
		t.Fatalf("expected owner-scoped job, status=%d dto=%+v err=%v", status, dto, err)
	}
	if dto.User == other.Id {
		t.Fatal("job used another user's owner id")
	}
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
