package enrichment

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	openai "github.com/sashabaranov/go-openai"

	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/platform"
)

const (
	qdrantCollectionName = "financial_events"
	embeddingModel       = openai.SmallEmbedding3
	embeddingDimensions  = 1536
)


type Service struct {
	openAIClient *openai.Client
	qdrantClient *qdrant.Client
}


func NewService(cfg config.AppConfig) (*Service, error) {

	qClient, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334, // Default REST port
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	openAIClient := openai.NewClient(cfg.OpenAIAPIKey)

	return &Service{
		openAIClient: openAIClient,
		qdrantClient: qClient,
	}, nil
}


func (s *Service) ProcessEvent(ctx context.Context, event platform.NormalizedEvent) error {
	log.Printf("[ENRICHMENT] Starting for event: %s", event.Title)

	article, err := readability.FromURL(event.ContentURL, 30*time.Second)
	if err != nil {
		log.Printf("[ENRICHMENT] Failed to parse URL %s: %v", event.ContentURL, err)
		return nil
	}
	event.RawText = article.TextContent
	if len(event.RawText) < 50 {
		log.Printf("[ENRICHMENT] Skipping event with insufficient text: %s", event.Title)
		return nil
	}

	embedding, err := s.createEmbedding(ctx, event.RawText)
	if err != nil {
		log.Printf("[ENRICHMENT] Failed to create embedding for %s: %v", event.Title, err)
		return err
	}

	err = s.upsertPoint(ctx, event, embedding)
	if err != nil {
		log.Printf("[ENRICHMENT] Failed to upsert point to Qdrant for %s: %v", event.Title, err)
		return err
	}

	log.Printf("[ENRICHMENT] Successfully processed and stored event: %s", event.Title)
	return nil
}


func (s *Service) createEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: embeddingModel,
	}
	resp, err := s.openAIClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}


func (s *Service) upsertPoint(ctx context.Context, event platform.NormalizedEvent, vector []float32) error {

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

	_, err := s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qdrantCollectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDUUID(pointUUID.String()), 
				Vectors: qdrant.NewVectors(vector...),
				Payload: payload,
			},
		},
		Wait: boolPtr(true), 
	})
	return err
}

func boolPtr(b bool) *bool {
	return &b
}


func EnsureCollectionExists(cfg config.AppConfig) error {
	qClient, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		return err
	}

	exists, err := qClient.CollectionExists(context.Background(), qdrantCollectionName)
	if err != nil {
		return fmt.Errorf("failed to check if collection '%s' exists: %w", qdrantCollectionName, err)
	}
	if exists {
		log.Printf("Qdrant collection '%s' already exists.", qdrantCollectionName)
		return nil
	}

	log.Printf("Qdrant collection '%s' not found. Creating...", qdrantCollectionName)
	err = qClient.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: qdrantCollectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     embeddingDimensions,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection '%s': %w", qdrantCollectionName, err)
	}

	log.Printf("Qdrant collection '%s' created successfully.", qdrantCollectionName)
	return nil
}

// stringSliceToQdrantValues converts a Go string slice to Qdrant's list format.
func stringSliceToQdrantValues(slice []string) []*qdrant.Value {
	values := make([]*qdrant.Value, len(slice))
	for i, s := range slice {
		values[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
	}
	return values
}
