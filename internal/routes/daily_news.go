package routes

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jgordijn/knowledgehub/internal/engine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type DailyNewsDigestDTO struct {
	ID                    string   `json:"id"`
	User                  string   `json:"user"`
	Status                string   `json:"status"`
	Trigger               string   `json:"trigger"`
	LocalDate             string   `json:"local_date"`
	Title                 string   `json:"title,omitempty"`
	BodyMarkdown          string   `json:"body_markdown,omitempty"`
	ReferencedIDs         []string `json:"referenced_entry_ids,omitempty"`
	CandidateCount        int      `json:"candidate_count"`
	IncludedCount         int      `json:"included_count"`
	UsedSubset            bool     `json:"used_subset"`
	ErrorMessage          string   `json:"error_message,omitempty"`
	GeneratedAt           string   `json:"generated_at,omitempty"`
	LastSuccessAt         string   `json:"last_success_at,omitempty"`
	HasSuccessfulSnapshot bool     `json:"has_successful_snapshot"`
	AttemptFinishedAt     string   `json:"attempt_finished_at,omitempty"`
	QueuedAt              string   `json:"queued_at,omitempty"`
	StartedAt             string   `json:"started_at,omitempty"`
	HeartbeatAt           string   `json:"heartbeat_at,omitempty"`
	PeriodStart           string   `json:"period_start,omitempty"`
	PeriodEnd             string   `json:"period_end,omitempty"`
}

