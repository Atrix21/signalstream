package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/qdrant/go-client/qdrant"

	"github.com/Atrix21/signalstream/internal/alerter"
	"github.com/Atrix21/signalstream/internal/api"
	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/embedding"
	"github.com/Atrix21/signalstream/internal/enrichment"
	"github.com/Atrix21/signalstream/internal/ingestion"
	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/notification"
	"github.com/Atrix21/signalstream/internal/platform"
	"github.com/Atrix21/signalstream/internal/sse"
)

func main() {
	// --- Logging setup ---
	logFile, err := os.OpenFile("signalstream.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	mw := io.MultiWriter(os.Stdout, logFile)
	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	// --- Configuration ---
	cfg := config.Get()
	slog.Info("configuration loaded")

	// --- Database ---
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	// --- SSE Broker ---
	broker := sse.NewBroker()

	// --- API Server ---
	srv, err := api.NewServer(cfg, db, broker)
	if err != nil {
		slog.Error("failed to create API server", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("API server starting", "port", cfg.ServerPort)
		if err := http.ListenAndServe(":"+cfg.ServerPort, srv.Routes()); err != nil {
			slog.Error("API server failed", "error", err)
			os.Exit(1)
		}
	}()

	// --- Shared Qdrant client ---
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.QdrantHost,
		Port: cfg.QdrantPort,
	})
	if err != nil {
		slog.Error("failed to create Qdrant client", "error", err)
		os.Exit(1)
	}

	if err := enrichment.EnsureCollectionExists(qdrantClient); err != nil {
		slog.Error("failed to ensure Qdrant collection", "error", err)
		os.Exit(1)
	}

	// --- Shared embedding client ---
	embedClient := embedding.NewClient(cfg.OpenAIAPIKey)

	// --- Enrichment Service ---
	enrichSvc := enrichment.NewService(embedClient, qdrantClient)

	// --- Notification: log + DB persist + SSE broadcast ---
	logNotifier := notification.NewLogNotifier()
	dbNotifier := notification.NewDatabaseNotifier(db, broker)
	notifier := notification.NewMultiNotifier(logNotifier, dbNotifier)

	// --- Alerter Service ---
	searcher := alerter.NewQdrantSearcher(qdrantClient)
	alerterSvc := alerter.NewService(embedClient, searcher, notifier, db)

	// --- Application lifecycle ---
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	eventsChan := make(chan platform.NormalizedEvent, 100)

	// --- Producers ---
	wg.Add(2)
	go ingestion.RunNewsAPIPoller(ctx, &wg, eventsChan, cfg)
	go ingestion.RunSECFilingPoller(ctx, &wg, eventsChan)

	// --- Consumer worker pool ---
	const numWorkers = 4
	wg.Add(numWorkers)
	for i := range numWorkers {
		go eventWorker(i, ctx, &wg, enrichSvc, alerterSvc, eventsChan)
	}

	// --- Graceful shutdown ---
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	<-stopChan
	slog.Info("shutdown signal received, starting graceful shutdown")

	cancel()
	close(eventsChan)
	wg.Wait()

	snapshot := metrics.Global.Snapshot()
	slog.Info("final metrics", "metrics", snapshot)
	slog.Info("all services stopped, exiting")
}

func eventWorker(
	id int,
	ctx context.Context,
	wg *sync.WaitGroup,
	enrichSvc *enrichment.Service,
	alerterSvc *alerter.Service,
	events <-chan platform.NormalizedEvent,
) {
	defer wg.Done()
	slog.Info("event worker started", "worker_id", id)

	for event := range events {
		processCtx, processCancel := context.WithTimeout(ctx, 2*time.Minute)

		if err := enrichSvc.ProcessEvent(processCtx, event); err != nil {
			slog.Error("enrichment failed",
				"worker_id", id,
				"event_id", event.ID,
				"error", err,
			)
			processCancel()
			continue
		}

		alerterSvc.CheckEventAgainstStrategies(processCtx, event)
		metrics.Global.EventsProcessed.Add(1)

		processCancel()
	}

	slog.Info("event worker shutting down", "worker_id", id)
}
