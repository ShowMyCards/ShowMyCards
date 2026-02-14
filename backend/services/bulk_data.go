// Package services contains business logic for managing cards, jobs, bulk imports, and settings.
// It provides services for background processing and external API integration.
package services

import (
	"backend/models"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// BulkDataBatchSize is the number of cards processed in each database insert
	BulkDataBatchSize = 1000

	// BulkDataMaxFailureRate is the maximum acceptable failure rate for bulk imports
	// Real-world Scryfall data has ~0.1-0.5% failures due to incomplete card data
	// If failures exceed this threshold, the job is marked as failed
	BulkDataMaxFailureRate = 0.05 // 5%

	// BulkDataTypeAllCards is the Scryfall bulk data type for all cards
	BulkDataTypeAllCards = "all_cards"
)

// BulkDataService handles bulk data download and import
type BulkDataService struct {
	db              *gorm.DB
	jobService      *JobService
	settingsService *SettingsService
	httpClient      *http.Client // short-lived API requests
	downloadClient  *http.Client // long-running bulk downloads
}

// NewBulkDataService creates a new bulk data service
func NewBulkDataService(db *gorm.DB, jobService *JobService, settingsService *SettingsService) *BulkDataService {
	return &BulkDataService{
		db:              db,
		jobService:      jobService,
		settingsService: settingsService,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		downloadClient:  &http.Client{Timeout: 30 * time.Minute},
	}
}

// CreateImportJob creates a new job for bulk data import
func (s *BulkDataService) CreateImportJob(ctx context.Context) (*models.Job, error) {
	return s.jobService.Create(ctx, models.JobTypeBulkDataImport, "{}")
}

