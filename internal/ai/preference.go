package ai

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const regenerateThreshold = 20

// CheckAndRegeneratePreferences counts corrections since last profile
// generation and regenerates the profile if the threshold is met.
func CheckAndRegeneratePreferences(app core.App) {
	count, err := countCorrectionsSinceLastProfile(app)
	if err != nil {
		log.Printf("Preference check failed: %v", err)
		return
	}

	if count >= regenerateThreshold {
		log.Printf("Regenerating preference profile (%d corrections since last generation)", count)
		if err := GeneratePreferenceProfile(app); err != nil {
			log.Printf("Failed to regenerate preferences: %v", err)
		}
	}
}

// GeneratePreferenceProfile collects all entries where user_stars differs from
// ai_stars and sends them to the LLM to generate a preference profile.
func GeneratePreferenceProfile(app core.App) error {
	apiKey, err := GetAPIKey(app)
	if err != nil {
		return fmt.Errorf("no API key configured: %w", err)
	}
	model := GetModel(app)

	corrections, err := app.FindRecordsByFilter(
		"entries",
		"user_stars > 0 && ai_stars > 0 && user_stars != ai_stars",
		"-created",
		100, 0,
		nil,
	)
	if err != nil || len(corrections) == 0 {
		return fmt.Errorf("no corrections found")
	}

	prompt := buildPreferencePrompt(corrections)

	response, err := clientCompleteFunc(apiKey, model, []Message{
		{Role: "system", Content: "You are a helpful assistant that analyzes reading preferences. Be concise and specific."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return fmt.Errorf("AI completion failed: %w", err)
	}

	return savePreferenceProfile(app, strings.TrimSpace(response))
}

func buildPreferencePrompt(corrections []*core.Record) string {
	var sb strings.Builder
	sb.WriteString("Based on the following rating corrections, generate a brief preference profile describing what topics and content the user values highly vs. finds less interesting.\n\n")
	sb.WriteString("Rating corrections (AI rating → User rating):\n")

	for _, r := range corrections {
		sb.WriteString(fmt.Sprintf("- \"%s\" (summary: %s): AI=%d, User=%d\n",
			r.GetString("title"),
			truncateText(r.GetString("summary"), 100),
			r.GetInt("ai_stars"),
			r.GetInt("user_stars"),
		))
	}

	sb.WriteString("\nGenerate a concise preference profile (3-5 paragraphs) that can guide future article scoring.")
	return sb.String()
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func savePreferenceProfile(app core.App, profileText string) error {
	// Try to find existing preference record
	records, err := app.FindRecordsByFilter("preferences", "1=1", "-generated_at", 1, 0, nil)
	if err == nil && len(records) > 0 {
		records[0].Set("profile_text", profileText)
		records[0].Set("generated_at", time.Now().UTC().Format(time.RFC3339))
		return app.Save(records[0])
	}

	// Create new preference record
	collection, err := app.FindCollectionByNameOrId("preferences")
	if err != nil {
		return fmt.Errorf("preferences collection not found: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("profile_text", profileText)
	record.Set("generated_at", time.Now().UTC().Format(time.RFC3339))
	return app.Save(record)
}

func countCorrectionsSinceLastProfile(app core.App) (int, error) {
	// Get last profile generation time
	profiles, err := app.FindRecordsByFilter("preferences", "1=1", "-generated_at", 1, 0, nil)

	var filter string
	params := map[string]any{}

	if err != nil || len(profiles) == 0 {
		// No profile ever generated — count all corrections
		filter = "user_stars > 0 && ai_stars > 0 && user_stars != ai_stars"
	} else {
		generatedAt := profiles[0].GetString("generated_at")
		filter = "user_stars > 0 && ai_stars > 0 && user_stars != ai_stars && created > {:since}"
		params["since"] = generatedAt
	}

	records, err := app.FindRecordsByFilter("entries", filter, "", 0, 0, params)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}
