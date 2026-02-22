package metrics

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Counters holds application-wide metric counters using lock-free atomics.
type Counters struct {
	EventsIngested  atomic.Int64
	EventsProcessed atomic.Int64
	EventsEnriched  atomic.Int64
	AlertsTriggered atomic.Int64
	ErrorsTotal     atomic.Int64
	RetryAttempts   atomic.Int64
}

// Global is the singleton metrics instance.
var Global = &Counters{}

// Snapshot returns all counter values as a map.
func (c *Counters) Snapshot() map[string]int64 {
	return map[string]int64{
		"events_ingested":  c.EventsIngested.Load(),
		"events_processed": c.EventsProcessed.Load(),
		"events_enriched":  c.EventsEnriched.Load(),
		"alerts_triggered": c.AlertsTriggered.Load(),
		"errors_total":     c.ErrorsTotal.Load(),
		"retry_attempts":   c.RetryAttempts.Load(),
	}
}

// Handler returns an HTTP handler that serves metrics as JSON.
func (c *Counters) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c.Snapshot())
	}
}
