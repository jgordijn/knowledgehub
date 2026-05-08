package engine

import (
	"testing"
	"time"

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
