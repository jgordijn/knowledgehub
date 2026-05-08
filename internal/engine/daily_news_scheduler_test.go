package engine

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"

	"github.com/pocketbase/pocketbase/core"
)

func TestDailyNewsDueChecksAndValidation(t *testing.T) {
	amsterdam, _ := time.LoadLocation("Europe/Amsterdam")
	settings := DailyNewsScheduleSettings{Enabled: true, GenerationTime: "08:00", Timezone: "Europe/Amsterdam"}

	if err := ValidateDailyNewsScheduleSettings(settings); err != nil {
		t.Fatalf("valid settings rejected: %v", err)
	}
	if err := ValidateDailyNewsScheduleSettings(DailyNewsScheduleSettings{Enabled: true, GenerationTime: "24:00", Timezone: "Europe/Amsterdam"}); err == nil {
		t.Fatalf("invalid generation time accepted")
	}
	if err := ValidateDailyNewsScheduleSettings(DailyNewsScheduleSettings{Enabled: true, GenerationTime: "08:00", Timezone: "No/SuchZone"}); err == nil {
		t.Fatalf("invalid timezone accepted")
	}
	if due, localDate, periodEnd, err := IsDailyNewsDue(settings, time.Date(2026, 5, 8, 7, 59, 0, 0, amsterdam)); err != nil || due || localDate != "2026-05-08" || !periodEnd.IsZero() {
		t.Fatalf("pre-due = due:%v localDate:%s periodEnd:%s err:%v", due, localDate, periodEnd, err)
	}
	wantEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)
	if due, localDate, periodEnd, err := IsDailyNewsDue(settings, time.Date(2026, 5, 8, 10, 30, 0, 0, amsterdam)); err != nil || !due || localDate != "2026-05-08" || !periodEnd.Equal(wantEnd) {
		t.Fatalf("same-day catch-up = due:%v localDate:%s periodEnd:%s want %s err:%v", due, localDate, periodEnd, wantEnd, err)
	}
	if due, _, _, err := IsDailyNewsDue(DailyNewsScheduleSettings{Enabled: false, GenerationTime: "08:00", Timezone: "Europe/Amsterdam"}, time.Date(2026, 5, 8, 10, 0, 0, 0, amsterdam)); err != nil || due {
		t.Fatalf("disabled settings should not be due, due=%v err=%v", due, err)
	}
}

func TestRunDailyNewsScheduleMaterializesNewSuperuserSettingsAndClaimsDueJobs(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-new-user@example.com")

	created, err := RunDailyNewsSchedule(app, time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC))
	if err != nil || created != 1 {
		t.Fatalf("created due jobs=%d err=%v", created, err)
	}
	settings, err := app.FindRecordsByFilter("daily_news_settings", "user = {:user}", "", 10, 0, map[string]any{"user": user.Id})
	if err != nil || len(settings) != 1 || settings[0].GetString("generation_time") != "08:00" {
		t.Fatalf("default settings not materialized: len=%d err=%v", len(settings), err)
	}
}

func TestRunDailyNewsScheduleClaimsDueEnabledSettings(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-schedule@example.com")
	disabledUser := testutil.CreateSuperuser(t, app, "daily-news-disabled@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	testutil.CreateDailyNewsSettings(t, app, disabledUser.Id, false, "08:00", "Europe/Amsterdam", "")

	created, err := RunDailyNewsSchedule(app, time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC))
	if err != nil || created != 1 {
		t.Fatalf("created due jobs=%d err=%v", created, err)
	}
	dueJobs, err := app.FindRecordsByFilter("daily_digests", "user = {:user}", "", 10, 0, map[string]any{"user": user.Id})
	if err != nil || len(dueJobs) != 1 || dueJobs[0].GetString("status") != "pending" || dueJobs[0].GetString("trigger") != "automatic" {
		t.Fatalf("due user jobs=%d status=%q trigger=%q err=%v", len(dueJobs), firstString(dueJobs, "status"), firstString(dueJobs, "trigger"), err)
	}
	disabledJobs, err := app.FindRecordsByFilter("daily_digests", "user = {:user}", "", 10, 0, map[string]any{"user": disabledUser.Id})
	if err != nil || len(disabledJobs) != 0 {
		t.Fatalf("disabled user jobs=%d err=%v", len(disabledJobs), err)
	}

	created, err = RunDailyNewsSchedule(app, time.Date(2026, 5, 8, 11, 0, 0, 0, time.UTC))
	if err != nil || created != 0 {
		t.Fatalf("duplicate schedule created=%d err=%v", created, err)
	}
}

