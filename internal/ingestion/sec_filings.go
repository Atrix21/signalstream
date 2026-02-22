package ingestion

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/platform"
	"github.com/Atrix21/signalstream/internal/sec"
	"github.com/mmcdole/gofeed"
)

const secEdgarURL = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-k&count=40&output=atom"

var cikRegex = regexp.MustCompile(`\(([^)]+)\)`)

// RunSECFilingPoller is a producer goroutine that polls the SEC EDGAR feed.
func RunSECFilingPoller(ctx context.Context, wg *sync.WaitGroup, events chan<- platform.NormalizedEvent) {
	defer wg.Done()
	slog.Info("SEC filing poller started")

	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	fp.UserAgent = "SignalStream App v0.1 adityawinning3@gmail.com"

	ticker := time.NewTicker(90 * time.Second)
	defer ticker.Stop()

	lastSeenGUID := ""

	// Run the first check immediately.
	go func() {
		pollSECFeed(ctx, fp, &lastSeenGUID, events)
	}()

	for {
		select {
		case <-ticker.C:
			pollSECFeed(ctx, fp, &lastSeenGUID, events)
		case <-ctx.Done():
			slog.Info("SEC filing poller shutting down")
			return
		}
	}
}

func pollSECFeed(ctx context.Context, fp *gofeed.Parser, lastSeenGUID *string, events chan<- platform.NormalizedEvent) {
	slog.Debug("polling SEC EDGAR RSS feed")

	feed, err := fp.ParseURLWithContext(secEdgarURL, ctx)
	if err != nil {
		slog.Error("error fetching SEC feed", "error", err)
		metrics.Global.ErrorsTotal.Add(1)
		return
	}

	cikMap := sec.GetCIKMap()
	var newFilings []platform.NormalizedEvent

	for _, item := range feed.Items {
		if item.GUID == *lastSeenGUID {
			break
		}

		event := platform.NormalizedEvent{
			ID:          item.GUID,
			Source:      "SEC EDGAR",
			Tickers:     lookupTickers(item.Title, cikMap),
			Title:       item.Title,
			ContentURL:  item.Link,
			PublishedAt: *item.PublishedParsed,
			IngestedAt:  time.Now().UTC(),
		}
		newFilings = append(newFilings, event)
	}

	if len(newFilings) > 0 {
		*lastSeenGUID = newFilings[0].ID
		for i := len(newFilings) - 1; i >= 0; i-- {
			events <- newFilings[i]
			metrics.Global.EventsIngested.Add(1)
		}
		slog.Info("ingested SEC filings", "count", len(newFilings))
	}
}

func lookupTickers(title string, cikMap *sec.CIKMap) []string {
	matches := cikRegex.FindStringSubmatch(title)
	if len(matches) < 2 {
		return nil
	}

	cikStr := matches[1]
	if ticker, found := cikMap.Ticker(cikStr); found {
		return []string{ticker}
	}

	return nil
}
