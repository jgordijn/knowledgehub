package ai

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

const (
	SettingAPIKey = "openrouter_api_key"
	SettingModel  = "openrouter_model"
	DefaultModel  = "openai/gpt-4o-mini"
)

// GetAPIKey reads the OpenRouter API key from app_settings.
func GetAPIKey(app core.App) (string, error) {
	return getSetting(app, SettingAPIKey)
}

// GetModel reads the model name from app_settings, returning DefaultModel if unset.
func GetModel(app core.App) string {
	model, err := getSetting(app, SettingModel)
	if err != nil || model == "" {
		return DefaultModel
	}
	return model
}

func getSetting(app core.App, key string) (string, error) {
	record, err := app.FindFirstRecordByFilter("app_settings", "key = {:key}", map[string]any{"key": key})
	if err != nil {
		return "", fmt.Errorf("setting %q not found: %w", key, err)
	}
	return record.GetString("value"), nil
}
