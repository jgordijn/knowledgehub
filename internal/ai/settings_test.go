package ai

import (
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, app interface{ Save(any) error })
		wantErr bool
		wantKey string
	}{
		{
			name:    "no key configured",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			key, err := GetAPIKey(app)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestGetAPIKey_WithKey(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_api_key", "sk-test-12345")

	key, err := GetAPIKey(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "sk-test-12345" {
		t.Errorf("key = %q, want %q", key, "sk-test-12345")
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name      string
		setValue  string
		wantModel string
	}{
		{
			name:      "returns default when not set",
			wantModel: DefaultModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			model := GetModel(app)
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
		})
	}
}

func TestGetModel_CustomModel(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_model", "anthropic/claude-3.5-sonnet")

	model := GetModel(app)
	if model != "anthropic/claude-3.5-sonnet" {
		t.Errorf("model = %q, want %q", model, "anthropic/claude-3.5-sonnet")
	}
}

func TestGetModel_EmptyValue(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	testutil.CreateSetting(t, app, "openrouter_model", "")

	model := GetModel(app)
	if model != DefaultModel {
		t.Errorf("model = %q, want default %q", model, DefaultModel)
	}
}
