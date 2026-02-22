package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCounters_Snapshot(t *testing.T) {
	c := &Counters{}
	c.EventsIngested.Add(10)
	c.EventsProcessed.Add(8)
	c.EventsEnriched.Add(7)
	c.AlertsTriggered.Add(3)
	c.ErrorsTotal.Add(2)
	c.RetryAttempts.Add(5)

	snap := c.Snapshot()

	expected := map[string]int64{
		"events_ingested":  10,
		"events_processed": 8,
		"events_enriched":  7,
		"alerts_triggered": 3,
		"errors_total":     2,
		"retry_attempts":   5,
	}

	for k, want := range expected {
		if got := snap[k]; got != want {
			t.Errorf("%s: got %d, want %d", k, got, want)
		}
	}
}

func TestCounters_Handler(t *testing.T) {
	c := &Counters{}
	c.EventsIngested.Add(42)
	c.AlertsTriggered.Add(7)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	c.Handler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var body map[string]int64
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["events_ingested"] != 42 {
		t.Errorf("events_ingested: got %d, want 42", body["events_ingested"])
	}
	if body["alerts_triggered"] != 7 {
		t.Errorf("alerts_triggered: got %d, want 7", body["alerts_triggered"])
	}
}

func TestCounters_Concurrent(t *testing.T) {
	c := &Counters{}

	done := make(chan struct{})
	for range 100 {
		go func() {
			c.EventsIngested.Add(1)
			c.ErrorsTotal.Add(1)
			done <- struct{}{}
		}()
	}
	for range 100 {
		<-done
	}

	if c.EventsIngested.Load() != 100 {
		t.Errorf("EventsIngested: got %d, want 100", c.EventsIngested.Load())
	}
	if c.ErrorsTotal.Load() != 100 {
		t.Errorf("ErrorsTotal: got %d, want 100", c.ErrorsTotal.Load())
	}
}
