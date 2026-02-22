package alerter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/notification"
	"github.com/Atrix21/signalstream/internal/platform"
)

// --- Mock implementations ---

type mockEmbedder struct {
	embedFn func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.embedFn(ctx, text)
}

type mockSearcher struct {
	queryFn func(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error)
}

func (m *mockSearcher) QuerySimilar(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error) {
	return m.queryFn(ctx, vector, sources, tickers, limit)
}

type mockStrategyStore struct {
	strategies []database.Strategy
	err        error
}

func (m *mockStrategyStore) GetAllActiveStrategies(ctx context.Context) ([]database.Strategy, error) {
	return m.strategies, m.err
}

type mockNotifier struct {
	mu     sync.Mutex
	alerts []notification.AlertData
	err    error
}

func (m *mockNotifier) Send(data notification.AlertData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, data)
	return m.err
}

func (m *mockNotifier) getAlerts() []notification.AlertData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]notification.AlertData(nil), m.alerts...)
}

// --- Tests ---

func TestCheckEventAgainstStrategies_AlertTriggered(t *testing.T) {
	userID := uuid.New()
	strategyID := uuid.New()
	eventID := "test-event-123"
	eventUUID := uuid.NewSHA1(uuid.Nil, []byte(eventID)).String()

	strategies := []database.Strategy{
		{
			ID:                  strategyID,
			UserID:              userID,
			Name:                "Test Strategy",
			Description:         "Matches tech news",
			Query:               "technology acquisition",
			Source:              []string{"Polygon.io"},
			Tickers:             []string{"AAPL"},
			SimilarityThreshold: 0.75,
			IsActive:            true,
			OwnerEmail:          "user@test.com",
		},
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	searcher := &mockSearcher{
		queryFn: func(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error) {
			return []ScoredResult{
				{PointID: eventUUID, Score: 0.85},
			}, nil
		},
	}

	notifier := &mockNotifier{}
	store := &mockStrategyStore{strategies: strategies}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{
		ID:          eventID,
		Source:      "Polygon.io",
		Tickers:     []string{"AAPL"},
		Title:       "Apple acquires AI startup",
		ContentURL:  "https://example.com/article",
		PublishedAt: time.Now(),
	}

	svc.CheckEventAgainstStrategies(context.Background(), event)

	alerts := notifier.getAlerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.UserID != userID {
		t.Errorf("user ID mismatch: got %v, want %v", alert.UserID, userID)
	}
	if alert.StrategyID != strategyID {
		t.Errorf("strategy ID mismatch: got %v, want %v", alert.StrategyID, strategyID)
	}
	if alert.EventTitle != "Apple acquires AI startup" {
		t.Errorf("event title mismatch: got %q", alert.EventTitle)
	}
	if alert.SimilarityScore != 0.85 {
		t.Errorf("similarity score mismatch: got %v, want 0.85", alert.SimilarityScore)
	}
}

func TestCheckEventAgainstStrategies_FilteredBySource(t *testing.T) {
	strategies := []database.Strategy{
		{
			ID:     uuid.New(),
			Name:   "SEC Only Strategy",
			Query:  "earnings report",
			Source: []string{"SEC EDGAR"},
		},
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatal("embedder should not be called when event is filtered out")
			return nil, nil
		},
	}

	notifier := &mockNotifier{}
	store := &mockStrategyStore{strategies: strategies}
	searcher := &mockSearcher{}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{
		ID:      "polygon-event",
		Source:  "Polygon.io", // Doesn't match SEC EDGAR filter
		Tickers: []string{"AAPL"},
	}

	svc.CheckEventAgainstStrategies(context.Background(), event)

	if len(notifier.getAlerts()) != 0 {
		t.Fatal("no alert should be triggered when source doesn't match")
	}
}

