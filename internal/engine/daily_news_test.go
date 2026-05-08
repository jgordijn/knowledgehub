package engine

import (
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"

	"github.com/pocketbase/pocketbase/core"
)

func TestDailyNewsDigestWindowAndCandidates(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "Feed", "https://example.com/feed", "rss", "healthy", 0, true)
	userID := "user_daily_news"
	periodEnd := time.Date(2026, 5, 8, 8, 0, 0, 0, time.UTC)

	createEntryAt := func(title string, publishedAt, discoveredAt time.Time) *core.Record {
		record := testutil.CreateEntry(t, app, resource.Id, title, "https://example.com/"+title, title)
		record.Set("published_at", publishedAt.Format(time.RFC3339))
		record.Set("discovered_at", discoveredAt.Format(time.RFC3339))
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to update entry dates: %v", err)
		}
		return record
	}

	t.Run("previous successful digest defines lower bound", func(t *testing.T) {
		previousEnd := periodEnd.Add(-12 * time.Hour)
		previous := testutil.CreateDailyDigest(t, app, userID, "2026-05-07", "success", "automatic")
		previous.Set("period_end", previousEnd.Format(time.RFC3339))
		if err := app.Save(previous); err != nil {
			t.Fatalf("failed to update previous digest: %v", err)
		}
		inside := createEntryAt("inside-success-window", previousEnd.Add(time.Minute), previousEnd.Add(2*time.Minute))
		createEntryAt("outside-success-window", previousEnd.Add(-time.Minute), previousEnd.Add(-time.Minute))

		window, candidates, err := FindDailyNewsCandidates(app, userID, periodEnd)
		if err != nil {
			t.Fatalf("FindDailyNewsCandidates returned error: %v", err)
		}
		if !window.Start.Equal(previousEnd) {
			t.Fatalf("window start = %s, want %s", window.Start, previousEnd)
		}
		assertCandidateIDs(t, candidates, inside.Id)
	})
}

func TestDailyNewsCandidatesFallbackFailedDigestAndDateMatching(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "Feed", "https://example.com/feed", "rss", "healthy", 0, true)
	userID := "user_daily_news"
	periodEnd := time.Date(2026, 5, 8, 8, 0, 0, 0, time.UTC)
	fallbackStart := periodEnd.Add(-24 * time.Hour)

	createEntryAt := func(title string, publishedAt, discoveredAt time.Time) *core.Record {
		record := testutil.CreateEntry(t, app, resource.Id, title, "https://example.com/"+title, title)
		record.Set("published_at", publishedAt.Format(time.RFC3339))
		record.Set("discovered_at", discoveredAt.Format(time.RFC3339))
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to update entry dates: %v", err)
		}
		return record
	}

	failed := testutil.CreateDailyDigest(t, app, userID, "2026-05-08", "failed", "automatic")
	failed.Set("period_end", periodEnd.Add(-2*time.Hour).Format(time.RFC3339))
	if err := app.Save(failed); err != nil {
		t.Fatalf("failed to update failed digest: %v", err)
	}

	publishedMatch := createEntryAt("published-match", fallbackStart.Add(time.Hour), fallbackStart.Add(-time.Hour))
	discoveredMatch := createEntryAt("discovered-match", fallbackStart.Add(-time.Hour), fallbackStart.Add(time.Hour))
	createEntryAt("outside-window", fallbackStart.Add(-time.Minute), fallbackStart.Add(-time.Minute))

	window, candidates, err := FindDailyNewsCandidates(app, userID, periodEnd)
	if err != nil {
		t.Fatalf("FindDailyNewsCandidates returned error: %v", err)
	}
	if !window.Start.Equal(fallbackStart) {
		t.Fatalf("window start = %s, want 24 hour fallback %s", window.Start, fallbackStart)
	}
	assertCandidateIDs(t, candidates, publishedMatch.Id, discoveredMatch.Id)
}

func assertCandidateIDs(t *testing.T, candidates []*core.Record, want ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, candidate := range candidates {
		got[candidate.Id] = true
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d (ids=%v)", len(got), len(want), got)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("missing candidate %s in %v", id, got)
		}
	}
}
