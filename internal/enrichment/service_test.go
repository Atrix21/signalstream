package enrichment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Atrix21/signalstream/internal/platform"
)

// --- Mocks ---

type mockEmbedder struct {
	embedFn func(ctx context.Context, text string) ([]float32, error)
	calls   int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.calls++
	return m.embedFn(ctx, text)
}

type mockFetcher struct {
	fetchFn func(url string, timeout time.Duration) (string, error)
	calls   int
}

func (m *mockFetcher) Fetch(url string, timeout time.Duration) (string, error) {
	m.calls++
	return m.fetchFn(url, timeout)
}

// --- Tests ---

func TestProcessEvent_SkipsShortContent(t *testing.T) {
	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatal("embedder should not be called for short content")
			return nil, nil
		},
	}

	fetcher := &mockFetcher{
		fetchFn: func(url string, timeout time.Duration) (string, error) {
			return "too short", nil // < 50 chars
		},
	}

	svc := &Service{
		embedder: embedder,
		fetcher:  fetcher,
	}

	event := platform.NormalizedEvent{
		ID:         "test-1",
		ContentURL: "https://example.com/short",
	}

	err := svc.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("expected nil error for short content, got: %v", err)
	}

	if embedder.calls != 0 {
		t.Fatalf("embedder should not be called, was called %d times", embedder.calls)
	}
}

func TestProcessEvent_SkipsFetchError(t *testing.T) {
	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatal("embedder should not be called on fetch error")
			return nil, nil
		},
	}

	fetcher := &mockFetcher{
		fetchFn: func(url string, timeout time.Duration) (string, error) {
			return "", errors.New("404 not found")
		},
	}

	svc := &Service{
		embedder: embedder,
		fetcher:  fetcher,
	}

	event := platform.NormalizedEvent{
		ID:         "test-2",
		ContentURL: "https://example.com/missing",
	}

	err := svc.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("fetch errors should be non-fatal, got: %v", err)
	}
}

func TestProcessEvent_EmbeddingFailure(t *testing.T) {
	longContent := make([]byte, 200)
	for i := range longContent {
		longContent[i] = 'a'
	}

	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return nil, errors.New("rate limited")
		},
	}

	fetcher := &mockFetcher{
		fetchFn: func(url string, timeout time.Duration) (string, error) {
			return string(longContent), nil
		},
	}

	svc := &Service{
		embedder: embedder,
		fetcher:  fetcher,
	}

	event := platform.NormalizedEvent{
		ID:         "test-3",
		ContentURL: "https://example.com/article",
	}

	err := svc.ProcessEvent(context.Background(), event)
	if err == nil {
		t.Fatal("embedding failures should propagate as errors")
	}
}

func TestProcessEvent_FetchAndEmbedPipeline(t *testing.T) {
	longContent := make([]byte, 200)
	for i := range longContent {
		longContent[i] = 'x'
	}

	var embeddedText string
	embedder := &mockEmbedder{
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			embeddedText = text
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	var fetchedURL string
	fetcher := &mockFetcher{
		fetchFn: func(url string, timeout time.Duration) (string, error) {
			fetchedURL = url
			return string(longContent), nil
		},
	}

	// Use a wrapper that records calls without actually hitting Qdrant.
	// We test the fetch → embed portion of the pipeline here.
	// The upsert (Qdrant) layer is tested separately via integration tests.
	svc := &Service{
		embedder: embedder,
		fetcher:  fetcher,
		// qdrantClient intentionally nil — we verify fetch+embed and accept the upsert error.
	}

	event := platform.NormalizedEvent{
		ID:          "test-4",
		Source:      "Polygon.io",
		ContentURL:  "https://example.com/article",
		PublishedAt: time.Now(),
	}

	// ProcessEvent will fail at the upsert stage (nil client), which is expected.
	// We verify that the earlier pipeline stages executed correctly.
	err := svc.ProcessEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error from nil qdrant client upsert")
	}

	if fetcher.calls != 1 {
		t.Errorf("fetcher should be called once, got %d", fetcher.calls)
	}
	if embedder.calls != 1 {
		t.Errorf("embedder should be called once, got %d", embedder.calls)
	}
	if fetchedURL != "https://example.com/article" {
		t.Errorf("fetched wrong URL: %s", fetchedURL)
	}
	if embeddedText != string(longContent) {
		t.Error("embedded text should match fetched content")
	}
}
