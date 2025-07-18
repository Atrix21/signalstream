package ingestion

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"golang.org/x/time/rate"

	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/platform"
)

// RunNewsAPIPoller is a producer goroutine that polls the Polygon.io news API
// and sends normalized events into the events channel.
func RunNewsAPIPoller(ctx context.Context, wg *sync.WaitGroup, events chan<- platform.NormalizedEvent, cfg config.AppConfig) {
	defer wg.Done()
	log.Println("News API poller started.")

	c := polygon.New(cfg.PolygonAPIKey)

	limiter := rate.NewLimiter(rate.Every(12*time.Second), 1)

	lastSeenID := ""

	for {
		err := limiter.Wait(ctx)
		if err != nil {

			log.Println("News API poller shutting down.")
			return
		}

		pollNews(ctx, c, &lastSeenID, events)
	}
}

// pollNews contains the logic for a single polling action.
func pollNews(ctx context.Context, c *polygon.Client, lastSeenID *string, events chan<- platform.NormalizedEvent) {
	log.Println("Polling news API...")

	params := models.ListTickerNewsParams{}.WithLimit(50)
	iter := c.ListTickerNews(ctx, params)

	var newArticles []platform.NormalizedEvent

	for iter.Next() {
		newsItem := iter.Item()

		if newsItem.ID == *lastSeenID {
			break
		}

		log.Printf("Found new article: %s", newsItem.Title)

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
		log.Printf("Error iterating news: %v", iter.Err())
		return
	}

	if len(newArticles) > 0 {
		*lastSeenID = newArticles[0].ID
		for i := len(newArticles) - 1; i >= 0; i-- {
			events <- newArticles[i]
		}
	}
}
