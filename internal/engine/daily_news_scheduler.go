package engine

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var dailyNewsTimePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)

var errDailyNewsScheduledSuccessExists = errors.New("successful scheduled digest already exists")

var dailyNewsHeartbeatInterval = 30 * time.Second

// DailyNewsScheduleSettings contains the user-specific values needed for due checks.
type DailyNewsScheduleSettings struct {
	Enabled        bool
	GenerationTime string
	Timezone       string
}

// DailyNewsJobClaim contains a canonical active job claim request.
type DailyNewsJobClaim struct {
	UserID      string
	LocalDate   string
	PeriodStart time.Time
	PeriodEnd   time.Time
	Trigger     string
	Scheduled   bool
	Now         time.Time
}

// DailyNewsRecoveryConfig configures stale active-job recovery.
type DailyNewsRecoveryConfig struct {
	PendingTimeout time.Duration
	RunningTimeout time.Duration
	Now            time.Time
}

func ValidateDailyNewsScheduleSettings(settings DailyNewsScheduleSettings) error {
	if !dailyNewsTimePattern.MatchString(settings.GenerationTime) {
		return fmt.Errorf("invalid daily news generation time")
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		return fmt.Errorf("invalid daily news timezone")
	}
	return nil
}

func IsDailyNewsDue(settings DailyNewsScheduleSettings, now time.Time) (bool, string, time.Time, error) {
	if err := ValidateDailyNewsScheduleSettings(settings); err != nil {
		return false, "", time.Time{}, err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	localNow := now.In(loc)
	localDate := localNow.Format("2006-01-02")
	if !settings.Enabled {
		return false, localDate, time.Time{}, nil
	}
	parts := dailyNewsTimePattern.FindStringSubmatch(settings.GenerationTime)
	hour := atoi2(parts[1])
	minute := atoi2(parts[2])
	dueLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, loc)
	if localNow.Before(dueLocal) {
		return false, localDate, time.Time{}, nil
	}
	return true, localDate, dueLocal.UTC().Truncate(time.Second), nil
}

func RunDailyNewsSchedule(app core.App, now time.Time) (int, error) {
	if err := EnsureDailyNewsSettingsForSuperusers(app); err != nil {
		return 0, err
	}
	if _, err := RecoverStaleDailyNewsJobs(app, DailyNewsRecoveryConfig{PendingTimeout: 24 * time.Hour, RunningTimeout: time.Hour, Now: now}); err != nil {
		return 0, err
	}
	settingsRecords, err := app.FindAllRecords("daily_news_settings")
	if err != nil {
		return 0, err
	}
	created := 0
	for _, settingsRecord := range settingsRecords {
		settings := DailyNewsScheduleSettings{
			Enabled:        settingsRecord.GetBool("enabled"),
			GenerationTime: settingsRecord.GetString("generation_time"),
			Timezone:       settingsRecord.GetString("timezone"),
		}
		due, localDate, periodEnd, err := IsDailyNewsDue(settings, now)
		if err != nil {
			return created, err
		}
		if !due {
			continue
		}
		userID := settingsRecord.GetString("user")
		window, _, err := FindDailyNewsCandidates(app, userID, periodEnd)
		if err != nil {
			return created, err
		}
		_, wasCreated, err := ClaimDailyNewsJob(app, DailyNewsJobClaim{
			UserID:      userID,
			LocalDate:   localDate,
			PeriodStart: window.Start,
			PeriodEnd:   window.End,
			Trigger:     "automatic",
			Scheduled:   true,
			Now:         now,
		})
		if err != nil {
			if errors.Is(err, errDailyNewsScheduledSuccessExists) {
				continue
			}
			return created, err
		}
		if wasCreated {
			created++
		}
	}
	return created, nil
}

