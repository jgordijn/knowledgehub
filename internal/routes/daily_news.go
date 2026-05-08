package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type DailyNewsDigestDTO struct {
	ID             string   `json:"id"`
	User           string   `json:"user"`
	Status         string   `json:"status"`
	Trigger        string   `json:"trigger"`
	LocalDate      string   `json:"local_date"`
	Title          string   `json:"title,omitempty"`
	BodyMarkdown   string   `json:"body_markdown,omitempty"`
	ReferencedIDs  []string `json:"referenced_entry_ids,omitempty"`
	CandidateCount int      `json:"candidate_count"`
	IncludedCount  int      `json:"included_count"`
	UsedSubset     bool     `json:"used_subset"`
	ErrorMessage   string   `json:"error_message,omitempty"`
	GeneratedAt    string   `json:"generated_at,omitempty"`
	PeriodStart    string   `json:"period_start,omitempty"`
	PeriodEnd      string   `json:"period_end,omitempty"`
}

type DailyNewsDigestListDTO struct {
	Latest   DailyNewsDigestDTO   `json:"latest"`
	Selected DailyNewsDigestDTO   `json:"selected"`
	Archive  []DailyNewsDigestDTO `json:"archive"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
	HasMore  bool                 `json:"has_more"`
}

type DailyNewsEntryReferenceDTO struct {
	Available bool                   `json:"available"`
	Message   string                 `json:"message,omitempty"`
	Entry     *DailyNewsEntryCardDTO `json:"entry,omitempty"`
}

type DailyNewsEntryCardDTO struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Summary        string   `json:"summary,omitempty"`
	Takeaways      []string `json:"takeaways,omitempty"`
	EffectiveStars int      `json:"effective_stars"`
	SourceName     string   `json:"source_name,omitempty"`
	PublishedAt    string   `json:"published_at,omitempty"`
	DiscoveredAt   string   `json:"discovered_at,omitempty"`
}

func RegisterDailyNewsRoutes(se *core.ServeEvent) {
	se.Router.GET("/api/daily-news/digests", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		limit, _ := strconv.Atoi(re.Request.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(re.Request.URL.Query().Get("offset"))
		status, dto, err := HandleDailyNewsListDigests(re.App, re.Auth.Id, re.Request.URL.Query().Get("selected"), limit, offset)
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
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
	se.Router.POST("/api/daily-news/digests/{id}/regenerate", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		status, dto, err := HandleDailyNewsRegenerate(re.App, re.Auth.Id, re.Request.PathValue("id"), time.Now())
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
	se.Router.GET("/api/daily-news/digests/{digestId}/entries/{entryId}", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		status, dto, err := HandleDailyNewsEntryReference(re.App, re.Auth.Id, re.Request.PathValue("digestId"), re.Request.PathValue("entryId"))
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
}

func HandleDailyNewsEntryReference(app core.App, userID, digestID, entryID string) (int, DailyNewsEntryReferenceDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsEntryReferenceDTO{}, errors.New("Authentication required.")
	}
	digest, err := app.FindRecordById("daily_digests", digestID)
	if err != nil || digest.GetString("user") != userID {
		return http.StatusNotFound, DailyNewsEntryReferenceDTO{}, errors.New("Entry reference not found.")
	}
	if !containsString(digest.GetStringSlice("referenced_entry_ids"), entryID) {
		return http.StatusNotFound, DailyNewsEntryReferenceDTO{}, errors.New("Entry reference not found.")
	}
	entry, err := app.FindRecordById("entries", entryID)
	if err != nil {
		return http.StatusOK, DailyNewsEntryReferenceDTO{Available: false, Message: "Referenced entry is no longer available."}, nil
	}
	return http.StatusOK, DailyNewsEntryReferenceDTO{Available: true, Entry: dailyNewsEntryCardDTO(app, entry)}, nil
}

func HandleDailyNewsListDigests(app core.App, userID, selectedID string, limit, offset int) (int, DailyNewsDigestListDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsDigestListDTO{}, errors.New("Authentication required.")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	latestRecords, err := app.FindRecordsByFilter("daily_digests", "user = {:user}", "-period_end", 1, 0, dbx.Params{"user": userID})
	if err != nil || len(latestRecords) == 0 {
		return http.StatusOK, DailyNewsDigestListDTO{Archive: []DailyNewsDigestDTO{}, Limit: limit, Offset: offset}, nil
	}
	latest := latestRecords[0]
	selected := latest
	if selectedID != "" && selectedID != latest.Id {
		candidate, err := app.FindRecordById("daily_digests", selectedID)
		if err != nil || candidate.GetString("user") != userID {
			return http.StatusNotFound, DailyNewsDigestListDTO{}, errors.New("Digest not found.")
		}
		selected = candidate
	}
	records, err := app.FindRecordsByFilter("daily_digests", "user = {:user} && id != {:latest}", "-period_end", limit+1, offset, dbx.Params{"user": userID, "latest": latest.Id})
	if err != nil {
		return http.StatusInternalServerError, DailyNewsDigestListDTO{}, err
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	archive := make([]DailyNewsDigestDTO, 0, len(records))
	for _, record := range records {
		archive = append(archive, dailyNewsDigestDTO(record))
	}
	return http.StatusOK, DailyNewsDigestListDTO{Latest: dailyNewsDigestDTO(latest), Selected: dailyNewsDigestDTO(selected), Archive: archive, Limit: limit, Offset: offset, HasMore: hasMore}, nil
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

func HandleDailyNewsRegenerate(app core.App, userID, digestID string, now time.Time) (int, DailyNewsDigestDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsDigestDTO{}, errors.New("Authentication required.")
	}
	digest, err := app.FindRecordById("daily_digests", digestID)
	if err != nil || digest.GetString("user") != userID {
		return http.StatusNotFound, DailyNewsDigestDTO{}, errors.New("Digest not found.")
	}
	if digest.GetString("status") == "pending" || digest.GetString("status") == "running" {
		return http.StatusAccepted, dailyNewsDigestDTO(digest), nil
	}
	periodStart := digest.GetDateTime("period_start").Time().UTC().Truncate(time.Second)
	periodEnd := digest.GetDateTime("period_end").Time().UTC().Truncate(time.Second)
	windowKey := userID + "|" + digest.GetString("local_date") + "|" + periodStart.Format(time.RFC3339) + "|" + periodEnd.Format(time.RFC3339)
	if active, err := app.FindFirstRecordByFilter("daily_digests", "user = {:user} && local_date = {:date} && (status = 'pending' || status = 'running')", dbx.Params{"user": userID, "date": digest.GetString("local_date")}); err == nil && active.Id != digest.Id {
		return http.StatusAccepted, dailyNewsDigestDTO(active), nil
	}
	digest.Set("status", "pending")
	digest.Set("queued_at", now.UTC().Truncate(time.Second).Format(time.RFC3339))
	digest.Set("started_at", "")
	digest.Set("heartbeat_at", "")
	digest.Set("attempt_finished_at", "")
	digest.Set("error_message", "")
	digest.Set("window_key", windowKey)
	digest.Set("active_window_key", windowKey)
	if key := digest.GetString("successful_scheduled_day_key"); key != "" {
		digest.Set("scheduled_day_key", key)
		digest.Set("active_scheduled_day_key", key)
	} else if strings.EqualFold(digest.GetString("trigger"), "automatic") {
		key := userID + "|" + digest.GetString("local_date")
		digest.Set("scheduled_day_key", key)
		digest.Set("active_scheduled_day_key", key)
	}
	if err := app.Save(digest); err != nil {
		if active, findErr := app.FindFirstRecordByFilter("daily_digests", "user = {:user} && local_date = {:date} && (status = 'pending' || status = 'running')", dbx.Params{"user": userID, "date": digest.GetString("local_date")}); findErr == nil {
			return http.StatusAccepted, dailyNewsDigestDTO(active), nil
		}
		return http.StatusInternalServerError, DailyNewsDigestDTO{}, err
	}
	return http.StatusAccepted, dailyNewsDigestDTO(digest), nil
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
	return DailyNewsDigestDTO{
		ID:             record.Id,
		User:           record.GetString("user"),
		Status:         record.GetString("status"),
		Trigger:        record.GetString("trigger"),
		LocalDate:      record.GetString("local_date"),
		Title:          record.GetString("title"),
		BodyMarkdown:   record.GetString("body_markdown"),
		ReferencedIDs:  record.GetStringSlice("referenced_entry_ids"),
		CandidateCount: int(record.GetFloat("candidate_count")),
		IncludedCount:  int(record.GetFloat("included_count")),
		UsedSubset:     record.GetBool("used_subset"),
		ErrorMessage:   record.GetString("error_message"),
		GeneratedAt:    record.GetDateTime("last_success_at").String(),
		PeriodStart:    record.GetDateTime("period_start").String(),
		PeriodEnd:      record.GetDateTime("period_end").String(),
	}
}

func dailyNewsEntryCardDTO(app core.App, entry *core.Record) *DailyNewsEntryCardDTO {
	effectiveStars := int(entry.GetFloat("ai_stars"))
	if userStars := int(entry.GetFloat("user_stars")); userStars > 0 {
		effectiveStars = userStars
	}
	dto := &DailyNewsEntryCardDTO{
		ID:             entry.Id,
		Title:          entry.GetString("title"),
		URL:            entry.GetString("url"),
		Summary:        entry.GetString("summary"),
		Takeaways:      entry.GetStringSlice("takeaways"),
		EffectiveStars: effectiveStars,
		PublishedAt:    entry.GetDateTime("published_at").String(),
		DiscoveredAt:   entry.GetDateTime("discovered_at").String(),
	}
	if resourceID := entry.GetString("resource"); resourceID != "" {
		if resource, err := app.FindRecordById("resources", resourceID); err == nil {
			dto.SourceName = resource.GetString("name")
		}
	}
	return dto
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
