// cmd/signalstream/main.go
package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Atrix21/signalstream/internal/alerter" 
	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/enrichment"
	"github.com/Atrix21/signalstream/internal/ingestion"
	"github.com/Atrix21/signalstream/internal/notification" 
	"github.com/Atrix21/signalstream/internal/platform"
)

func main() {
	logFile, err := os.OpenFile("signalstream.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// MultiWriter logs to both the console and the file.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// 1. Load Config
	cfg := config.Get()
	log.Printf("Configuration loaded.")

	// 2. Ensure Qdrant Collection Exists
	if err := enrichment.EnsureCollectionExists(cfg); err != nil {
		log.Fatalf("Failed to ensure Qdrant collection exists: %v", err)
	}

	// 3. Initialize Enrichment Service
	enrichSvc, err := enrichment.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create enrichment service: %v", err)
	}

	// 4. Define User Strategies & Initialize Alerter 
	userStrategies := []alerter.Strategy{
		{
			ID:                  "palantir-strategy",
			Description:         "General news and sentiment about Palantir stock",
			OwnerEmail:          "palantir.investor@example.com",
			SearchQuery:         "Palantir stock price, company performance, and market analysis",
			SourceFilter:        []string{"Polygon.io"}, 
			TickersFilter:       []string{"PLTR"},       
			SimilarityThreshold: 0.35,                   
		},
		{
			ID:                  "tesla-strategy",
			Description:         "General news and sentiment about Tesla stock",
			OwnerEmail:          "tesla.investor@example.com",
			SearchQuery:         "Tesla stock price, vehicle production, and market analysis",
			SourceFilter:        []string{"Polygon.io"}, 
			TickersFilter:       []string{"TSLA"},       
			SimilarityThreshold: 0.35,                   
		},
		{
			ID:                  "test-sec-edgar-strategy",
			Description:         "ANYTHING from SEC EDGAR",
			OwnerEmail:          "sec-tester@example.com",
			SearchQuery:         "company filing",     
			SourceFilter:        []string{"SEC EDGAR"},
			TickersFilter:       []string{},          
			SimilarityThreshold: 0.0,                 
		},
	}
	log.Printf("Loaded %d user strategies.", len(userStrategies))

	// Initialize the Notifier and the Alerter Service
	notifier := notification.NewLogNotifier()
	alerterSvc, err := alerter.NewService(cfg, notifier, userStrategies)
	if err != nil {
		log.Fatalf("Failed to create alerter service: %v", err)
	}


	// 5. Setup application context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 6. Setup the central channel
	eventsChan := make(chan platform.NormalizedEvent, 100)

	// --- Launch Producer Goroutines ---
	wg.Add(2)
	go ingestion.RunNewsAPIPoller(ctx, &wg, eventsChan, cfg)
	go ingestion.RunSECFilingPoller(ctx, &wg, eventsChan)

	// --- Launch Consumer WORKER POOL ---
	const numWorkers = 4
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Pass the new alerter service to the worker.
		go eventWorker(i, ctx, &wg, enrichSvc, alerterSvc, eventsChan)
	}

	// --- Setup graceful shutdown ---
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	<-stopChan
	log.Println("Shutdown signal received. Starting graceful shutdown...")

	cancel()
	close(eventsChan)
	wg.Wait()

	log.Println("All services stopped. Exiting.")
}

// --- The eventWorker function signature AND body ---
func eventWorker(
	id int,
	ctx context.Context,
	wg *sync.WaitGroup,
	enrichSvc *enrichment.Service,
	alerterSvc *alerter.Service, // <-- Add this parameter
	events <-chan platform.NormalizedEvent,
) {
	defer wg.Done()
	log.Printf("Event worker %d started.", id)
	for event := range events {
		processCtx, processCancel := context.WithTimeout(ctx, 2*time.Minute)

		// Step 1: Enrich the event (same as before).
		if err := enrichSvc.ProcessEvent(processCtx, event); err != nil {
			log.Printf("[WORKER %d] Error enriching event %s: %v", id, event.ID, err)
			processCancel()
			continue // Don't try to alert on a failed event.
		}

		// --- NEW: Step 2: Check for alerts ---
		// If enrichment was successful, pass the event to the alerter service.
		alerterSvc.CheckEventAgainstStrategies(processCtx, event)

		processCancel()
	}
	log.Printf("Event worker %d shutting down.", id)
}