// HasBulkData checks if bulk card data exists in the database
func (s *BulkDataService) HasBulkData() (bool, error) {
	var count int64
	if err := s.db.Model(&models.Card{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// TriggerInitialImport triggers an initial bulk data import if no data exists
func (s *BulkDataService) TriggerInitialImport(ctx context.Context) error {
	// Check if bulk data already exists
	hasData, err := s.HasBulkData()
	if err != nil {
		return fmt.Errorf("failed to check for existing bulk data: %w", err)
	}

	if hasData {
		slog.Info("bulk data already exists, skipping initial import")
		return nil
	}

	slog.Info("no bulk data found, triggering initial import")

	// Create and start import job
	job, err := s.CreateImportJob(ctx)
	if err != nil {
		return fmt.Errorf("failed to create initial import job: %w", err)
	}

	slog.Info("initial import job created", "job_id", job.ID)

	// Run import in background with context
	go func() {
		if err := s.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("initial bulk data import failed", "error", err)
		} else {
			slog.Info("initial bulk data import completed successfully")
		}
	}()

	return nil
}

// BulkDataInfo represents a bulk data file from Scryfall's API
type BulkDataInfo struct {
	Object      string    `json:"object"`
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	UpdatedAt   time.Time `json:"updated_at"`
	URI         string    `json:"uri"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Size        int64     `json:"size"`
	DownloadURI string    `json:"download_uri"`
}

// BulkDataListResponse represents the response from Scryfall's bulk data list endpoint
type BulkDataListResponse struct {
	Object  string         `json:"object"`
	HasMore bool           `json:"has_more"`
	Data    []BulkDataInfo `json:"data"`
}

// No longer need custom ScryfallCard type - using scryfall.Card from library

// JobMetadata represents the metadata stored in job.Metadata field
type JobMetadata struct {
	TotalCards      int      `json:"total_cards"`
	ProcessedCards  int      `json:"processed_cards"`
	FailedCards     int      `json:"failed_cards"`
	FailureExamples []string `json:"failure_examples"` // First 10 failures, max 100 chars each
	Phase           string   `json:"phase"`            // "downloading", "importing", "completed"
}

// DownloadAndImport downloads and imports bulk data from Scryfall with context support
func (s *BulkDataService) DownloadAndImport(ctx context.Context, jobID uint) error {
	// Update job status to in progress
	if err := s.jobService.Start(ctx, jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	// Update settings to show import is in progress
	if err := s.settingsService.Set(ctx, "bulk_data_last_update_status", "in_progress"); err != nil {
		slog.Warn("failed to update status setting", "error", err)
	}

	// Perform the download and import with context
	if err := s.downloadAndImportInternal(ctx, jobID); err != nil {
		// Mark job as failed
		if failErr := s.jobService.Fail(ctx, jobID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "job_id", jobID, "error", failErr)
		}
		// Update settings to show failure
		if setErr := s.settingsService.Set(ctx, "bulk_data_last_update_status", "failed"); setErr != nil {
			slog.Warn("failed to update status setting", "key", "bulk_data_last_update_status", "error", setErr)
		}
		if setErr := s.settingsService.SetTime(ctx, "bulk_data_last_update", time.Now()); setErr != nil {
			slog.Warn("failed to update time setting", "key", "bulk_data_last_update", "error", setErr)
		}
		return err
	}

	// Mark job as completed
	if err := s.jobService.Complete(ctx, jobID); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	// Update settings to show success
	if setErr := s.settingsService.Set(ctx, "bulk_data_last_update_status", "success"); setErr != nil {
		slog.Warn("failed to update status setting", "key", "bulk_data_last_update_status", "error", setErr)
	}
	if setErr := s.settingsService.SetTime(ctx, "bulk_data_last_update", time.Now()); setErr != nil {
		slog.Warn("failed to update time setting", "key", "bulk_data_last_update", "error", setErr)
	}

	return nil
}

func (s *BulkDataService) downloadAndImportInternal(ctx context.Context, jobID uint) error {
	// Step 1: Fetch bulk data list
	s.updateJobMetadata(ctx, jobID, JobMetadata{Phase: "fetching_list"})

	bulkDataURL, err := s.settingsService.Get(ctx, "bulk_data_url")
	if err != nil {
		return fmt.Errorf("failed to get bulk data URL setting: %w", err)
	}

	downloadURI, err := s.fetchBulkDataDownloadURI(ctx, bulkDataURL)
	if err != nil {
		return fmt.Errorf("failed to fetch bulk data list: %w", err)
	}

	// Step 2: Download and import bulk data file in streaming fashion (UPSERT strategy)
	s.updateJobMetadata(ctx, jobID, JobMetadata{Phase: "downloading_and_importing"})

	totalProcessed := 0
	totalFailed := 0
	allFailureExamples := make([]string, 0, 10)

	err = s.downloadBulkDataStream(ctx, downloadURI, BulkDataBatchSize, func(batch []scryfall.Card) error {
		// Check context before processing batch
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("import cancelled: %w", err)
		}

		// Import this batch with context
		batchResult, err := s.importCardsBatch(ctx, batch)
		if err != nil {
			return err
		}

		totalProcessed += batchResult.SuccessCards
		totalFailed += batchResult.FailedCards

		// Aggregate failure examples (keep first 10 total)
		for _, example := range batchResult.FailureExamples {
			if len(allFailureExamples) < 10 {
				allFailureExamples = append(allFailureExamples, example)
			}
		}

		// Update progress
		s.updateJobMetadata(ctx, jobID, JobMetadata{
			Phase:           "downloading_and_importing",
			ProcessedCards:  totalProcessed,
			FailedCards:     totalFailed,
			FailureExamples: allFailureExamples,
		})

		slog.Info("import progress", "processed", totalProcessed, "failed", totalFailed)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to download and import bulk data: %w", err)
	}

	// Step 3: Check failure rate and determine job outcome
	totalCards := totalProcessed + totalFailed
	failureRate := 0.0
	if totalCards > 0 {
		failureRate = float64(totalFailed) / float64(totalCards)
	}

	s.updateJobMetadata(ctx, jobID, JobMetadata{
		Phase:           "completed",
		TotalCards:      totalCards,
		ProcessedCards:  totalProcessed,
		FailedCards:     totalFailed,
		FailureExamples: allFailureExamples,
	})

	// If failure rate exceeds threshold, return error to mark job as failed
	if failureRate > BulkDataMaxFailureRate {
		return fmt.Errorf("bulk import failed: %d/%d cards failed (%.2f%% > %.2f%% threshold)",
			totalFailed, totalCards, failureRate*100, BulkDataMaxFailureRate*100)
	}

	// If there were failures but below threshold, log warning
	if totalFailed > 0 {
		slog.Warn("bulk import completed with warnings", "failed", totalFailed, "total", totalCards, "failure_rate_pct", fmt.Sprintf("%.2f", failureRate*100))
	}

	return nil
}

func (s *BulkDataService) fetchBulkDataDownloadURI(ctx context.Context, bulkDataURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", bulkDataURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch bulk data list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bulk data list returned status %d", resp.StatusCode)
	}

	var bulkDataList BulkDataListResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkDataList); err != nil {
		return "", fmt.Errorf("failed to decode bulk data list: %w", err)
	}

	// Find the "all_cards" bulk data
	for _, bulkData := range bulkDataList.Data {
		if bulkData.Type == BulkDataTypeAllCards {
			return bulkData.DownloadURI, nil
		}
	}

	return "", fmt.Errorf("%s bulk data not found", BulkDataTypeAllCards)
}

// downloadBulkDataStream downloads and streams bulk data, calling the callback
// for each batch of cards. This avoids loading the entire file into memory.
func (s *BulkDataService) downloadBulkDataStream(ctx context.Context, downloadURI string, batchSize int, callback func([]scryfall.Card) error) error {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download bulk data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bulk data download returned status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)

	// Read opening bracket of array
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read JSON array start: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected JSON array, got %v", token)
	}

	batch := make([]scryfall.Card, 0, batchSize)

	// Process cards one at a time
	for decoder.More() {
		// Check context periodically
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("download cancelled: %w", err)
		}

		var card scryfall.Card
		if err := decoder.Decode(&card); err != nil {
			return fmt.Errorf("failed to decode card: %w", err)
		}

		batch = append(batch, card)

		// Process batch when it reaches batchSize
		if len(batch) >= batchSize {
			if err := callback(batch); err != nil {
				return fmt.Errorf("failed to process batch: %w", err)
			}
			batch = make([]scryfall.Card, 0, batchSize)
		}
	}

	// Process remaining cards
	if len(batch) > 0 {
		if err := callback(batch); err != nil {
			return fmt.Errorf("failed to process final batch: %w", err)
		}
	}

	// Read closing bracket
	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read JSON array end: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != ']' {
		return fmt.Errorf("expected JSON array end, got %v", token)
	}

	return nil
}

// BatchImportResult contains statistics about a batch import operation
type BatchImportResult struct {
	TotalCards      int
	SuccessCards    int
	FailedCards     int
	FailureExamples []string // First 10 failures, max 100 chars each
}

// importCardsBatch imports a single batch of cards into the database
// Uses UPSERT (ON CONFLICT) to skip unchanged records for better performance
// Returns statistics about the import including failure tracking
func (s *BulkDataService) importCardsBatch(ctx context.Context, cards []scryfall.Card) (BatchImportResult, error) {
	result := BatchImportResult{
		TotalCards:      len(cards),
		FailureExamples: make([]string, 0),
	}

	if len(cards) == 0 {
		return result, nil
	}

	// Check context before expensive conversion loop
	if err := ctx.Err(); err != nil {
		return result, fmt.Errorf("batch import cancelled: %w", err)
	}

	// Convert scryfall.Card to our Card model
	dbCards := make([]*models.Card, 0, len(cards))
	for _, scryfallCard := range cards {
		card, err := models.FromScryfallCard(scryfallCard)
		if err != nil {
			result.FailedCards++

			// Store failure example (first 10, truncated to 100 chars)
			if len(result.FailureExamples) < 10 {
				failureMsg := fmt.Sprintf("Card %s (%s): %v", scryfallCard.ID, scryfallCard.Name, err)
				if len(failureMsg) > 100 {
					failureMsg = failureMsg[:97] + "..."
				}
				result.FailureExamples = append(result.FailureExamples, failureMsg)
			}

			slog.Warn("failed to convert card", "scryfall_id", scryfallCard.ID, "name", scryfallCard.Name, "error", err)
			continue
		}
		dbCards = append(dbCards, card)
	}

	if len(dbCards) == 0 {
		return result, fmt.Errorf("no valid cards to import in batch")
	}

	// Use UPSERT to insert or update cards
	// SQLite syntax: INSERT ... ON CONFLICT(scryfall_id) DO UPDATE SET ...
	// This skips unchanged records automatically (no UPDATE if values match)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scryfall_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"raw_json", "oracle_id"}),
	}).Create(&dbCards).Error; err != nil {
		firstID := ""
		lastName := ""
		if len(dbCards) > 0 {
			firstID = dbCards[0].ScryfallID
			lastName = dbCards[len(dbCards)-1].Name
		}
		return result, fmt.Errorf("failed to insert batch of %d cards (first: %s, last: %s): %w",
			len(dbCards), firstID, lastName, err)
	}

	result.SuccessCards = len(dbCards)
	return result, nil
}

func (s *BulkDataService) updateJobMetadata(ctx context.Context, jobID uint, metadata JobMetadata) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		slog.Warn("failed to marshal job metadata", "error", err)
		return
	}

	if err := s.jobService.UpdateMetadata(ctx, jobID, string(metadataJSON)); err != nil {
		slog.Warn("failed to update job metadata", "error", err)
	}
}
