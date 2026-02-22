package embedding

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/retry"
	openai "github.com/sashabaranov/go-openai"
)

const (
	Model      = openai.SmallEmbedding3
	Dimensions = 1536
)

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Client implements Embedder using the OpenAI API with retry.
type Client struct {
	openai *openai.Client
	retry  retry.Config
}

// NewClient creates an embedding client backed by OpenAI.
func NewClient(apiKey string) *Client {
	return &Client{
		openai: openai.NewClient(apiKey),
		retry:  retry.DefaultConfig(),
	}
}

// Embed generates a vector embedding for the given text with automatic retry.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	var result []float32

	err := retry.Do(ctx, c.retry, "openai-embedding", func() error {
		req := openai.EmbeddingRequest{
			Input: []string{text},
			Model: Model,
		}
		resp, err := c.openai.CreateEmbeddings(ctx, req)
		if err != nil {
			metrics.Global.RetryAttempts.Add(1)
			slog.Warn("embedding API call failed, will retry", "error", err)
			return fmt.Errorf("create embedding: %w", err)
		}
		if len(resp.Data) == 0 {
			return fmt.Errorf("empty embedding response")
		}
		result = resp.Data[0].Embedding
		return nil
	})

	return result, err
}
