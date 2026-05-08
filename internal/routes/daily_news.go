package routes

import (
	"errors"
	"net/http"
	"time"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type DailyNewsDigestDTO struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	Status    string `json:"status"`
	Trigger   string `json:"trigger"`
	LocalDate string `json:"local_date"`
}

func RegisterDailyNewsRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/daily-news/generate", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		status, dto, err := HandleDailyNewsGenerateNow(re.App, re.Auth.Id, time.Now())
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
}

func HandleDailyNewsGenerateNow(app core.App, userID string, now time.Time) (int, DailyNewsDigestDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsDigestDTO{}, errors.New("Authentication required.")
	}
	settings, err := getOrCreateDailyNewsSettingsForUser(app, userID)
	if err != nil {
		return http.StatusInternalServerError, DailyNewsDigestDTO{}, err
	}
	schedule := engine.DailyNewsScheduleSettings{
		Enabled:        settings.GetBool("enabled"),
		GenerationTime: settings.GetString("generation_time"),
		Timezone:       settings.GetString("timezone"),
	}
	if err := engine.ValidateDailyNewsScheduleSettings(schedule); err != nil {
		return http.StatusBadRequest, DailyNewsDigestDTO{}, err
	}
	due, localDate, scheduledEnd, err := engine.IsDailyNewsDue(schedule, now)
	if err != nil {
		return http.StatusBadRequest, DailyNewsDigestDTO{}, err
	}
	periodEnd := scheduledEnd
	trigger := "automatic"
	scheduled := true
	if !due {
		loc, _ := time.LoadLocation(schedule.Timezone)
		localDate = now.In(loc).Format("2006-01-02")
		periodEnd = now.UTC().Truncate(time.Second)
		trigger = "manual"
		scheduled = false
	}
	window, _, err := engine.FindDailyNewsCandidates(app, userID, periodEnd)
	if err != nil {
		return http.StatusInternalServerError, DailyNewsDigestDTO{}, err
	}
	digest, created, err := engine.ClaimDailyNewsJob(app, engine.DailyNewsJobClaim{
		UserID:      userID,
		LocalDate:   localDate,
		PeriodStart: window.Start,
		PeriodEnd:   window.End,
		Trigger:     trigger,
		Scheduled:   scheduled,
		Now:         now,
	})
	if err != nil {
		if existing, findErr := findSuccessfulScheduledDigest(app, userID, localDate); findErr == nil {
			return http.StatusOK, dailyNewsDigestDTO(existing), nil
		}
		return http.StatusInternalServerError, DailyNewsDigestDTO{}, err
	}
	if created || digest.GetString("status") == "pending" || digest.GetString("status") == "running" {
		return http.StatusAccepted, dailyNewsDigestDTO(digest), nil
	}
	return http.StatusOK, dailyNewsDigestDTO(digest), nil
}

func getOrCreateDailyNewsSettingsForUser(app core.App, userID string) (*core.Record, error) {
	existing, err := app.FindFirstRecordByFilter("daily_news_settings", "user = {:user}", dbx.Params{"user": userID})
	if err == nil {
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

func findSuccessfulScheduledDigest(app core.App, userID, localDate string) (*core.Record, error) {
	key := userID + "|" + localDate
	return app.FindFirstRecordByFilter("daily_digests", "user = {:user} && local_date = {:date} && successful_scheduled_day_key = {:key}", dbx.Params{"user": userID, "date": localDate, "key": key})
}

func dailyNewsDigestDTO(record *core.Record) DailyNewsDigestDTO {
	return DailyNewsDigestDTO{ID: record.Id, User: record.GetString("user"), Status: record.GetString("status"), Trigger: record.GetString("trigger"), LocalDate: record.GetString("local_date")}
}
