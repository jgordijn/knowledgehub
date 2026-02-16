package engine

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const defaultInterval = 30 * time.Minute

// Scheduler periodically fetches new content for active resources.
type Scheduler struct {
	app      core.App
	interval time.Duration
	stopCh   chan struct{}
}

// NewScheduler creates a new Scheduler with the default 30-minute interval.
func NewScheduler(app core.App) *Scheduler {
	return &Scheduler{
		app:      app,
		interval: defaultInterval,
		stopCh:   make(chan struct{}),
	}
}

// NewSchedulerWithInterval creates a Scheduler with a custom interval (for testing).
func NewSchedulerWithInterval(app core.App, interval time.Duration) *Scheduler {
	return &Scheduler{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the scheduling loop. It runs an initial fetch, then ticks
// at the configured interval. Blocks until Stop is called.
func (s *Scheduler) Start() {
	log.Printf("Scheduler started with %v interval", s.interval)

	// Run immediately on start
	s.fetchAll()

	// Also retry previously failed entries
	s.retryFailedEntries()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.fetchAll()
			s.retryFailedEntries()
		case <-s.stopCh:
			log.Println("Scheduler stopped")
			return
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) fetchAll() {
	resources, err := s.app.FindRecordsByFilter(
		"resources",
		"active = true && status != 'quarantined'",
		"",
		0, 0,
		nil,
	)
	if err != nil {
		log.Printf("Scheduler: failed to load resources: %v", err)
		return
	}

	log.Printf("Scheduler: processing %d active resources", len(resources))

	for _, resource := range resources {
		if err := FetchResource(s.app, resource, DefaultHTTPClient); err != nil {
			log.Printf("Scheduler: fetch failed for resource %s (%s): %v",
				resource.GetString("name"), resource.Id, err)
			if qErr := RecordFailure(s.app, resource, err.Error()); qErr != nil {
				log.Printf("Scheduler: failed to record failure: %v", qErr)
			}
		} else {
			if sErr := RecordSuccess(s.app, resource); sErr != nil {
				log.Printf("Scheduler: failed to record success: %v", sErr)
			}
		}
	}
}

func (s *Scheduler) retryFailedEntries() {
	entries, err := s.app.FindRecordsByFilter(
		"entries",
		"processing_status = 'failed' || processing_status = 'pending'",
		"-created",
		50, 0,
		nil,
	)
	if err != nil {
		log.Printf("Scheduler: failed to load pending entries: %v", err)
		return
	}

	for _, entry := range entries {
		go processEntry(s.app, entry)
	}
}
