package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"backend/database"
	"backend/scryfall"
	"backend/server"
	"backend/services"
)

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	logLevel := parseLogLevel(os.Getenv("LOG_LEVEL"))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// Initialize database client
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}
	dbClient, err := database.NewClient(dbPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := dbClient.Close(); err != nil {
			slog.Error("error closing database", "error", err)
		}
	}()

	// Initialize Scryfall client
	scryfallClient, err := scryfall.NewClient()
	if err != nil {
		slog.Error("failed to initialize scryfall client", "error", err)
		os.Exit(1)
	}
	defer scryfallClient.Close()

	// Data directory for storing files (icons, etc.)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Initialize services
	settingsService := services.NewSettingsService(dbClient.DB)
	jobService := services.NewJobService(dbClient.DB)
	bulkDataService := services.NewBulkDataService(dbClient.DB, jobService, settingsService)
	setDataService := services.NewSetDataService(dbClient.DB, jobService, settingsService, scryfallClient, dataDir)

	// Create application-level cancellable context for background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel any stale jobs from previous runs
	cancelledCount, err := jobService.CancelStaleJobs(ctx)
	if err != nil {
		slog.Warn("failed to cancel stale jobs", "error", err)
	} else if cancelledCount > 0 {
		slog.Info("cancelled stale jobs from previous run", "count", cancelledCount)
	}

	// Trigger initial bulk data import if no data exists
	if err := bulkDataService.TriggerInitialImport(ctx); err != nil {
		slog.Warn("failed to trigger initial import", "error", err)
	}

	// Trigger initial set data import if no data exists
	if err := setDataService.TriggerInitialImport(ctx); err != nil {
		slog.Warn("failed to trigger initial set import", "error", err)
	}

	// Initialize server with database, scryfall clients, and services
	srv := server.NewServer(ctx, dbClient, scryfallClient, settingsService, jobService, bulkDataService, setDataService, dataDir)

	scheduler := services.NewScheduler(bulkDataService, setDataService, jobService, settingsService)
	scheduler.Start(ctx)
	defer scheduler.Stop()

	// Graceful shutdown on interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("shutting down")
		cancel() // Stop scheduler
		if err := srv.Close(); err != nil {
			slog.Error("error shutting down server", "error", err)
		}
	}()

	// Start the server (blocks until shutdown)
	if err := srv.Start(); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
