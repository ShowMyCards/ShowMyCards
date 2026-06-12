package services

import (
	"backend/models"
	scryfallclient "backend/scryfall"
	"backend/version"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SymbolDataService handles symbology download and import.
//
// It mirrors SetDataService: it fetches the list of symbols from Scryfall's
// symbology endpoint (via the rate-limited Scryfall client), downloads each
// symbol's SVG, and upserts the cached SVG content into SQLite keyed by a
// normalized symbol code.
type SymbolDataService struct {
	db              *gorm.DB
	jobService      *JobService
	settingsService *SettingsService
	scryfallClient  *scryfallclient.Client
	httpClient      *http.Client
}

// NewSymbolDataService creates a new symbol data service
func NewSymbolDataService(db *gorm.DB, jobService *JobService, settingsService *SettingsService, scryfallClient *scryfallclient.Client) *SymbolDataService {
	return &SymbolDataService{
		db:              db,
		jobService:      jobService,
		settingsService: settingsService,
		scryfallClient:  scryfallClient,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

// SymbolJobMetadata represents the metadata stored in job.Metadata for symbol imports
type SymbolJobMetadata struct {
	Phase            string   `json:"phase"`
	TotalSymbols     int      `json:"total_symbols"`
	ProcessedSymbols int      `json:"processed_symbols"`
	SVGsDownloaded   int      `json:"svgs_downloaded"`
	FailedSymbols    int      `json:"failed_symbols"`
	FailureExamples  []string `json:"failure_examples"`
}

// CreateImportJob creates a new job for symbol data import
func (s *SymbolDataService) CreateImportJob(ctx context.Context) (*models.Job, error) {
	return s.jobService.Create(ctx, models.JobTypeSymbolDataImport, "{}")
}

// HasSymbolData checks if symbol data exists in the database
func (s *SymbolDataService) HasSymbolData() (bool, error) {
	var count int64
	if err := s.db.Model(&models.Symbol{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// TriggerInitialImport triggers an initial symbol data import if no data exists
func (s *SymbolDataService) TriggerInitialImport(ctx context.Context) error {
	hasData, err := s.HasSymbolData()
	if err != nil {
		return fmt.Errorf("failed to check for existing symbol data: %w", err)
	}

	if hasData {
		slog.Info("symbol data already exists, skipping initial import")
		return nil
	}

	slog.Info("no symbol data found, triggering initial import")

	job, err := s.CreateImportJob(ctx)
	if err != nil {
		return fmt.Errorf("failed to create initial symbol import job: %w", err)
	}

	slog.Info("initial symbol import job created", "job_id", job.ID)

	// Record that an import was just triggered so the scheduler's catch-up
	// doesn't create a duplicate job before this one finishes.
	if err := s.settingsService.SetTime(ctx, "symbol_data_last_update", time.Now()); err != nil {
		slog.Warn("failed to record initial symbol import time", "error", err)
	}

	go func() {
		if err := s.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("initial symbol data import failed", "error", err)
		} else {
			slog.Info("initial symbol data import completed successfully")
		}
	}()

	return nil
}

// DownloadAndImport downloads and imports symbol data from Scryfall
func (s *SymbolDataService) DownloadAndImport(ctx context.Context, jobID uint) error {
	if err := s.jobService.Start(ctx, jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	if err := s.settingsService.Set(ctx, "symbol_data_last_update_status", "in_progress"); err != nil {
		slog.Warn("failed to update status setting", "error", err)
	}

	if err := s.downloadAndImportInternal(ctx, jobID); err != nil {
		if failErr := s.jobService.Fail(ctx, jobID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "job_id", jobID, "error", failErr)
		}
		if setErr := s.settingsService.Set(ctx, "symbol_data_last_update_status", "failed"); setErr != nil {
			slog.Warn("failed to update status setting", "key", "symbol_data_last_update_status", "error", setErr)
		}
		if setErr := s.settingsService.SetTime(ctx, "symbol_data_last_update", time.Now()); setErr != nil {
			slog.Warn("failed to update time setting", "key", "symbol_data_last_update", "error", setErr)
		}
		return err
	}

	if err := s.jobService.Complete(ctx, jobID); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	if setErr := s.settingsService.Set(ctx, "symbol_data_last_update_status", "success"); setErr != nil {
		slog.Warn("failed to update status setting", "key", "symbol_data_last_update_status", "error", setErr)
	}
	if setErr := s.settingsService.SetTime(ctx, "symbol_data_last_update", time.Now()); setErr != nil {
		slog.Warn("failed to update time setting", "key", "symbol_data_last_update", "error", setErr)
	}

	return nil
}

func (s *SymbolDataService) downloadAndImportInternal(ctx context.Context, jobID uint) error {
	// Step 1: Fetch symbols from Scryfall
	s.updateJobMetadata(ctx, jobID, SymbolJobMetadata{Phase: "fetching"})

	symbols, err := s.scryfallClient.ListSymbols(ctx)
	if err != nil {
		return fmt.Errorf("failed to list symbols: %w", err)
	}

	slog.Info("downloaded symbols from scryfall", "count", len(symbols))

	metadata := SymbolJobMetadata{
		Phase:           "downloading_svgs",
		TotalSymbols:    len(symbols),
		FailureExamples: make([]string, 0),
	}

	dbSymbols := make([]*models.Symbol, 0, len(symbols))

	for i, symbol := range symbols {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("import cancelled: %w", err)
		}

		svgURI := ""
		if symbol.SVGURI != nil {
			svgURI = *symbol.SVGURI
		}

		svg, err := s.downloadSVG(ctx, svgURI)
		if err != nil {
			metadata.FailedSymbols++
			if len(metadata.FailureExamples) < 10 {
				failureMsg := fmt.Sprintf("Symbol %s: svg download failed: %v", symbol.Symbol, err)
				if len(failureMsg) > 100 {
					failureMsg = failureMsg[:97] + "..."
				}
				metadata.FailureExamples = append(metadata.FailureExamples, failureMsg)
			}
			slog.Warn("failed to download svg for symbol", "symbol", symbol.Symbol, "error", err)
			continue
		}

		metadata.SVGsDownloaded++
		dbSymbols = append(dbSymbols, &models.Symbol{
			Code:    models.NormalizeSymbolCode(symbol.Symbol),
			Symbol:  symbol.Symbol,
			English: symbol.English,
			SVG:     svg,
		})
		metadata.ProcessedSymbols = i + 1
	}

	// Step 2: Upsert all symbols to database
	s.updateJobMetadata(ctx, jobID, SymbolJobMetadata{
		Phase:            "saving",
		TotalSymbols:     len(symbols),
		ProcessedSymbols: metadata.ProcessedSymbols,
		SVGsDownloaded:   metadata.SVGsDownloaded,
		FailedSymbols:    metadata.FailedSymbols,
		FailureExamples:  metadata.FailureExamples,
	})

	if err := s.upsertSymbols(ctx, dbSymbols); err != nil {
		return fmt.Errorf("failed to save symbols: %w", err)
	}

	metadata.Phase = "completed"
	s.updateJobMetadata(ctx, jobID, metadata)

	slog.Info("symbol import completed", "total_symbols", len(symbols), "svgs_downloaded", metadata.SVGsDownloaded, "failures", metadata.FailedSymbols)

	return nil
}

func (s *SymbolDataService) downloadSVG(ctx context.Context, svgURL string) (string, error) {
	if svgURL == "" {
		return "", fmt.Errorf("empty svg uri")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svgURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download svg: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("svg download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read svg data: %w", err)
	}

	return string(data), nil
}

func (s *SymbolDataService) upsertSymbols(ctx context.Context, symbols []*models.Symbol) error {
	if len(symbols) == 0 {
		return nil
	}

	// Use UPSERT (ON CONFLICT) to handle updates keyed by the normalized code.
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"symbol", "english", "svg"}),
	}).Create(&symbols).Error
}

func (s *SymbolDataService) updateJobMetadata(ctx context.Context, jobID uint, metadata SymbolJobMetadata) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		slog.Warn("failed to marshal job metadata", "error", err)
		return
	}

	if err := s.jobService.UpdateMetadata(ctx, jobID, string(metadataJSON)); err != nil {
		slog.Warn("failed to update job metadata", "error", err)
	}
}