func TestProcessPendingDailyNewsJobsUsesStoredRegenerationWindow(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-regenerate-window@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	testutil.CreateSetting(t, app, ai.SettingAPIKey, "test-key")
	resource := testutil.CreateResource(t, app, "Older Source", "https://example.com/feed", "rss", "healthy", 0, true)
	oldStart := time.Date(2026, 5, 6, 6, 0, 0, 0, time.UTC)
	oldEnd := time.Date(2026, 5, 7, 6, 0, 0, 0, time.UTC)
	newEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)
	oldEntry := testutil.CreateEntryWithStars(t, app, resource.Id, "Older article", "https://example.com/old", 4, 0)
	oldEntry.Set("discovered_at", oldStart.Add(time.Hour).Format(time.RFC3339))
	if err := app.Save(oldEntry); err != nil {
		t.Fatalf("save old entry: %v", err)
	}
	newEntry := testutil.CreateEntryWithStars(t, app, resource.Id, "Newer article", "https://example.com/new", 5, 0)
	newEntry.Set("discovered_at", oldEnd.Add(time.Hour).Format(time.RFC3339))
	if err := app.Save(newEntry); err != nil {
		t.Fatalf("save new entry: %v", err)
	}
	newerSuccess := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "success", "automatic")
	newerSuccess.Set("period_start", oldEnd.Format(time.RFC3339))
	newerSuccess.Set("period_end", newEnd.Format(time.RFC3339))
	newerSuccess.Set("has_successful_snapshot", true)
	if err := app.Save(newerSuccess); err != nil {
		t.Fatalf("save newer success: %v", err)
	}
	_, _, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-07", PeriodStart: oldStart, PeriodEnd: oldEnd, Trigger: "manual", Scheduled: false, Now: newEnd})
	if err != nil {
		t.Fatalf("claim old job: %v", err)
	}
	var prompt string
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		prompt = messages[1].Content
		return `{"title":"Old Daily","body_markdown":"# Old Daily","referenced_entry_ids":["` + oldEntry.Id + `"]}`, nil
	})
	defer restore()

	processed, err := ProcessPendingDailyNewsJobs(app, newEnd.Add(time.Minute))
	if err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	if !strings.Contains(prompt, "Window UTC: 2026-05-06T06:00:00Z to 2026-05-07T06:00:00Z") || !strings.Contains(prompt, oldEntry.Id) || strings.Contains(prompt, newEntry.Id) {
		t.Fatalf("prompt did not use stored old window/candidates:\n%s", prompt)
	}
}

func TestProcessPendingDailyNewsJobsRefreshesHeartbeatDuringGeneration(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "heartbeat@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	testutil.CreateSetting(t, app, "openrouter_api_key", "test-key")
	resource := testutil.CreateResource(t, app, "Source", "https://example.com/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "A", "https://example.com/a", "a")
	entry.Set("published_at", "2026-05-08T05:00:00Z")
	if err := app.Save(entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	periodEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)
	job, _, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd})
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}

	oldInterval := dailyNewsHeartbeatInterval
	dailyNewsHeartbeatInterval = 10 * time.Millisecond
	t.Cleanup(func() { dailyNewsHeartbeatInterval = oldInterval })
	var observed int32
	release := make(chan struct{})
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		deadline := time.After(250 * time.Millisecond)
		for atomic.LoadInt32(&observed) == 0 {
			updated, _ := app.FindRecordById("daily_digests", job.Id)
			if updated.GetDateTime("heartbeat_at").Time().After(periodEnd) {
				atomic.StoreInt32(&observed, 1)
				close(release)
				break
			}
			select {
			case <-deadline:
				return "", nil
			case <-time.After(5 * time.Millisecond):
			}
		}
		<-release
		return `{"title":"Brief","body_markdown":"# Brief","referenced_entry_ids":[]}`, nil
	})
	defer restore()

	processed, err := ProcessPendingDailyNewsJobs(app, periodEnd)
	if err != nil || processed != 1 {
		t.Fatalf("process jobs: processed=%d err=%v", processed, err)
	}
	if atomic.LoadInt32(&observed) == 0 {
		t.Fatalf("expected heartbeat to advance while generation was running")
	}
}