func ClaimDailyNewsJob(app core.App, claim DailyNewsJobClaim) (*core.Record, bool, error) {
	periodStart := claim.PeriodStart.UTC().Truncate(time.Second)
	periodEnd := claim.PeriodEnd.UTC().Truncate(time.Second)
	windowKey := dailyNewsWindowKey(claim.UserID, claim.LocalDate, periodStart, periodEnd)
	activeDayKey := claim.UserID + "|" + claim.LocalDate
	scheduledDayKey := activeDayKey + "|manual"
	if claim.Scheduled {
		scheduledDayKey = activeDayKey
	}
	if existing, err := findActiveDigestForLocalDate(app, claim.UserID, claim.LocalDate); err == nil {
		return existing, false, nil
	}
	if claim.Scheduled {
		if existing, err := findDigestByKey(app, "successful_scheduled_day_key", scheduledDayKey); err == nil {
			return existing, false, errDailyNewsScheduledSuccessExists
		}
	}
	if existing, err := findDigestByKey(app, "active_window_key", windowKey); err == nil {
		return existing, false, nil
	}

	col, err := app.FindCollectionByNameOrId("daily_digests")
	if err != nil {
		return nil, false, err
	}
	record := core.NewRecord(col)
	record.Set("user", claim.UserID)
	record.Set("local_date", claim.LocalDate)
	record.Set("status", "pending")
	record.Set("trigger", claim.Trigger)
	record.Set("period_start", periodStart.Format(time.RFC3339))
	record.Set("period_end", periodEnd.Format(time.RFC3339))
	record.Set("window_key", windowKey)
	record.Set("active_window_key", windowKey)
	record.Set("scheduled_day_key", scheduledDayKey)
	record.Set("active_scheduled_day_key", activeDayKey)
	record.Set("queued_at", normalizedNow(claim.Now).Format(time.RFC3339))
	if err := app.Save(record); err != nil {
		// A concurrent writer may have won the concrete unique-index race.
		if existing, findErr := findDigestByKey(app, "active_window_key", windowKey); findErr == nil {
			return existing, false, nil
		}
		if existing, findErr := findDigestByKey(app, "active_scheduled_day_key", activeDayKey); findErr == nil {
			return existing, false, nil
		}
		return nil, false, err
	}
	return record, true, nil
}

func findActiveDigestForLocalDate(app core.App, userID, localDate string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("daily_digests", "user = {:user} && local_date = {:date} && (status = 'pending' || status = 'running')", dbx.Params{"user": userID, "date": localDate})
}

func EnsureDailyNewsSettingsForSuperusers(app core.App) error {
	superusers, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	for _, user := range superusers {
		if _, err := getOrCreateDailyNewsSettings(app, user.Id); err != nil {
			return err
		}
	}
	return nil
}

func getOrCreateDailyNewsSettings(app core.App, userID string) (*core.Record, error) {
	if existing, err := app.FindFirstRecordByFilter("daily_news_settings", "user = {:user}", dbx.Params{"user": userID}); err == nil {
		return existing, nil
	}
	col, err := app.FindCollectionByNameOrId("daily_news_settings")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	record.Set("user", userID)
	record.Set("enabled", true)
	record.Set("generation_time", "08:00")
	record.Set("timezone", "Europe/Amsterdam")
	record.Set("extra_instructions", "")
	if err := app.Save(record); err != nil {
		if winner, findErr := app.FindFirstRecordByFilter("daily_news_settings", "user = {:user}", dbx.Params{"user": userID}); findErr == nil {
			return winner, nil
		}
		return nil, err
	}
	return record, nil
}

