package routes

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/core"
)

func TestHandleDailyNewsSettingsMaterializesAndSavesValidSettings(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "settings@example.com")

	status, dto, err := HandleDailyNewsGetSettings(app, user.Id)
	if err != nil || status != http.StatusOK {
		t.Fatalf("get settings failed: status=%d err=%v", status, err)
	}
	if dto.User != user.Id || !dto.Enabled || dto.GenerationTime != "08:00" || dto.Timezone != "Europe/Amsterdam" {
		t.Fatalf("unexpected defaults: %+v", dto)
	}
	status, saved, err := HandleDailyNewsSaveSettings(app, user.Id, DailyNewsSettingsInput{Enabled: false, GenerationTime: "07:15", Timezone: "UTC", ExtraInstructions: "Prioritize AI releases\nUse bullets"})
	if err != nil || status != http.StatusOK {
		t.Fatalf("save settings failed: status=%d err=%v", status, err)
	}
	if saved.Enabled || saved.GenerationTime != "07:15" || saved.Timezone != "UTC" || saved.ExtraInstructions != "Prioritize AI releases\nUse bullets" {
		t.Fatalf("unexpected saved settings: %+v", saved)
	}
}

func TestHandleDailyNewsSettingsRejectsInvalidValuesWithoutMutation(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "settings-invalid@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "Keep me")

	cases := []DailyNewsSettingsInput{
		{Enabled: true, GenerationTime: "24:00", Timezone: "Europe/Amsterdam"},
		{Enabled: true, GenerationTime: "08:00", Timezone: "No/SuchZone"},
		{Enabled: true, GenerationTime: "08:00", Timezone: "Europe/Amsterdam", ExtraInstructions: string(rune(0x202e))},
	}
	for _, input := range cases {
		status, _, err := HandleDailyNewsSaveSettings(app, user.Id, input)
		if status != http.StatusBadRequest || err == nil {
			t.Fatalf("expected validation failure for %+v, status=%d err=%v", input, status, err)
		}
	}
	_, dto, err := HandleDailyNewsGetSettings(app, user.Id)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if dto.GenerationTime != "08:00" || dto.Timezone != "Europe/Amsterdam" || dto.ExtraInstructions != "Keep me" {
		t.Fatalf("invalid save mutated settings: %+v", dto)
	}
}