func TestProcessPendingDailyNewsJobsStoresClearMissingAPIKeyFailureOnlyWhenCandidatesNeedAI(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-missing-api@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "")
	periodEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)

	emptyJob, _, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd})
	if err != nil {
		t.Fatalf("claim empty job: %v", err)
	}
	processed, err := ProcessPendingDailyNewsJobs(app, periodEnd.Add(time.Minute))
	if err != nil || processed != 1 {
		t.Fatalf("empty processed=%d err=%v", processed, err)
	}
	updatedEmpty, err := app.FindRecordById("daily_digests", emptyJob.Id)
	if err != nil {
		t.Fatalf("find empty digest: %v", err)
	}
	if updatedEmpty.GetString("status") != "success" || updatedEmpty.GetString("title") != "No articles today" || !updatedEmpty.GetBool("has_successful_snapshot") {
		t.Fatalf("expected no-candidate success without api key, status=%q title=%q snapshot=%v", updatedEmpty.GetString("status"), updatedEmpty.GetString("title"), updatedEmpty.GetBool("has_successful_snapshot"))
	}

	resource := testutil.CreateResource(t, app, "Source", "https://example.com/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntry(t, app, resource.Id, "Needs AI", "https://example.com/needs-ai", "needs-ai")
	entry.Set("discovered_at", "2026-05-09T05:30:00Z")
	if err := app.Save(entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	nextEnd := periodEnd.Add(24 * time.Hour)
	job, _, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-09", PeriodStart: periodEnd, PeriodEnd: nextEnd, Trigger: "automatic", Scheduled: true, Now: nextEnd})
	if err != nil {
		t.Fatalf("claim candidate job: %v", err)
	}

	processed, err = ProcessPendingDailyNewsJobs(app, nextEnd.Add(time.Minute))
	if err != nil || processed != 1 {
		t.Fatalf("candidate processed=%d err=%v", processed, err)
	}
	updated, err := app.FindRecordById("daily_digests", job.Id)
	if err != nil {
		t.Fatalf("find digest: %v", err)
	}
	if updated.GetString("status") != "failed" || updated.GetString("error_message") != "OpenRouter API key is not configured. Configure it in Settings before generating Daily News." {
		t.Fatalf("expected clear missing API key failure, status=%q error=%q", updated.GetString("status"), updated.GetString("error_message"))
	}
}

func TestProcessPendingDailyNewsJobsGeneratesTerminalDigest(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-worker@example.com")
	testutil.CreateDailyNewsSettings(t, app, user.Id, true, "08:00", "Europe/Amsterdam", "Focus on impact")
	testutil.CreateSetting(t, app, ai.SettingAPIKey, "test-key")
	testutil.CreateSetting(t, app, ai.SettingModel, "test-model")
	resource := testutil.CreateResource(t, app, "Source", "https://example.com/feed", "rss", "healthy", 0, true)
	entry := testutil.CreateEntryWithStars(t, app, resource.Id, "Important", "https://example.com/important", 5, 0)
	entry.Set("summary", "A useful summary")
	entry.Set("discovered_at", "2026-05-08T05:30:00Z")
	if err := app.Save(entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	periodEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)
	job, _, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd})
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	var prompt string
	restore := ai.SetCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		if apiKey != "test-key" || model != "test-model" {
			t.Fatalf("unexpected ai config %q/%q", apiKey, model)
		}
		prompt = messages[1].Content
		return `{"title":"Daily","body_markdown":"# Daily\n[[kh-entry:` + entry.Id + `]]","referenced_entry_ids":["` + entry.Id + `"]}`, nil
	})
	defer restore()

	processed, err := ProcessPendingDailyNewsJobs(app, periodEnd.Add(time.Minute))
	if err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	if !strings.Contains(prompt, `"source":"Source"`) || strings.Contains(prompt, `"source":"`+resource.Id+`"`) {
		t.Fatalf("worker prompt should contain human-readable source name, got:\n%s", prompt)
	}
	updated, _ := app.FindRecordById("daily_digests", job.Id)
	if updated.GetString("status") != "success" || updated.GetString("title") != "Daily" || !updated.GetBool("has_successful_snapshot") || updated.GetString("active_window_key") != "" {
		t.Fatalf("job not completed successfully: status=%q title=%q snapshot=%v active=%q", updated.GetString("status"), updated.GetString("title"), updated.GetBool("has_successful_snapshot"), updated.GetString("active_window_key"))
	}
}

