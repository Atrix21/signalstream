package ingestion

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"golang.org/x/time/rate"

	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/platform"
)

// RunNewsAPIPoller is a producer goroutine that polls the Polygon.io news API
// and sends normalized events into the events channel.
func RunNewsAPIPoller(ctx context.Context, wg *sync.WaitGroup, events chan<- platform.NormalizedEvent, cfg config.AppConfig) {
	defer wg.Done()
	slog.Info("news API poller started")

	c := polygon.New(cfg.PolygonAPIKey)
	limiter := rate.NewLimiter(rate.Every(12*time.Second), 1)
	lastSeenID := ""

	for {
		if err := limiter.Wait(ctx); err != nil {
			slog.Info("news API poller shutting down")
			return
		}

		pollNews(ctx, c, &lastSeenID, events)
	}
}

func pollNews(ctx context.Context, c *polygon.Client, lastSeenID *string, events chan<- platform.NormalizedEvent) {
	slog.Debug("polling news API")

	params := models.ListTickerNewsParams{}.WithLimit(50)
	iter := c.ListTickerNews(ctx, params)

	var newArticles []platform.NormalizedEvent

	for iter.Next() {
		newsItem := iter.Item()

		if newsItem.ID == *lastSeenID {
			break
		}

		var tickers []string
		tickers = append(tickers, newsItem.Tickers...)

		event := platform.NormalizedEvent{
			ID:          newsItem.ID,
			Source:      "Polygon.io",
			Tickers:     tickers,
			Title:       newsItem.Title,
			ContentURL:  newsItem.ArticleURL,
			PublishedAt: time.Time(newsItem.PublishedUTC),
			IngestedAt:  time.Now().UTC(),
		}
		newArticles = append(newArticles, event)
	}

	if iter.Err() != nil {
		slog.Error("error iterating news", "error", iter.Err())
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	if len(newArticles) > 0 {
		*lastSeenID = newArticles[0].ID
		for i := len(newArticles) - 1; i >= 0; i-- {
			events <- newArticles[i]
			metrics.Global.EventsIngested.Add(1)
		}
		slog.Info("ingested news articles", "count", len(newArticles))
	}
}