func TestHandleDailyNewsSettingsRequiresAuthentication(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	status, _, err := HandleDailyNewsGetSettings(app, "")
	if status != http.StatusUnauthorized || err == nil {
		t.Fatalf("expected get auth failure, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsSaveSettings(app, "", DailyNewsSettingsInput{})
	if status != http.StatusUnauthorized || err == nil {
		t.Fatalf("expected save auth failure, status=%d err=%v", status, err)
	}
}

func TestHandleDailyNewsGetDigestReturnsOwnedDigestAndDeniesCrossUser(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	owner := testutil.CreateSuperuser(t, app, "digest-owner@example.com")
	other := testutil.CreateSuperuser(t, app, "digest-other@example.com")
	digest := testutil.CreateDailyDigest(t, app, owner.Id, "2026-05-08", "success", "automatic")
	digest.Set("title", "Daily Briefing")
	digest.Set("body_markdown", "# Lead")
	digest.Set("referenced_entry_ids", []string{"entry-one"})
	digest.Set("candidate_count", 3)
	digest.Set("included_count", 1)
	digest.Set("used_subset", true)
	digest.Set("has_successful_snapshot", true)
	digest.Set("last_success_at", "2026-05-08T08:01:00Z")
	digest.Set("queued_at", "2026-05-08T08:00:00Z")
	digest.Set("started_at", "2026-05-08T08:00:10Z")
	digest.Set("heartbeat_at", "2026-05-08T08:00:20Z")
	digest.Set("attempt_finished_at", "2026-05-08T08:01:00Z")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, dto, err := HandleDailyNewsGetDigest(app, owner.Id, digest.Id)
	if err != nil || status != http.StatusOK {
		t.Fatalf("expected owned digest, status=%d err=%v", status, err)
	}
	if dto.ID != digest.Id || dto.User != owner.Id || dto.Title != "Daily Briefing" || dto.BodyMarkdown != "# Lead" || dto.CandidateCount != 3 || dto.IncludedCount != 1 || !dto.UsedSubset || len(dto.ReferencedIDs) != 1 {
		t.Fatalf("unexpected digest dto: %+v", dto)
	}
	if !dto.HasSuccessfulSnapshot || dto.LastSuccessAt == "" || dto.QueuedAt == "" || dto.StartedAt == "" || dto.HeartbeatAt == "" || dto.AttemptFinishedAt == "" {
		t.Fatalf("digest dto missing snapshot/attempt metadata: %+v", dto)
	}

	status, _, err = HandleDailyNewsGetDigest(app, other.Id, digest.Id)
	if status != http.StatusNotFound || err == nil {
		t.Fatalf("expected cross-user safe not found, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsGetDigest(app, "", digest.Id)
	if status != http.StatusUnauthorized || err == nil {
		t.Fatalf("expected auth denial, status=%d err=%v", status, err)
	}
}

func TestHandleDailyNewsEntryReferenceReturnsSanitizedReferencedEntry(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "ref@example.com")
	resource := testutil.CreateResource(t, app, "Source", "https://source.example/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Referenced story", "https://source.example/story", "guid-ref")
	entry.Set("summary", "Useful summary")
	entry.Set("takeaways", []string{"First takeaway", "Second takeaway"})
	entry.Set("ai_stars", 4)
	entry.Set("user_stars", 5)
	if err := app.Save(entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	digest.Set("referenced_entry_ids", []string{entry.Id})
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, dto, err := HandleDailyNewsEntryReference(app, user.Id, digest.Id, entry.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK || !dto.Available || dto.Entry == nil {
		t.Fatalf("expected available entry, status=%d dto=%+v", status, dto)
	}
	if dto.Entry.ID != entry.Id || dto.Entry.Title != "Referenced story" || dto.Entry.URL != "https://source.example/story" || dto.Entry.Summary != "Useful summary" || dto.Entry.EffectiveStars != 5 {
		t.Fatalf("unexpected entry dto: %+v", dto.Entry)
	}
	if len(dto.Entry.Takeaways) != 2 || dto.Entry.Takeaways[0] != "First takeaway" {
		t.Fatalf("unexpected takeaways: %+v", dto.Entry.Takeaways)
	}
}

func TestHandleDailyNewsEntryReferenceDeniesCrossUserAndNonReferencedWithoutLeak(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	owner := testutil.CreateSuperuser(t, app, "owner-ref@example.com")
	other := testutil.CreateSuperuser(t, app, "other-ref@example.com")
	resource := testutil.CreateResource(t, app, "Source", "https://source.example/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Referenced story", "https://source.example/story", "guid-ref")
	digest := testutil.CreateDailyDigest(t, app, owner.Id, "2026-05-08", "success", "automatic")
	digest.Set("referenced_entry_ids", []string{entry.Id})
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, _, err := HandleDailyNewsEntryReference(app, other.Id, digest.Id, entry.Id)
	if status != http.StatusNotFound || err == nil {
		t.Fatalf("expected cross-user safe not found, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsEntryReference(app, owner.Id, digest.Id, "missingentryid")
	if status != http.StatusNotFound || err == nil {
		t.Fatalf("expected non-referenced safe not found, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsEntryReference(app, "", digest.Id, entry.Id)
	if status != http.StatusUnauthorized || err == nil {
		t.Fatalf("expected auth denial, status=%d err=%v", status, err)
	}
}

func TestHandleDailyNewsEntryReferenceReportsUnavailableForDeletedReferencedEntry(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "deleted-ref@example.com")
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	digest.Set("referenced_entry_ids", []string{"deletedentry"})
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, dto, err := HandleDailyNewsEntryReference(app, user.Id, digest.Id, "deletedentry")
	if err != nil {
		t.Fatalf("unexpected unavailable response error: %v", err)
	}
	if status != http.StatusOK || dto.Available || dto.Message != "Referenced entry is no longer available." || dto.Entry != nil {
		t.Fatalf("expected unavailable dto, status=%d dto=%+v", status, dto)
	}
}

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

func TestHandleDailyNewsGenerateNowReusesPreDueManualJobAcrossSeconds(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "predue-active@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	now := mustTime("2026-05-08T05:30:01Z")
	_, first, err := HandleDailyNewsGenerateNow(app, user.Id, now)
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}

	status, second, err := HandleDailyNewsGenerateNow(app, user.Id, now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("second generate failed: %v", err)
	}
	if status != http.StatusAccepted || second.ID != first.ID {
		t.Fatalf("expected pre-due active job reuse across seconds, status=%d first=%s second=%s", status, first.ID, second.ID)
	}
}

func TestHandleDailyNewsGenerateNowDueReusesActivePreDueManualJob(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "predue-due-active@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	_, first, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T05:30:00Z"))
	if err != nil {
		t.Fatalf("pre-due generate failed: %v", err)
	}

	status, second, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T06:00:00Z"))
	if err != nil {
		t.Fatalf("due generate failed: %v", err)
	}
	if status != http.StatusAccepted || second.ID != first.ID {
		t.Fatalf("expected due generate to reuse active pre-due manual job, status=%d first=%s second=%s", status, first.ID, second.ID)
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

func TestHandleDailyNewsGenerateNowReturnsActiveScheduledRegeneration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "active-scheduled@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	status, dto, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T07:30:00Z"))
	if err != nil || status != http.StatusAccepted {
		t.Fatalf("queue failed: status=%d err=%v", status, err)
	}
	if err := engine.CompleteDailyNewsJob(app, dto.ID, "success", "", mustTime("2026-05-08T07:45:00Z")); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	status, regenerating, err := HandleDailyNewsRegenerate(app, user.Id, dto.ID, mustTime("2026-05-08T08:05:00Z"))
	if err != nil || status != http.StatusAccepted || regenerating.Status != "pending" {
		t.Fatalf("regenerate failed: status=%d dto=%+v err=%v", status, regenerating, err)
	}

	status, again, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T08:06:00Z"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusAccepted || again.ID != dto.ID || again.Status != "pending" {
		t.Fatalf("expected active scheduled regeneration, status=%d dto=%+v", status, again)
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

func TestHandleDailyNewsGenerateAndRegenerateWakeWorkerForQueuedJobs(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "wake@example.com")
	var wakeCount int32
	oldWake := wakeDailyNewsWorker
	wakeDailyNewsWorker = func(core.App, time.Time) { atomic.AddInt32(&wakeCount, 1) }
	t.Cleanup(func() { wakeDailyNewsWorker = oldWake })

	status, dto, err := HandleDailyNewsGenerateNow(app, user.Id, mustTime("2026-05-08T05:00:00Z"))
	if err != nil || status != http.StatusAccepted || dto.Status != "pending" {
		t.Fatalf("expected accepted pending generate, status=%d dto=%+v err=%v", status, dto, err)
	}
	if atomic.LoadInt32(&wakeCount) != 1 {
		t.Fatalf("expected worker wake after generate queued, got %d", wakeCount)
	}

	reloaded, err := app.FindRecordById("daily_digests", dto.ID)
	if err != nil {
		t.Fatalf("find generated digest: %v", err)
	}
	if err := engine.CompleteDailyNewsJob(app, reloaded.Id, "failed", "retry me", mustTime("2026-05-08T05:01:00Z")); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	status, dto, err = HandleDailyNewsRegenerate(app, user.Id, reloaded.Id, mustTime("2026-05-08T05:02:00Z"))
	if err != nil || status != http.StatusAccepted || dto.Status != "pending" {
		t.Fatalf("expected accepted pending regeneration, status=%d dto=%+v err=%v", status, dto, err)
	}
	if atomic.LoadInt32(&wakeCount) != 2 {
		t.Fatalf("expected worker wake after regenerate queued, got %d", wakeCount)
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

func TestHandleDailyNewsRegenerateManualDigestSetsSameDayActiveLock(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "regen-manual-lock@example.com")
	digest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "manual")
	digest.Set("period_start", "2026-05-07T08:00:00Z")
	digest.Set("period_end", "2026-05-08T08:00:00Z")
	if err := app.Save(digest); err != nil {
		t.Fatalf("save digest: %v", err)
	}

	status, _, err := HandleDailyNewsRegenerate(app, user.Id, digest.Id, mustTime("2026-05-08T08:05:00Z"))
	if err != nil || status != http.StatusAccepted {
		t.Fatalf("expected accepted regeneration, status=%d err=%v", status, err)
	}
	reloaded, err := app.FindRecordById("daily_digests", digest.Id)
	if err != nil {
		t.Fatalf("reload digest: %v", err)
	}
	wantKey := user.Id + "|2026-05-08"
	if reloaded.GetString("active_scheduled_day_key") != wantKey {
		t.Fatalf("manual regeneration did not claim same-day active lock: got %q want %q", reloaded.GetString("active_scheduled_day_key"), wantKey)
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

func TestHandleDailyNewsListDigestsReturnsExplicitEmptyState(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "empty-digests@example.com")

	status, result, err := HandleDailyNewsListDigests(app, user.Id, "", 10, 0)
	if err != nil || status != http.StatusOK {
		t.Fatalf("expected empty list success, status=%d result=%+v err=%v", status, result, err)
	}
	if result.Latest != nil || result.Selected != nil || len(result.Archive) != 0 || result.HasMore {
		t.Fatalf("expected nil latest/selected empty archive, got %+v", result)
	}
}

func TestHandleDailyNewsListDigestsReturnsLatestArchiveAndSelectedOwnedDigest(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "archive@example.com")
	other := testutil.CreateSuperuser(t, app, "archive-other@example.com")
	older := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-07", "success", "automatic")
	older.Set("title", "Older")
	older.Set("body_markdown", "# Older")
	older.Set("period_end", "2026-05-07T06:00:00Z")
	if err := app.Save(older); err != nil {
		t.Fatalf("save older: %v", err)
	}
	middle := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-07", "failed", "manual")
	middle.Set("title", "Middle")
	middle.Set("period_end", "2026-05-07T12:00:00Z")
	if err := app.Save(middle); err != nil {
		t.Fatalf("save middle: %v", err)
	}
	latest := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	latest.Set("title", "Latest")
	latest.Set("body_markdown", "# Latest")
	latest.Set("period_end", "2026-05-08T06:00:00Z")
	latest.Set("candidate_count", 3)
	latest.Set("included_count", 2)
	latest.Set("used_subset", true)
	if err := app.Save(latest); err != nil {
		t.Fatalf("save latest: %v", err)
	}
	otherDigest := testutil.CreateDailyDigest(t, app, other.Id, "2026-05-09", "success", "automatic")
	otherDigest.Set("period_end", "2026-05-09T06:00:00Z")
	if err := app.Save(otherDigest); err != nil {
		t.Fatalf("save other: %v", err)
	}

	status, result, err := HandleDailyNewsListDigests(app, user.Id, "", 1, 0)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list failed: status=%d err=%v", status, err)
	}
	if result.Latest.ID != latest.Id || result.Selected.ID != latest.Id || result.Latest.Title != "Latest" || result.Latest.BodyMarkdown != "# Latest" {
		t.Fatalf("expected latest selected digest DTO, got %+v", result)
	}
	if len(result.Archive) != 1 || result.Archive[0].ID != middle.Id || !result.HasMore {
		t.Fatalf("expected paginated owner archive with has_more, got %+v", result)
	}
	if result.Archive[0].ID == otherDigest.Id {
		t.Fatal("archive leaked another user's digest")
	}

	status, selected, err := HandleDailyNewsListDigests(app, user.Id, older.Id, 10, 0)
	if err != nil || status != http.StatusOK || selected.Selected.ID != older.Id || selected.Latest.ID != latest.Id {
		t.Fatalf("expected explicit owned selection, status=%d result=%+v err=%v", status, selected, err)
	}
	status, _, err = HandleDailyNewsListDigests(app, user.Id, otherDigest.Id, 10, 0)
	if err == nil || status != http.StatusNotFound {
		t.Fatalf("expected cross-user selected digest denial, status=%d err=%v", status, err)
	}
	status, _, err = HandleDailyNewsListDigests(app, "", "", 10, 0)
	if err == nil || status != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated denial, status=%d err=%v", status, err)
	}
}

func TestHandleDailyNewsListDigestsPrefersPendingRetryForSameWindow(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "archive-pending-retry@example.com")

	failed := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "failed", "automatic")
	failed.Set("title", "Failed attempt")
	failed.Set("period_end", "2026-05-08T06:00:00Z")
	failed.Set("attempt_finished_at", "2026-05-08T06:01:00Z")
	if err := app.Save(failed); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	retry := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "pending", "automatic")
	retry.Set("title", "Retry pending")
	retry.Set("period_end", "2026-05-08T06:00:00Z")
	retry.Set("queued_at", "2026-05-08T06:02:00Z")
	if err := app.Save(retry); err != nil {
		t.Fatalf("save retry: %v", err)
	}

	status, result, err := HandleDailyNewsListDigests(app, user.Id, "", 10, 0)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list failed: status=%d err=%v", status, err)
	}
	if result.Latest.ID != retry.Id || result.Selected.ID != retry.Id {
		t.Fatalf("expected pending retry as latest, got %+v", result)
	}
	if len(result.Archive) != 1 || result.Archive[0].ID != failed.Id {
		t.Fatalf("expected failed attempt in archive after retry, got %+v", result.Archive)
	}
}

