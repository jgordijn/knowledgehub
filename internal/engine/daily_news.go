package engine

import (
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// DailyNewsWindow is the canonical input window used to select digest entries.
type DailyNewsWindow struct {
	Start time.Time
	End   time.Time
}

// FindDailyNewsCandidates returns entries visible to the target user whose
// published_at or discovered_at falls after the previous successful digest's
// period_end and at or before periodEnd. If the user has no previous successful
// digest, the window falls back to the 24 hours before periodEnd.
func FindDailyNewsCandidates(app core.App, userID string, periodEnd time.Time) (DailyNewsWindow, []*core.Record, error) {
	end := periodEnd.UTC().Truncate(time.Second)
	start, err := previousSuccessfulDigestEnd(app, userID)
	if err != nil {
		return DailyNewsWindow{}, nil, err
	}
	if start.IsZero() {
		start = end.Add(-24 * time.Hour)
	}
	start = start.UTC().Truncate(time.Second)

	entries, err := app.FindAllRecords("entries")
	if err != nil {
		return DailyNewsWindow{}, nil, err
	}
	candidates := make([]*core.Record, 0, len(entries))
	for _, entry := range entries {
		if dateInDigestWindow(entry.GetDateTime("published_at").Time(), start, end) || dateInDigestWindow(entry.GetDateTime("discovered_at").Time(), start, end) {
			candidates = append(candidates, entry)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidateSortTime(candidates[i])
		right := candidateSortTime(candidates[j])
		if !left.Equal(right) {
			return left.Before(right)
		}
		return candidates[i].Id < candidates[j].Id
	})
	return DailyNewsWindow{Start: start, End: end}, candidates, nil
}

func previousSuccessfulDigestEnd(app core.App, userID string) (time.Time, error) {
	digests, err := app.FindRecordsByFilter(
		"daily_digests",
		"user = {:user} && status = 'success' && period_end != ''",
		"-period_end",
		1,
		0,
		map[string]any{"user": userID},
	)
	if err != nil {
		return time.Time{}, err
	}
	if len(digests) == 0 {
		return time.Time{}, nil
	}
	return digests[0].GetDateTime("period_end").Time().UTC(), nil
}

func dateInDigestWindow(value time.Time, start, end time.Time) bool {
	if value.IsZero() {
		return false
	}
	value = value.UTC().Truncate(time.Second)
	return value.After(start) && (value.Equal(end) || value.Before(end))
}

func candidateSortTime(record *core.Record) time.Time {
	published := record.GetDateTime("published_at").Time().UTC()
	discovered := record.GetDateTime("discovered_at").Time().UTC()
	if published.IsZero() {
		return discovered
	}
	if discovered.IsZero() || published.Before(discovered) {
		return published
	}
	return discovered
}
