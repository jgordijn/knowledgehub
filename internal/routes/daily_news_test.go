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