func TestCheckEventAgainstStrategies_BelowThreshold(t *testing.T) {
	eventID := "test-event-456"
	eventUUID := uuid.NewSHA1(uuid.Nil, []byte(eventID)).String()

	strategies := []database.Strategy{
		{
			ID:                  uuid.New(),
			Query:               "specific niche query",
			SimilarityThreshold: 0.90,
			IsActive:            true,
		},
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2}, nil
		},
	}

	searcher := &mockSearcher{
		queryFn: func(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error) {
			return []ScoredResult{
				{PointID: eventUUID, Score: 0.70}, // Below 0.90 threshold
			}, nil
		},
	}

	notifier := &mockNotifier{}
	store := &mockStrategyStore{strategies: strategies}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{ID: eventID, Source: "Polygon.io"}
	svc.CheckEventAgainstStrategies(context.Background(), event)

	if len(notifier.getAlerts()) != 0 {
		t.Fatal("no alert should be triggered when score is below threshold")
	}
}

func TestCheckEventAgainstStrategies_EmbedderError(t *testing.T) {
	strategies := []database.Strategy{
		{ID: uuid.New(), Query: "test", IsActive: true},
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return nil, errors.New("API rate limit")
		},
	}

	notifier := &mockNotifier{}
	store := &mockStrategyStore{strategies: strategies}
	searcher := &mockSearcher{}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{ID: "test", Source: "Polygon.io"}
	svc.CheckEventAgainstStrategies(context.Background(), event)

	if len(notifier.getAlerts()) != 0 {
		t.Fatal("no alert should be triggered on embedder error")
	}
}

func TestCheckEventAgainstStrategies_StrategyStoreError(t *testing.T) {
	embedder := &mockEmbedder{}
	notifier := &mockNotifier{}
	searcher := &mockSearcher{}
	store := &mockStrategyStore{err: errors.New("database down")}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{ID: "test", Source: "Polygon.io"}
	svc.CheckEventAgainstStrategies(context.Background(), event)

	if len(notifier.getAlerts()) != 0 {
		t.Fatal("no alert should be triggered on strategy store error")
	}
}

func TestCheckEventAgainstStrategies_MultipleStrategies(t *testing.T) {
	eventID := "multi-strategy-event"
	eventUUID := uuid.NewSHA1(uuid.Nil, []byte(eventID)).String()

	strategies := []database.Strategy{
		{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			Name:                "Strategy A",
			Query:               "tech mergers",
			Source:              []string{"Polygon.io"},
			SimilarityThreshold: 0.70,
			IsActive:            true,
			OwnerEmail:          "a@test.com",
		},
		{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			Name:                "Strategy B",
			Query:               "biotech patents",
			Source:              []string{"SEC EDGAR"}, // Won't match Polygon.io event
			SimilarityThreshold: 0.50,
			IsActive:            true,
			OwnerEmail:          "b@test.com",
		},
		{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			Name:                "Strategy C",
			Query:               "general news",
			SimilarityThreshold: 0.95, // Threshold too high
			IsActive:            true,
			OwnerEmail:          "c@test.com",
		},
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.5}, nil
		},
	}

	searcher := &mockSearcher{
		queryFn: func(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error) {
			return []ScoredResult{
				{PointID: eventUUID, Score: 0.80},
			}, nil
		},
	}

	notifier := &mockNotifier{}
	store := &mockStrategyStore{strategies: strategies}

	svc := NewService(embedder, searcher, notifier, store)

	event := platform.NormalizedEvent{ID: eventID, Source: "Polygon.io"}
	svc.CheckEventAgainstStrategies(context.Background(), event)

	alerts := notifier.getAlerts()
	// Strategy A: matches (source ok, score 0.80 >= 0.70)
	// Strategy B: filtered out (source mismatch)
	// Strategy C: score 0.80 < 0.95 threshold
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Recipient != "a@test.com" {
		t.Errorf("alert should be for Strategy A, got recipient %q", alerts[0].Recipient)
	}
}
