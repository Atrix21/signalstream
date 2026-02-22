package alerter

import (
	"testing"

	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/platform"
)

func TestMatchesSimpleFilters(t *testing.T) {
	tests := []struct {
		name     string
		event    platform.NormalizedEvent
		strategy database.Strategy
		want     bool
	}{
		{
			name:     "no filters matches everything",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"AAPL"}},
			strategy: database.Strategy{Source: nil, Tickers: nil},
			want:     true,
		},
		{
			name:     "source match",
			event:    platform.NormalizedEvent{Source: "Polygon.io"},
			strategy: database.Strategy{Source: []string{"Polygon.io"}},
			want:     true,
		},
		{
			name:     "source mismatch",
			event:    platform.NormalizedEvent{Source: "SEC EDGAR"},
			strategy: database.Strategy{Source: []string{"Polygon.io"}},
			want:     false,
		},
		{
			name:     "source match one of many",
			event:    platform.NormalizedEvent{Source: "SEC EDGAR"},
			strategy: database.Strategy{Source: []string{"Polygon.io", "SEC EDGAR"}},
			want:     true,
		},
		{
			name:     "ticker match",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"AAPL"}},
			strategy: database.Strategy{Tickers: []string{"AAPL"}},
			want:     true,
		},
		{
			name:     "ticker mismatch",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"GOOG"}},
			strategy: database.Strategy{Tickers: []string{"AAPL"}},
			want:     false,
		},
		{
			name:     "ticker match one of many",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"TSLA", "AAPL"}},
			strategy: database.Strategy{Tickers: []string{"AAPL", "MSFT"}},
			want:     true,
		},
		{
			name:     "event has no tickers but strategy requires them",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: nil},
			strategy: database.Strategy{Tickers: []string{"AAPL"}},
			want:     false,
		},
		{
			name:     "both source and ticker must match",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"AAPL"}},
			strategy: database.Strategy{Source: []string{"Polygon.io"}, Tickers: []string{"AAPL"}},
			want:     true,
		},
		{
			name:     "source matches but ticker does not",
			event:    platform.NormalizedEvent{Source: "Polygon.io", Tickers: []string{"GOOG"}},
			strategy: database.Strategy{Source: []string{"Polygon.io"}, Tickers: []string{"AAPL"}},
			want:     false,
		},
		{
			name:     "ticker matches but source does not",
			event:    platform.NormalizedEvent{Source: "SEC EDGAR", Tickers: []string{"AAPL"}},
			strategy: database.Strategy{Source: []string{"Polygon.io"}, Tickers: []string{"AAPL"}},
			want:     false,
		},
		{
			name:     "empty source filter with ticker match",
			event:    platform.NormalizedEvent{Source: "SEC EDGAR", Tickers: []string{"AAPL"}},
			strategy: database.Strategy{Source: nil, Tickers: []string{"AAPL"}},
			want:     true,
		},
		{
			name:     "empty event tickers with no ticker filter",
			event:    platform.NormalizedEvent{Source: "SEC EDGAR", Tickers: nil},
			strategy: database.Strategy{Source: []string{"SEC EDGAR"}, Tickers: nil},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesSimpleFilters(tt.event, tt.strategy)
			if got != tt.want {
				t.Errorf("MatchesSimpleFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindEventMatch(t *testing.T) {
	tests := []struct {
		name      string
		eventUUID string
		results   []ScoredResult
		threshold float64
		wantScore float32
		wantMatch bool
	}{
		{
			name:      "match above threshold",
			eventUUID: "abc-123",
			results: []ScoredResult{
				{PointID: "other-1", Score: 0.95},
				{PointID: "abc-123", Score: 0.85},
			},
			threshold: 0.80,
			wantScore: 0.85,
			wantMatch: true,
		},
		{
			name:      "match at exact threshold",
			eventUUID: "abc-123",
			results: []ScoredResult{
				{PointID: "abc-123", Score: 0.80},
			},
			threshold: 0.80,
			wantScore: 0.80,
			wantMatch: true,
		},
		{
			name:      "match below threshold",
			eventUUID: "abc-123",
			results: []ScoredResult{
				{PointID: "abc-123", Score: 0.50},
			},
			threshold: 0.80,
			wantScore: 0,
			wantMatch: false,
		},
		{
			name:      "event not in results",
			eventUUID: "abc-123",
			results: []ScoredResult{
				{PointID: "other-1", Score: 0.95},
				{PointID: "other-2", Score: 0.90},
			},
			threshold: 0.50,
			wantScore: 0,
			wantMatch: false,
		},
		{
			name:      "empty results",
			eventUUID: "abc-123",
			results:   nil,
			threshold: 0.50,
			wantScore: 0,
			wantMatch: false,
		},
		{
			name:      "zero threshold",
			eventUUID: "abc-123",
			results: []ScoredResult{
				{PointID: "abc-123", Score: 0.01},
			},
			threshold: 0.0,
			wantScore: 0.01,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, matched := FindEventMatch(tt.eventUUID, tt.results, tt.threshold)
			if matched != tt.wantMatch {
				t.Errorf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if score != tt.wantScore {
				t.Errorf("score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}
