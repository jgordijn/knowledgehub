package engine

import (
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestRecordFailure(t *testing.T) {
	tests := []struct {
		name             string
		initialFailures  int
		initialStatus    string
		expectedFailures int
		expectedStatus   string
	}{
		{
			name:             "first failure transitions healthy to failing",
			initialFailures:  0,
			initialStatus:    StatusHealthy,
			expectedFailures: 1,
			expectedStatus:   StatusFailing,
		},
		{
			name:             "second failure stays failing",
			initialFailures:  1,
			initialStatus:    StatusFailing,
			expectedFailures: 2,
			expectedStatus:   StatusFailing,
		},
		{
			name:             "fourth failure stays failing",
			initialFailures:  3,
			initialStatus:    StatusFailing,
			expectedFailures: 4,
			expectedStatus:   StatusFailing,
		},
		{
			name:             "fifth failure transitions to quarantined",
			initialFailures:  4,
			initialStatus:    StatusFailing,
			expectedFailures: 5,
			expectedStatus:   StatusQuarantined,
		},
		{
			name:             "already quarantined stays quarantined",
			initialFailures:  5,
			initialStatus:    StatusQuarantined,
			expectedFailures: 6,
			expectedStatus:   StatusQuarantined,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", tt.initialStatus, tt.initialFailures, true)

			err := RecordFailure(app, resource, "test error")
			if err != nil {
				t.Fatalf("RecordFailure returned error: %v", err)
			}

			updated, err := app.FindRecordById("resources", resource.Id)
			if err != nil {
				t.Fatalf("failed to reload resource: %v", err)
			}

			if got := updated.GetInt("consecutive_failures"); got != tt.expectedFailures {
				t.Errorf("consecutive_failures = %d, want %d", got, tt.expectedFailures)
			}
			if got := updated.GetString("status"); got != tt.expectedStatus {
				t.Errorf("status = %q, want %q", got, tt.expectedStatus)
			}
			if got := updated.GetString("last_error"); got != "test error" {
				t.Errorf("last_error = %q, want %q", got, "test error")
			}

			if tt.expectedStatus == StatusQuarantined {
				if got := updated.GetString("quarantined_at"); got == "" {
					t.Error("quarantined_at should be set")
				}
			}
		})
	}
}

func TestRecordSuccess(t *testing.T) {
	tests := []struct {
		name            string
		initialFailures int
		initialStatus   string
	}{
		{
			name:            "resets from failing state",
			initialFailures: 3,
			initialStatus:   StatusFailing,
		},
		{
			name:            "resets from healthy state",
			initialFailures: 0,
			initialStatus:   StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cleanup := testutil.NewTestApp(t)
			defer cleanup()

			resource := testutil.CreateResource(t, app, "test", "https://example.com/feed", "rss", tt.initialStatus, tt.initialFailures, true)

			err := RecordSuccess(app, resource)
			if err != nil {
				t.Fatalf("RecordSuccess returned error: %v", err)
			}

			updated, err := app.FindRecordById("resources", resource.Id)
			if err != nil {
				t.Fatalf("failed to reload resource: %v", err)
			}

			if got := updated.GetInt("consecutive_failures"); got != 0 {
				t.Errorf("consecutive_failures = %d, want 0", got)
			}
			if got := updated.GetString("status"); got != StatusHealthy {
				t.Errorf("status = %q, want %q", got, StatusHealthy)
			}
			if got := updated.GetString("last_error"); got != "" {
				t.Errorf("last_error = %q, want empty", got)
			}
			if got := updated.GetString("last_checked"); got == "" {
				t.Error("last_checked should be set")
			}
		})
	}
}
