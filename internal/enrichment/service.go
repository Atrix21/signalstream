package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"github.com/Atrix21/signalstream/internal/embedding"
	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/platform"
	"github.com/Atrix21/signalstream/internal/retry"
)

const qdrantCollectionName = "financial_events"

// ContentFetcher fetches and extracts article text from a URL.
type ContentFetcher interface {
	Fetch(url string, timeout time.Duration) (string, error)
}

// ReadabilityFetcher implements ContentFetcher using go-readability.
type ReadabilityFetcher struct{}

func (f *ReadabilityFetcher) Fetch(url string, timeout time.Duration) (string, error) {
	article, err := readability.FromURL(url, timeout)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

// Service handles event enrichment: fetch content, embed, store in vector DB.
type Service struct {
	embedder     embedding.Embedder
	qdrantClient *qdrant.Client
	fetcher      ContentFetcher
	retry        retry.Config
}

// NewService creates an enrichment service with injected dependencies.
func NewService(embedder embedding.Embedder, qdrantClient *qdrant.Client) *Service {
	return &Service{
		embedder:     embedder,
		qdrantClient: qdrantClient,
		fetcher:      &ReadabilityFetcher{},
		retry:        retry.DefaultConfig(),
	}
}

// ProcessEvent fetches article content, generates an embedding, and upserts to Qdrant.
func (s *Service) ProcessEvent(ctx context.Context, event platform.NormalizedEvent) error {
	start := time.Now()
	logger := slog.With("event_id", event.ID, "source", event.Source, "title", event.Title)

	logger.Info("enrichment started")

	text, err := s.fetcher.Fetch(event.ContentURL, 30*time.Second)
	if err != nil {
		logger.Warn("failed to fetch article content", "url", event.ContentURL, "error", err)
		return nil // Non-fatal: skip events we can't fetch.
	}

	if len(text) < 50 {
		logger.Info("skipping event with insufficient text", "text_length", len(text))
		return nil
	}
	event.RawText = text

	vector, err := s.embedder.Embed(ctx, event.RawText)
	if err != nil {
		metrics.Global.ErrorsTotal.Add(1)
		return fmt.Errorf("embed event text: %w", err)
	}

	if err := s.upsertPoint(ctx, event, vector); err != nil {
		metrics.Global.ErrorsTotal.Add(1)
		return fmt.Errorf("upsert to qdrant: %w", err)
	}

	metrics.Global.EventsEnriched.Add(1)
	logger.Info("enrichment completed", "latency_ms", time.Since(start).Milliseconds())
	return nil
}

func (s *Service) upsertPoint(ctx context.Context, event platform.NormalizedEvent, vector []float32) error {
	if s.qdrantClient == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	payload := qdrant.NewValueMap(map[string]any{
		"source":       event.Source,
		"title":        event.Title,
		"content_url":  event.ContentURL,
		"published_at": event.PublishedAt.Format(time.RFC3339),
	})

	if len(event.Tickers) > 0 {
		payload["tickers"] = &qdrant.Value{
			Kind: &qdrant.Value_ListValue{
				ListValue: &qdrant.ListValue{Values: stringSliceToQdrantValues(event.Tickers)},
			},
		}
	}

	pointUUID := uuid.NewSHA1(uuid.Nil, []byte(event.ID))
	wait := true

	return retry.Do(ctx, s.retry, "qdrant-upsert", func() error {
		_, err := s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: qdrantCollectionName,
			Points: []*qdrant.PointStruct{
				{
					Id:      qdrant.NewIDUUID(pointUUID.String()),
					Vectors: qdrant.NewVectors(vector...),
					Payload: payload,
				},
			},
			Wait: &wait,
		})
		if err != nil {
			metrics.Global.RetryAttempts.Add(1)
		}
		return err
	})
}

// EnsureCollectionExists creates the Qdrant collection if it doesn't already exist.
func EnsureCollectionExists(qdrantClient *qdrant.Client) error {
	exists, err := qdrantClient.CollectionExists(context.Background(), qdrantCollectionName)
	if err != nil {
		return fmt.Errorf("check collection '%s': %w", qdrantCollectionName, err)
	}
	if exists {
		slog.Info("qdrant collection exists", "collection", qdrantCollectionName)
		return nil
	}

	slog.Info("creating qdrant collection", "collection", qdrantCollectionName, "dimensions", embedding.Dimensions)
	err = qdrantClient.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: qdrantCollectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     embedding.Dimensions,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("create collection '%s': %w", qdrantCollectionName, err)
	}

	slog.Info("qdrant collection created", "collection", qdrantCollectionName)
	return nil
}

func stringSliceToQdrantValues(slice []string) []*qdrant.Value {
	values := make([]*qdrant.Value, len(slice))
	for i, s := range slice {
		values[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
	}
	return values
}
