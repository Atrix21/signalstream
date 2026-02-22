package alerter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/embedding"
	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/notification"
	"github.com/Atrix21/signalstream/internal/platform"
	"github.com/Atrix21/signalstream/internal/retry"
)

const qdrantCollectionName = "financial_events"

// StrategyStore abstracts strategy persistence for testability.
type StrategyStore interface {
	GetAllActiveStrategies(ctx context.Context) ([]database.Strategy, error)
}

// VectorSearcher abstracts vector similarity search for testability.
type VectorSearcher interface {
	QuerySimilar(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error)
}

// ScoredResult is a provider-agnostic representation of a vector search result.
type ScoredResult struct {
	PointID string
	Score   float32
}

// QdrantSearcher implements VectorSearcher using the Qdrant client with retry.
type QdrantSearcher struct {
	client *qdrant.Client
	retry  retry.Config
}

// NewQdrantSearcher creates a VectorSearcher backed by Qdrant.
func NewQdrantSearcher(client *qdrant.Client) *QdrantSearcher {
	return &QdrantSearcher{
		client: client,
		retry:  retry.DefaultConfig(),
	}
}

func (q *QdrantSearcher) QuerySimilar(ctx context.Context, vector []float32, sources, tickers []string, limit int) ([]ScoredResult, error) {
	filter := buildQdrantFilter(sources, tickers)
	l := uint64(limit)

	var points []*qdrant.ScoredPoint
	err := retry.Do(ctx, q.retry, "qdrant-query", func() error {
		resp, err := q.client.Query(ctx, &qdrant.QueryPoints{
			CollectionName: qdrantCollectionName,
			Query:          qdrant.NewQuery(vector...),
			Filter:         filter,
			Limit:          &l,
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			metrics.Global.RetryAttempts.Add(1)
			return err
		}
		points = resp
		return nil
	})
	if err != nil {
		return nil, err
	}

	results := make([]ScoredResult, len(points))
	for i, p := range points {
		results[i] = ScoredResult{
			PointID: p.GetId().GetUuid(),
			Score:   p.Score,
		}
	}
	return results, nil
}

// Service evaluates incoming events against user-defined strategies.
type Service struct {
	embedder   embedding.Embedder
	searcher   VectorSearcher
	notifier   notification.Notifier
	strategies StrategyStore
}

// NewService creates an alerter service with injected dependencies.
func NewService(embedder embedding.Embedder, searcher VectorSearcher, notifier notification.Notifier, strategies StrategyStore) *Service {
	return &Service{
		embedder:   embedder,
		searcher:   searcher,
		notifier:   notifier,
		strategies: strategies,
	}
}

// CheckEventAgainstStrategies evaluates a single event against all active strategies.
func (s *Service) CheckEventAgainstStrategies(ctx context.Context, event platform.NormalizedEvent) {
	start := time.Now()
	logger := slog.With("event_id", event.ID, "source", event.Source)

	strategies, err := s.strategies.GetAllActiveStrategies(ctx)
	if err != nil {
		logger.Error("failed to fetch active strategies", "error", err)
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	for _, strategy := range strategies {
		s.evaluateStrategy(ctx, event, strategy, logger)
	}

	logger.Info("strategy evaluation completed",
		"strategies_checked", len(strategies),
		"latency_ms", time.Since(start).Milliseconds(),
	)
}

func (s *Service) evaluateStrategy(ctx context.Context, event platform.NormalizedEvent, strategy database.Strategy, logger *slog.Logger) {
	stratLogger := logger.With("strategy_id", strategy.ID, "strategy_name", strategy.Name)

	if !MatchesSimpleFilters(event, strategy) {
		return
	}

	queryVector, err := s.embedder.Embed(ctx, strategy.Query)
	if err != nil {
		stratLogger.Error("failed to embed strategy query", "error", err)
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	results, err := s.searcher.QuerySimilar(ctx, queryVector, strategy.Source, strategy.Tickers, 10)
	if err != nil {
		stratLogger.Error("vector search failed", "error", err)
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	eventUUID := uuid.NewSHA1(uuid.Nil, []byte(event.ID)).String()
	score, matched := FindEventMatch(eventUUID, results, strategy.SimilarityThreshold)

	if !matched {
		return
	}

	stratLogger.Info("alert triggered",
		"similarity_score", fmt.Sprintf("%.4f", score),
		"threshold", fmt.Sprintf("%.2f", strategy.SimilarityThreshold),
		"recipient", strategy.OwnerEmail,
	)

	alert := notification.AlertData{
		UserID:              strategy.UserID,
		StrategyID:          strategy.ID,
		StrategyDescription: strategy.Description,
		Recipient:           strategy.OwnerEmail,
		EventID:             event.ID,
		EventTitle:          event.Title,
		EventSource:         event.Source,
		EventURL:            event.ContentURL,
		SimilarityScore:     score,
		SimilarityThreshold: float32(strategy.SimilarityThreshold),
	}

	if err := s.notifier.Send(alert); err != nil {
		stratLogger.Error("failed to send notification", "error", err)
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	metrics.Global.AlertsTriggered.Add(1)
}

// MatchesSimpleFilters checks if an event passes a strategy's source and ticker filters.
// Exported for testing.
func MatchesSimpleFilters(event platform.NormalizedEvent, strategy database.Strategy) bool {
	if len(strategy.Source) > 0 {
		sourceMatch := false
		for _, source := range strategy.Source {
			if event.Source == source {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			return false
		}
	}

	if len(strategy.Tickers) > 0 {
		if len(event.Tickers) == 0 {
			return false
		}
		tickerMatch := false
		for _, eventTicker := range event.Tickers {
			for _, strategyTicker := range strategy.Tickers {
				if eventTicker == strategyTicker {
					tickerMatch = true
					break
				}
			}
			if tickerMatch {
				break
			}
		}
		if !tickerMatch {
			return false
		}
	}

	return true
}

// FindEventMatch checks if the event's UUID appears in the search results above the threshold.
// Exported for testing.
func FindEventMatch(eventUUID string, results []ScoredResult, threshold float64) (float32, bool) {
	for _, r := range results {
		if r.PointID == eventUUID && float64(r.Score) >= threshold {
			return r.Score, true
		}
	}
	return 0, false
}

func buildQdrantFilter(sources, tickers []string) *qdrant.Filter {
	filter := &qdrant.Filter{Must: []*qdrant.Condition{}}

	if len(sources) > 0 {
		var conditions []*qdrant.Condition
		for _, source := range sources {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key:   "source",
						Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: source}},
					},
				},
			})
		}
		if len(conditions) == 1 {
			filter.Must = append(filter.Must, conditions[0])
		} else {
			filter.Must = append(filter.Must, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Filter{
					Filter: &qdrant.Filter{Should: conditions},
				},
			})
		}
	}

	if len(tickers) > 0 {
		var conditions []*qdrant.Condition
		for _, ticker := range tickers {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key:   "tickers",
						Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: ticker}},
					},
				},
			})
		}
		if len(conditions) == 1 {
			filter.Must = append(filter.Must, conditions[0])
		} else {
			filter.Must = append(filter.Must, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Filter{
					Filter: &qdrant.Filter{Should: conditions},
				},
			})
		}
	}

	return filter
}