func TestDailyNewsJobClaimLifecycleAndRecovery(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-jobs@example.com")
	periodEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)

	first, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd})
	if err != nil || !created || first.GetString("status") != "pending" {
		t.Fatalf("first claim created=%v status=%q err=%v", created, first.GetString("status"), err)
	}
	second, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd.Add(500 * time.Millisecond), Trigger: "manual", Scheduled: true, Now: periodEnd.Add(time.Millisecond)})
	if err != nil || created || second.Id != first.Id {
		t.Fatalf("duplicate scheduled/window claim created=%v got=%s want=%s err=%v", created, second.Id, first.Id, err)
	}

	claimed, ok, err := ClaimPendingDailyNewsJob(app, first.Id, periodEnd.Add(time.Minute))
	if err != nil || !ok || claimed.GetString("status") != "running" {
		t.Fatalf("pending claim ok=%v status=%q err=%v", ok, claimed.GetString("status"), err)
	}
	if _, ok, err := ClaimPendingDailyNewsJob(app, first.Id, periodEnd.Add(2*time.Minute)); err != nil || ok {
		t.Fatalf("second worker claimed running job ok=%v err=%v", ok, err)
	}
	if err := CompleteDailyNewsJob(app, first.Id, "success", "", periodEnd.Add(3*time.Minute)); err != nil {
		t.Fatalf("complete success: %v", err)
	}
	if _, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodEnd.Add(-24 * time.Hour), PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd.Add(4 * time.Minute)}); err == nil || created {
		t.Fatalf("successful scheduled day should prevent duplicate success, created=%v err=%v", created, err)
	}

	failed := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-09", "failed", "automatic")
	failed.Set("period_start", periodEnd.Format(time.RFC3339))
	failed.Set("period_end", periodEnd.Add(24*time.Hour).Format(time.RFC3339))
	if err := app.Save(failed); err != nil {
		t.Fatalf("save failed digest: %v", err)
	}
	if _, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-09", PeriodStart: periodEnd, PeriodEnd: periodEnd.Add(24 * time.Hour), Trigger: "automatic", Scheduled: true, Now: periodEnd.Add(24 * time.Hour)}); err != nil || !created {
		t.Fatalf("failed retry should create active job, created=%v err=%v", created, err)
	}
}

func TestDailyNewsConcreteLockIndexesPreventDuplicateActiveJobs(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-locks@example.com")
	periodStart := time.Date(2026, 5, 7, 6, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)

	scheduled, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodStart, PeriodEnd: periodEnd, Trigger: "automatic", Scheduled: true, Now: periodEnd})
	if err != nil || !created {
		t.Fatalf("scheduled claim created=%v err=%v", created, err)
	}
	manual, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: periodStart, PeriodEnd: periodEnd.Add(900 * time.Millisecond), Trigger: "manual", Scheduled: true, Now: periodEnd.Add(750 * time.Millisecond)})
	if err != nil || created || manual.Id != scheduled.Id {
		t.Fatalf("manual/scheduled race bypassed canonical locks: created=%v got=%s want=%s err=%v", created, manual.Id, scheduled.Id, err)
	}

	duplicate := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-09", "pending", "manual")
	duplicate.Set("active_window_key", scheduled.GetString("active_window_key"))
	if err := app.Save(duplicate); err == nil {
		t.Fatalf("database accepted duplicate non-empty active_window_key")
	}
}

func TestDailyNewsPreDueManualClaimsReuseActiveSameDayManualLock(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-predue-manual-dedupe@example.com")
	firstEnd := time.Date(2026, 5, 8, 5, 30, 0, 0, time.UTC)
	secondEnd := firstEnd.Add(time.Second)

	first, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: firstEnd.Add(-24 * time.Hour), PeriodEnd: firstEnd, Trigger: "manual", Scheduled: false, Now: firstEnd})
	if err != nil || !created {
		t.Fatalf("first pre-due manual claim created=%v err=%v", created, err)
	}
	second, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: secondEnd.Add(-24 * time.Hour), PeriodEnd: secondEnd, Trigger: "manual", Scheduled: false, Now: secondEnd})
	if err != nil || created || second.Id != first.Id {
		t.Fatalf("same-day active manual claim was not reused: created=%v got=%s want=%s err=%v", created, second.Id, first.Id, err)
	}
	if first.GetString("active_scheduled_day_key") == "" {
		t.Fatalf("pre-due manual claim should use a concrete active day lock")
	}
}

