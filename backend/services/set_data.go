package services

import (
	"backend/models"
	"backend/version"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
)

// SetLister retrieves the Magic set catalog from an upstream source (Scryfall).
type SetLister interface {
	ListSets(ctx context.Context) ([]scryfall.Set, error)
}

// SetDataService handles set data download and import
type SetDataService struct {
	db              *gorm.DB
	jobService      *JobService
	settingsService *SettingsService
	scryfallClient  SetLister
	dataDir         string
	httpClient      *http.Client
}

// NewSetDataService creates a new set data service
func NewSetDataService(db *gorm.DB, jobService *JobService, settingsService *SettingsService, scryfallClient SetLister, dataDir string) *SetDataService {
	return &SetDataService{
		db:              db,
		jobService:      jobService,
		settingsService: settingsService,
		scryfallClient:  scryfallClient,
		dataDir:         dataDir,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

// SetJobMetadata represents the metadata stored in job.Metadata field for set imports
type SetJobMetadata struct {
	Phase           string   `json:"phase"`
	TotalSets       int      `json:"total_sets"`
	ProcessedSets   int      `json:"processed_sets"`
	IconsDownloaded int      `json:"icons_downloaded"`
	IconsSkipped    int      `json:"icons_skipped"`
	FailedSets      int      `json:"failed_sets"`
	FailureExamples []string `json:"failure_examples"`
}

// CreateImportJob creates a new job for set data import
func (s *SetDataService) CreateImportJob(ctx context.Context) (*models.Job, error) {
	return s.jobService.Create(ctx, models.JobTypeSetDataImport, "{}")
}

// HasSetData checks if set data exists in the database
func (s *SetDataService) HasSetData() (bool, error) {
	var count int64
	if err := s.db.Model(&models.Set{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// TriggerInitialImport triggers an initial set data import if no data exists
func (s *SetDataService) TriggerInitialImport(ctx context.Context) error {
	hasData, err := s.HasSetData()
	if err != nil {
		return fmt.Errorf("failed to check for existing set data: %w", err)
	}

	if hasData {
		slog.Info("set data already exists, skipping initial import")
		return nil
	}

	slog.Info("no set data found, triggering initial import")

	job, err := s.CreateImportJob(ctx)
	if err != nil {
		return fmt.Errorf("failed to create initial set import job: %w", err)
	}

	slog.Info("initial set import job created", "job_id", job.ID)

	// Record that an import was just triggered so the scheduler's catch-up
	// doesn't create a duplicate job before this one finishes.
	if err := s.settingsService.SetTime(ctx, "set_data_update_last_run", time.Now()); err != nil {
		slog.Warn("failed to record initial set import time", "error", err)
	}

	go func() {
		if err := s.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("initial set data import failed", "error", err)
		} else {
			slog.Info("initial set data import completed successfully")
		}
	}()

	return nil
}

// DownloadAndImport downloads and imports set data from Scryfall
func (s *SetDataService) DownloadAndImport(ctx context.Context, jobID uint) error {
	if err := s.jobService.Start(ctx, jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	if err := s.settingsService.Set(ctx, "set_data_update_last_status", "in_progress"); err != nil {
		slog.Warn("failed to update status setting", "error", err)
	}

	if err := s.downloadAndImportInternal(ctx, jobID); err != nil {
		if failErr := s.jobService.Fail(ctx, jobID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "job_id", jobID, "error", failErr)
		}
		if setErr := s.settingsService.Set(ctx, "set_data_update_last_status", "failed"); setErr != nil {
			slog.Warn("failed to update status setting", "key", "set_data_update_last_status", "error", setErr)
		}
		if setErr := s.settingsService.SetTime(ctx, "set_data_update_last_run", time.Now()); setErr != nil {
			slog.Warn("failed to update time setting", "key", "set_data_update_last_run", "error", setErr)
		}
		return err
	}

	if err := s.jobService.Complete(ctx, jobID); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	if setErr := s.settingsService.Set(ctx, "set_data_update_last_status", "success"); setErr != nil {
		slog.Warn("failed to update status setting", "key", "set_data_update_last_status", "error", setErr)
	}
	if setErr := s.settingsService.SetTime(ctx, "set_data_update_last_run", time.Now()); setErr != nil {
		slog.Warn("failed to update time setting", "key", "set_data_update_last_run", "error", setErr)
	}

	return nil
}

func (s *SetDataService) downloadAndImportInternal(ctx context.Context, jobID uint) error {
	// Step 1: Fetch sets from Scryfall
	s.updateJobMetadata(ctx, jobID, SetJobMetadata{Phase: "fetching"})

	sets, err := s.downloadSets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sets: %w", err)
	}

	slog.Info("downloaded sets from scryfall", "count", len(sets))

	// Step 2: Ensure icon directory exists
	iconDir := filepath.Join(s.dataDir, "set-icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return fmt.Errorf("failed to create icon directory: %w", err)
	}

	// Step 3: Download icons and import sets
	s.updateJobMetadata(ctx, jobID, SetJobMetadata{
		Phase:     "downloading_icons",
		TotalSets: len(sets),
	})

	metadata := SetJobMetadata{
		Phase:           "importing",
		TotalSets:       len(sets),
		FailureExamples: make([]string, 0),
	}

	dbSets := make([]*models.Set, 0, len(sets))

	for i, set := range sets {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("import cancelled: %w", err)
		}

		// Download icon if needed
		iconFilename, downloaded, err := s.downloadIconIfNeeded(ctx, set.IconSVGURI, set.Code)
		if err != nil {
			metadata.FailedSets++
			if len(metadata.FailureExamples) < 10 {
				failureMsg := fmt.Sprintf("Set %s: icon download failed: %v", set.Code, err)
				if len(failureMsg) > 100 {
					failureMsg = failureMsg[:97] + "..."
				}
				metadata.FailureExamples = append(metadata.FailureExamples, failureMsg)
			}
			slog.Warn("failed to download icon for set", "set_code", set.Code, "error", err)
			iconFilename = "" // Continue without icon
		}

		if downloaded {
			metadata.IconsDownloaded++
		} else if iconFilename != "" {
			metadata.IconsSkipped++
		}

		// Convert to database model
		dbSet := s.scryfallSetToModel(set, iconFilename)
		dbSets = append(dbSets, dbSet)
		metadata.ProcessedSets = i + 1

		// Update progress every 50 sets
		if (i+1)%50 == 0 {
			s.updateJobMetadata(ctx, jobID, metadata)
			slog.Info("set import progress", "processed", i+1, "total", len(sets))
		}
	}

	// Step 4: Replace sets in database
	s.updateJobMetadata(ctx, jobID, SetJobMetadata{
		Phase:           "saving",
		TotalSets:       len(sets),
		ProcessedSets:   metadata.ProcessedSets,
		IconsDownloaded: metadata.IconsDownloaded,
		IconsSkipped:    metadata.IconsSkipped,
		FailedSets:      metadata.FailedSets,
		FailureExamples: metadata.FailureExamples,
	})

	if err := s.replaceSets(ctx, dbSets); err != nil {
		return fmt.Errorf("failed to save sets: %w", err)
	}

	// Final metadata update
	metadata.Phase = "completed"
	s.updateJobMetadata(ctx, jobID, metadata)

	slog.Info("set import completed", "total_sets", len(sets), "icons_downloaded", metadata.IconsDownloaded, "icons_skipped", metadata.IconsSkipped, "failures", metadata.FailedSets)

	return nil
}

func (s *SetDataService) downloadSets(ctx context.Context) ([]scryfall.Set, error) {
	sets, err := s.scryfallClient.ListSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sets: %w", err)
	}

	return sets, nil
}

func (s *SetDataService) downloadIconIfNeeded(ctx context.Context, iconURL, setCode string) (string, bool, error) {
	if iconURL == "" {
		return "", false, nil
	}

	filename := setCode + ".svg"
	iconPath := filepath.Join(s.dataDir, "set-icons", filename)

	// Check if icon already exists
	if _, err := os.Stat(iconPath); err == nil {
		return filename, false, nil // Already exists
	}

	// Download the icon
	req, err := http.NewRequestWithContext(ctx, "GET", iconURL, nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("failed to download icon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("icon download returned status %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read icon data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(iconPath, data, 0644); err != nil {
		return "", false, fmt.Errorf("failed to write icon file: %w", err)
	}

	return filename, true, nil
}

func (s *SetDataService) scryfallSetToModel(set scryfall.Set, iconFilename string) *models.Set {
	var releasedAt *string
	if set.ReleasedAt != nil {
		dateStr := set.ReleasedAt.String()
		releasedAt = &dateStr
	}

	return &models.Set{
		ScryfallID:    set.ID,
		Code:          set.Code,
		Name:          set.Name,
		SetType:       string(set.SetType),
		ReleasedAt:    releasedAt,
		CardCount:     set.CardCount,
		Digital:       set.Digital,
		IconFilename:  iconFilename,
		ParentSetCode: set.ParentSetCode,
	}
}

const setInsertBatchSize = 100

func (s *SetDataService) replaceSets(ctx context.Context, sets []*models.Set) error {
	if len(sets) == 0 {
		return nil
	}

	// Replace the full set catalog atomically in a transaction. The sets table
	// enforces uniqueness on both scryfall_id (PK) and code (uniqueIndex), so an
	// ON CONFLICT (scryfall_id) upsert fails whenever Scryfall renames a set's
	// code and assigns the old code to another set, swaps codes, or reuses a
	// removed set's code. Scryfall's /sets endpoint returns the complete catalog
	// and no other table references sets via foreign key, so clearing and
	// re-inserting in one transaction avoids transient unique collisions, prunes
	// removed sets, and rolls back cleanly on error.
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Set{}).Error; err != nil {
			return fmt.Errorf("clearing existing sets: %w", err)
		}
		if err := tx.CreateInBatches(sets, setInsertBatchSize).Error; err != nil {
			return fmt.Errorf("inserting sets: %w", err)
		}
		return nil
	})
}

func (s *SetDataService) updateJobMetadata(ctx context.Context, jobID uint, metadata SetJobMetadata) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		slog.Warn("failed to marshal job metadata", "error", err)
		return
	}

	if err := s.jobService.UpdateMetadata(ctx, jobID, string(metadataJSON)); err != nil {
		slog.Warn("failed to update job metadata", "error", err)
	}
}
