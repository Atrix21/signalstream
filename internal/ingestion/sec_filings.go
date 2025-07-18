package ingestion

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/Atrix21/signalstream/internal/platform"
	"github.com/Atrix21/signalstream/internal/sec"
	"github.com/mmcdole/gofeed"
)

const secEdgarURL = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-k&count=40&output=atom"

// Used to find ticker symbols like (AAPL) or (TSLA) in the filing title.
var cikRegex = regexp.MustCompile(`\(([^)]+)\)`)

// RunSECFilingPoller is a producer goroutine that polls the SEC EDGAR feed.
func RunSECFilingPoller(ctx context.Context, wg *sync.WaitGroup, events chan<- platform.NormalizedEvent) {
	defer wg.Done()
	log.Println("SEC Filing poller started.")

	fp := gofeed.NewParser()

	fp.Client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	fp.UserAgent = "SignalStream App v0.1 adityawinning3@gmail.com"

	// Poll every 90 seconds. The feed doesn't update as rapidly as news.
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
			log.Println("SEC Filing poller shutting down.")
			return
		}
	}
}

func pollSECFeed(ctx context.Context, fp *gofeed.Parser, lastSeenGUID *string, events chan<- platform.NormalizedEvent) {
	log.Println("Polling SEC EDGAR RSS feed...")

	// The context is passed to the parser so the HTTP request can be cancelled.
	feed, err := fp.ParseURLWithContext(secEdgarURL, ctx)
	if err != nil {
		log.Printf("Error fetching or parsing SEC feed: %v", err)
		return
	}

	cikMap := sec.GetCIKMap()

	var newFilings []platform.NormalizedEvent

	for _, item := range feed.Items {
		if item.GUID == *lastSeenGUID {
			break 
		}

		log.Printf("Found new filing: %s", item.Title)

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
		}
	}
}

// extractTickers finds potential ticker symbols from a string like "Form 8-K for Apple Inc. (AAPL)".
func lookupTickers(title string, cikMap *sec.CIKMap) []string {
	matches := cikRegex.FindStringSubmatch(title)
	if len(matches) < 2 {
		return nil
	}

	cikStr := matches[1]

	if ticker, found := cikMap.Ticker(cikStr); found {
		return []string{ticker}
	}

	return nil // Return nil if no ticker was found for the CIK.
}