func TestDailyNewsPreDueManualAndLaterScheduledUseSeparateLocks(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-predue-locks@example.com")
	manualEnd := time.Date(2026, 5, 8, 5, 30, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, 5, 8, 6, 0, 0, 0, time.UTC)

	manual, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: manualEnd.Add(-24 * time.Hour), PeriodEnd: manualEnd, Trigger: "manual", Scheduled: false, Now: manualEnd})
	if err != nil || !created {
		t.Fatalf("pre-due manual claim created=%v err=%v", created, err)
	}
	if err := CompleteDailyNewsJob(app, manual.Id, "success", "", manualEnd.Add(time.Minute)); err != nil {
		t.Fatalf("complete manual: %v", err)
	}
	manual, err = app.FindRecordById("daily_digests", manual.Id)
	if err != nil {
		t.Fatalf("reload manual: %v", err)
	}
	if manual.GetString("successful_scheduled_day_key") != "" {
		t.Fatalf("pre-due manual should not reserve scheduled success key")
	}
	scheduled, created, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{UserID: user.Id, LocalDate: "2026-05-08", PeriodStart: manualEnd, PeriodEnd: scheduledEnd, Trigger: "automatic", Scheduled: true, Now: scheduledEnd})
	if err != nil || !created || scheduled.Id == manual.Id {
		t.Fatalf("later scheduled digest should be independently claimable, created=%v err=%v", created, err)
	}
}

func TestDailyNewsStaleRecovery(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()
	user := testutil.CreateSuperuser(t, app, "daily-news-stale@example.com")
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	pending := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-08", "pending", "automatic")
	pending.Set("queued_at", now.Add(-2*time.Hour).Format(time.RFC3339))
	pending.Set("active_window_key", "pending-key")
	if err := app.Save(pending); err != nil {
		t.Fatalf("save pending: %v", err)
	}
	running := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-09", "running", "automatic")
	running.Set("started_at", now.Add(-2*time.Hour).Format(time.RFC3339))
	running.Set("heartbeat_at", now.Add(-2*time.Hour).Format(time.RFC3339))
	running.Set("active_window_key", "running-key")
	if err := app.Save(running); err != nil {
		t.Fatalf("save running: %v", err)
	}
	fresh := testutil.CreateDailyDigest(t, app, user.Id, "2026-05-10", "running", "automatic")
	fresh.Set("started_at", now.Add(-5*time.Minute).Format(time.RFC3339))
	fresh.Set("heartbeat_at", now.Add(-5*time.Minute).Format(time.RFC3339))
	fresh.Set("active_window_key", "fresh-key")
	if err := app.Save(fresh); err != nil {
		t.Fatalf("save fresh: %v", err)
	}

	recovered, err := RecoverStaleDailyNewsJobs(app, DailyNewsRecoveryConfig{PendingTimeout: time.Hour, RunningTimeout: time.Hour, Now: now})
	if err != nil || recovered != 2 {
		t.Fatalf("recovered=%d err=%v", recovered, err)
	}
	for _, id := range []string{pending.Id, running.Id} {
		record, _ := app.FindRecordById("daily_digests", id)
		if record.GetString("status") != "failed" || record.GetString("active_window_key") != "" || record.GetString("error_message") == "" {
			t.Fatalf("stale record not failed/cleared: status=%q active=%q error=%q", record.GetString("status"), record.GetString("active_window_key"), record.GetString("error_message"))
		}
	}
	freshRecord, _ := app.FindRecordById("daily_digests", fresh.Id)
	if freshRecord.GetString("status") != "running" || freshRecord.GetString("active_window_key") == "" {
		t.Fatalf("fresh running job was recovered unexpectedly")
	}
}

func firstString(records []*core.Record, field string) string {
	if len(records) == 0 {
		return ""
	}
	return records[0].GetString(field)
}

func TestDailyNewsDSTDueChecks(t *testing.T) {
	settings := DailyNewsScheduleSettings{Enabled: true, GenerationTime: "02:30", Timezone: "Europe/Amsterdam"}
	loc, _ := time.LoadLocation("Europe/Amsterdam")
	if due, date, _, err := IsDailyNewsDue(settings, time.Date(2026, 3, 29, 3, 30, 0, 0, loc)); err != nil || !due || date != "2026-03-29" {
		t.Fatalf("spring-forward due=%v date=%s err=%v", due, date, err)
	}
	if due, date, _, err := IsDailyNewsDue(settings, time.Date(2026, 10, 25, 2, 45, 0, 0, loc)); err != nil || !due || date != "2026-10-25" {
		t.Fatalf("fall-back due=%v date=%s err=%v", due, date, err)
	}
}