func ClaimPendingDailyNewsJob(app core.App, id string, now time.Time) (*core.Record, bool, error) {
	var claimed *core.Record
	var ok bool
	err := app.RunInTransaction(func(txApp core.App) error {
		record, err := txApp.FindRecordById("daily_digests", id)
		if err != nil {
			return err
		}
		claimed = record
		if record.GetString("status") != "pending" {
			return nil
		}
		record.Set("status", "running")
		record.Set("started_at", normalizedNow(now).Format(time.RFC3339))
		record.Set("heartbeat_at", normalizedNow(now).Format(time.RFC3339))
		if err := txApp.Save(record); err != nil {
			return err
		}
		ok = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return claimed, ok, nil
}

func ProcessPendingDailyNewsJobs(app core.App, now time.Time) (int, error) {
	jobs, err := app.FindRecordsByFilter("daily_digests", "status = 'pending'", "queued_at", 50, 0)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, job := range jobs {
		claimed, ok, err := ClaimPendingDailyNewsJob(app, job.Id, now)
		if err != nil {
			return processed, err
		}
		if !ok {
			continue
		}
		processed++
		if err := generateClaimedDailyNewsJob(app, claimed, now); err != nil {
			return processed, err
		}
	}
	return processed, nil
}

func generateClaimedDailyNewsJob(app core.App, job *core.Record, now time.Time) error {
	settings, err := getOrCreateDailyNewsSettings(app, job.GetString("user"))
	if err != nil {
		return FailDailyNewsRegeneration(app, job.Id, err.Error(), now)
	}
	periodStart := job.GetDateTime("period_start").Time().UTC()
	periodEnd := job.GetDateTime("period_end").Time().UTC()
	window, candidates, err := FindDailyNewsCandidatesInWindow(app, job.GetString("user"), periodStart, periodEnd)
	if err != nil {
		return FailDailyNewsRegeneration(app, job.Id, err.Error(), now)
	}
	apiKey, err := ai.GetAPIKey(app)
	if (err != nil || apiKey == "") && len(candidates) > 0 {
		return FailDailyNewsRegeneration(app, job.Id, "OpenRouter API key is not configured.", now)
	}
	sourceNames, err := dailyNewsSourceNames(app, candidates)
	if err != nil {
		return FailDailyNewsRegeneration(app, job.Id, err.Error(), now)
	}
	stopHeartbeat := startDailyNewsHeartbeat(app, job.Id)
	result, err := GenerateDailyNewsDigest(app, DailyNewsGenerateInput{APIKey: apiKey, Model: ai.GetModel(app), Window: window, Candidates: candidates, ExtraInstructions: settings.GetString("extra_instructions"), SourceNames: sourceNames})
	stopHeartbeat()
	if err != nil {
		return FailDailyNewsRegeneration(app, job.Id, err.Error(), now)
	}
	return CompleteDailyNewsRegeneration(app, job.Id, result, now)
}

func startDailyNewsHeartbeat(app core.App, id string) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(dailyNewsHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = RefreshDailyNewsJobHeartbeat(app, id, time.Now())
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func RefreshDailyNewsJobHeartbeat(app core.App, id string, now time.Time) error {
	record, err := app.FindRecordById("daily_digests", id)
	if err != nil {
		return err
	}
	if record.GetString("status") != "running" {
		return nil
	}
	record.Set("heartbeat_at", normalizedNow(now).Format(time.RFC3339))
	return app.Save(record)
}

func CompleteDailyNewsRegeneration(app core.App, id string, result DailyNewsGenerateResult, now time.Time) error {
	record, err := app.FindRecordById("daily_digests", id)
	if err != nil {
		return err
	}
	record.Set("status", "success")
	record.Set("title", result.Title)
	record.Set("body_markdown", result.BodyMarkdown)
	record.Set("referenced_entry_ids", result.ReferencedEntryIDs)
	record.Set("candidate_count", result.CandidateCount)
	record.Set("included_count", result.IncludedCount)
	record.Set("used_subset", result.UsedSubset)
	record.Set("has_successful_snapshot", true)
	record.Set("last_success_at", normalizedNow(now).Format(time.RFC3339))
	record.Set("attempt_finished_at", normalizedNow(now).Format(time.RFC3339))
	record.Set("active_window_key", "")
	record.Set("active_scheduled_day_key", "")
	record.Set("error_message", "")
	if key := record.GetString("scheduled_day_key"); key != "" && record.GetString("trigger") != "manual" && record.GetString("successful_scheduled_day_key") == "" {
		record.Set("successful_scheduled_day_key", key)
	}
	return app.Save(record)
}

func FailDailyNewsRegeneration(app core.App, id, message string, now time.Time) error {
	record, err := app.FindRecordById("daily_digests", id)
	if err != nil {
		return err
	}
	record.Set("status", "failed")
	record.Set("active_window_key", "")
	record.Set("active_scheduled_day_key", "")
	record.Set("attempt_finished_at", normalizedNow(now).Format(time.RFC3339))
	record.Set("error_message", sanitizeDailyNewsError(message))
	return app.Save(record)
}

func CompleteDailyNewsJob(app core.App, id, status, message string, now time.Time) error {
	if status != "success" && status != "failed" {
		return errors.New("daily news job terminal status must be success or failed")
	}
	record, err := app.FindRecordById("daily_digests", id)
	if err != nil {
		return err
	}
	record.Set("status", status)
	record.Set("active_window_key", "")
	record.Set("active_scheduled_day_key", "")
	record.Set("attempt_finished_at", normalizedNow(now).Format(time.RFC3339))
	if status == "success" {
		record.Set("has_successful_snapshot", true)
		record.Set("last_success_at", normalizedNow(now).Format(time.RFC3339))
		if key := record.GetString("scheduled_day_key"); key != "" && record.GetString("trigger") != "manual" {
			record.Set("successful_scheduled_day_key", key)
		}
		record.Set("error_message", "")
	} else {
		record.Set("error_message", sanitizeDailyNewsError(message))
	}
	return app.Save(record)
}

func RecoverStaleDailyNewsJobs(app core.App, config DailyNewsRecoveryConfig) (int, error) {
	now := normalizedNow(config.Now)
	records, err := app.FindRecordsByFilter("daily_digests", "status = 'pending' || status = 'running'", "", 0, 0)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for _, record := range records {
		status := record.GetString("status")
		stale := false
		if status == "pending" && config.PendingTimeout > 0 {
			queuedAt := record.GetDateTime("queued_at").Time()
			stale = !queuedAt.IsZero() && !queuedAt.After(now.Add(-config.PendingTimeout))
		}
		if status == "running" && config.RunningTimeout > 0 {
			heartbeat := record.GetDateTime("heartbeat_at").Time()
			if heartbeat.IsZero() {
				heartbeat = record.GetDateTime("started_at").Time()
			}
			stale = !heartbeat.IsZero() && !heartbeat.After(now.Add(-config.RunningTimeout))
		}
		if !stale {
			continue
		}
		record.Set("status", "failed")
		record.Set("active_window_key", "")
		record.Set("active_scheduled_day_key", "")
		record.Set("attempt_finished_at", now.Format(time.RFC3339))
		record.Set("error_message", "Digest generation timed out and can be retried.")
		if err := app.Save(record); err != nil {
			return recovered, err
		}
		recovered++
	}
	return recovered, nil
}

func findDigestByKey(app core.App, field, key string) (*core.Record, error) {
	if key == "" {
		return nil, errors.New("empty key")
	}
	return app.FindFirstRecordByFilter("daily_digests", field+" = {:key}", dbx.Params{"key": key})
}

func dailyNewsWindowKey(userID, localDate string, start, end time.Time) string {
	return fmt.Sprintf("%s|%s|%s|%s", userID, localDate, start.UTC().Truncate(time.Second).Format(time.RFC3339), end.UTC().Truncate(time.Second).Format(time.RFC3339))
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Truncate(time.Second)
}

func sanitizeDailyNewsError(message string) string {
	if message == "OpenRouter API key is not configured." {
		return "OpenRouter API key is not configured. Configure it in Settings before generating Daily News."
	}
	return "Digest generation failed. Please try again."
}

func atoi2(s string) int {
	return int(s[0]-'0')*10 + int(s[1]-'0')
}
