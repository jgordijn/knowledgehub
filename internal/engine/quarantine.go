package engine

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	StatusHealthy      = "healthy"
	StatusFailing      = "failing"
	StatusQuarantined  = "quarantined"
	QuarantineThreshold = 5
)

// RecordFailure increments a resource's consecutive_failures counter and
// transitions its status through the state machine:
//
//	healthy → failing (at 1 failure)
//	failing → quarantined (at QuarantineThreshold failures)
func RecordFailure(app core.App, record *core.Record, errMsg string) error {
	failures := record.GetInt("consecutive_failures") + 1
	record.Set("consecutive_failures", failures)
	record.Set("last_error", errMsg)

	switch {
	case failures >= QuarantineThreshold:
		record.Set("status", StatusQuarantined)
		record.Set("quarantined_at", time.Now().UTC().Format(time.RFC3339))
	case failures >= 1:
		record.Set("status", StatusFailing)
	}

	return app.Save(record)
}

// RecordSuccess resets a resource's failure state back to healthy.
func RecordSuccess(app core.App, record *core.Record) error {
	record.Set("consecutive_failures", 0)
	record.Set("status", StatusHealthy)
	record.Set("last_error", "")
	record.Set("last_checked", time.Now().UTC().Format(time.RFC3339))
	return app.Save(record)
}