func TestHandleDailyNewsListDigestsPrefersSuccessfulRetryForSameWindow(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "archive-success-retry@example.com")

	failed := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "failed", "automatic")
	failed.Set("title", "Failed attempt")
	failed.Set("period_end", "2026-05-08T06:00:00Z")
	failed.Set("attempt_finished_at", "2026-05-08T06:01:00Z")
	if err := app.Save(failed); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	success := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	success.Set("title", "Successful retry")
	success.Set("body_markdown", "# Successful retry")
	success.Set("period_end", "2026-05-08T06:00:00Z")
	success.Set("last_success_at", "2026-05-08T06:03:00Z")
	success.Set("attempt_finished_at", "2026-05-08T06:03:00Z")
	if err := app.Save(success); err != nil {
		t.Fatalf("save success: %v", err)
	}

	status, result, err := HandleDailyNewsListDigests(app, user.Id, "", 10, 0)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list failed: status=%d err=%v", status, err)
	}
	if result.Latest.ID != success.Id || result.Selected.ID != success.Id || result.Latest.Title != "Successful retry" {
		t.Fatalf("expected successful retry as latest, got %+v", result)
	}
	if len(result.Archive) != 1 || result.Archive[0].ID != failed.Id {
		t.Fatalf("expected failed attempt in archive after success, got %+v", result.Archive)
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