type DailyNewsDigestListDTO struct {
	Latest   *DailyNewsDigestDTO  `json:"latest"`
	Selected *DailyNewsDigestDTO  `json:"selected"`
	Archive  []DailyNewsDigestDTO `json:"archive"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
	HasMore  bool                 `json:"has_more"`
}

type DailyNewsSettingsDTO struct {
	ID                string `json:"id"`
	User              string `json:"user"`
	Enabled           bool   `json:"enabled"`
	GenerationTime    string `json:"generation_time"`
	Timezone          string `json:"timezone"`
	ExtraInstructions string `json:"extra_instructions"`
}

type DailyNewsSettingsInput struct {
	Enabled           bool   `json:"enabled"`
	GenerationTime    string `json:"generation_time"`
	Timezone          string `json:"timezone"`
	ExtraInstructions string `json:"extra_instructions"`
}

type DailyNewsEntryReferenceDTO struct {
	Available bool                   `json:"available"`
	Message   string                 `json:"message,omitempty"`
	Entry     *DailyNewsEntryCardDTO `json:"entry,omitempty"`
}

var wakeDailyNewsWorker = func(app core.App, now time.Time) {
	time.AfterFunc(10*time.Millisecond, func() {
		defer func() { _ = recover() }()
		_, _ = engine.ProcessPendingDailyNewsJobs(app, now)
	})
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
	se.Router.GET("/api/daily-news/settings", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		status, dto, err := HandleDailyNewsGetSettings(re.App, re.Auth.Id)
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
	se.Router.PUT("/api/daily-news/settings", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		var input DailyNewsSettingsInput
		if err := re.BindBody(&input); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid settings payload."})
		}
		status, dto, err := HandleDailyNewsSaveSettings(re.App, re.Auth.Id, input)
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, dto)
	})
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
	se.Router.GET("/api/daily-news/digests/{id}", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		status, dto, err := HandleDailyNewsGetDigest(re.App, re.Auth.Id, re.Request.PathValue("id"))
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

func HandleDailyNewsGetSettings(app core.App, userID string) (int, DailyNewsSettingsDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsSettingsDTO{}, errors.New("Authentication required.")
	}
	settings, err := getOrCreateDailyNewsSettingsForUser(app, userID)
	if err != nil {
		return http.StatusInternalServerError, DailyNewsSettingsDTO{}, err
	}
	return http.StatusOK, dailyNewsSettingsDTO(settings), nil
}

func HandleDailyNewsSaveSettings(app core.App, userID string, input DailyNewsSettingsInput) (int, DailyNewsSettingsDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsSettingsDTO{}, errors.New("Authentication required.")
	}
	if err := validateDailyNewsSettingsInput(input); err != nil {
		return http.StatusBadRequest, DailyNewsSettingsDTO{}, err
	}
	settings, err := getOrCreateDailyNewsSettingsForUser(app, userID)
	if err != nil {
		return http.StatusInternalServerError, DailyNewsSettingsDTO{}, err
	}
	settings.Set("enabled", input.Enabled)
	settings.Set("generation_time", input.GenerationTime)
	settings.Set("timezone", input.Timezone)
	settings.Set("extra_instructions", input.ExtraInstructions)
	if err := app.Save(settings); err != nil {
		return http.StatusInternalServerError, DailyNewsSettingsDTO{}, err
	}
	return http.StatusOK, dailyNewsSettingsDTO(settings), nil
}

func HandleDailyNewsGetDigest(app core.App, userID, digestID string) (int, DailyNewsDigestDTO, error) {
	if userID == "" {
		return http.StatusUnauthorized, DailyNewsDigestDTO{}, errors.New("Authentication required.")
	}
	digest, err := app.FindRecordById("daily_digests", digestID)
	if err != nil || digest.GetString("user") != userID {
		return http.StatusNotFound, DailyNewsDigestDTO{}, errors.New("Digest not found.")
	}
	return http.StatusOK, dailyNewsDigestDTO(digest), nil
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

func dailyNewsDigestListRecency(record *core.Record) time.Time {
	for _, field := range []string{"last_success_at", "attempt_finished_at", "queued_at", "created"} {
		value := record.GetDateTime(field).Time()
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func sortDailyNewsDigestList(records []*core.Record) {
	sort.SliceStable(records, func(i, j int) bool {
		leftPeriodEnd := records[i].GetDateTime("period_end").Time()
		rightPeriodEnd := records[j].GetDateTime("period_end").Time()
		if !leftPeriodEnd.Equal(rightPeriodEnd) {
			return leftPeriodEnd.After(rightPeriodEnd)
		}
		leftRecency := dailyNewsDigestListRecency(records[i])
		rightRecency := dailyNewsDigestListRecency(records[j])
		if !leftRecency.Equal(rightRecency) {
			return leftRecency.After(rightRecency)
		}
		return records[i].Id > records[j].Id
	})
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
	userRecords, err := app.FindRecordsByFilter("daily_digests", "user = {:user}", "-period_end", 0, 0, dbx.Params{"user": userID})
	if err != nil || len(userRecords) == 0 {
		return http.StatusOK, DailyNewsDigestListDTO{Archive: []DailyNewsDigestDTO{}, Limit: limit, Offset: offset}, nil
	}
	sortDailyNewsDigestList(userRecords)
	latest := userRecords[0]
	selected := latest
	if selectedID != "" && selectedID != latest.Id {
		candidate, err := app.FindRecordById("daily_digests", selectedID)
		if err != nil || candidate.GetString("user") != userID {
			return http.StatusNotFound, DailyNewsDigestListDTO{}, errors.New("Digest not found.")
		}
		selected = candidate
	}
	records := make([]*core.Record, 0, len(userRecords)-1)
	for _, record := range userRecords {
		if record.Id != latest.Id {
			records = append(records, record)
		}
	}
	if offset > len(records) {
		offset = len(records)
	}
	end := offset + limit
	hasMore := end < len(records)
	if end > len(records) {
		end = len(records)
	}
	records = records[offset:end]
	archive := make([]DailyNewsDigestDTO, 0, len(records))
	for _, record := range records {
		archive = append(archive, dailyNewsDigestDTO(record))
	}
	latestDTO := dailyNewsDigestDTO(latest)
	selectedDTO := dailyNewsDigestDTO(selected)
	return http.StatusOK, DailyNewsDigestListDTO{Latest: &latestDTO, Selected: &selectedDTO, Archive: archive, Limit: limit, Offset: offset, HasMore: hasMore}, nil
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
		if active, findErr := findActiveManualDigest(app, userID, localDate); findErr == nil {
			return http.StatusAccepted, dailyNewsDigestDTO(active), nil
		}
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
	if created {
		wakeDailyNewsWorker(app, now)
		return http.StatusAccepted, dailyNewsDigestDTO(digest), nil
	}
	if digest.GetString("status") == "pending" || digest.GetString("status") == "running" {
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
	} else {
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
	wakeDailyNewsWorker(app, now)
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

func findActiveManualDigest(app core.App, userID, localDate string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("daily_digests", "user = {:user} && local_date = {:date} && trigger = 'manual' && (status = 'pending' || status = 'running')", dbx.Params{"user": userID, "date": localDate})
}

func dailyNewsSettingsDTO(record *core.Record) DailyNewsSettingsDTO {
	return DailyNewsSettingsDTO{
		ID:                record.Id,
		User:              record.GetString("user"),
		Enabled:           record.GetBool("enabled"),
		GenerationTime:    record.GetString("generation_time"),
		Timezone:          record.GetString("timezone"),
		ExtraInstructions: record.GetString("extra_instructions"),
	}
}

func validateDailyNewsSettingsInput(input DailyNewsSettingsInput) error {
	if err := engine.ValidateDailyNewsScheduleSettings(engine.DailyNewsScheduleSettings{Enabled: input.Enabled, GenerationTime: input.GenerationTime, Timezone: input.Timezone}); err != nil {
		return err
	}
	if utf8.RuneCountInString(input.ExtraInstructions) > 2000 {
		return errors.New("Extra instructions must be 2000 characters or fewer.")
	}
	for _, r := range input.ExtraInstructions {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return errors.New("Extra instructions contain unsupported control characters.")
		}
	}
	return nil
}

func dailyNewsDigestDTO(record *core.Record) DailyNewsDigestDTO {
	return DailyNewsDigestDTO{
		ID:                    record.Id,
		User:                  record.GetString("user"),
		Status:                record.GetString("status"),
		Trigger:               record.GetString("trigger"),
		LocalDate:             record.GetString("local_date"),
		Title:                 record.GetString("title"),
		BodyMarkdown:          record.GetString("body_markdown"),
		ReferencedIDs:         record.GetStringSlice("referenced_entry_ids"),
		CandidateCount:        int(record.GetFloat("candidate_count")),
		IncludedCount:         int(record.GetFloat("included_count")),
		UsedSubset:            record.GetBool("used_subset"),
		ErrorMessage:          record.GetString("error_message"),
		GeneratedAt:           record.GetDateTime("last_success_at").String(),
		LastSuccessAt:         record.GetDateTime("last_success_at").String(),
		HasSuccessfulSnapshot: record.GetBool("has_successful_snapshot"),
		AttemptFinishedAt:     record.GetDateTime("attempt_finished_at").String(),
		QueuedAt:              record.GetDateTime("queued_at").String(),
		StartedAt:             record.GetDateTime("started_at").String(),
		HeartbeatAt:           record.GetDateTime("heartbeat_at").String(),
		PeriodStart:           record.GetDateTime("period_start").String(),
		PeriodEnd:             record.GetDateTime("period_end").String(),
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
