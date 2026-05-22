package services

import (
	"backend/models"
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBannerServiceTest(t *testing.T) (*BannerService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return NewBannerService(NewJobService(db)), db
}

func createBannerTestJob(t *testing.T, db *gorm.DB, jobType models.JobType, status models.JobStatus) *models.Job {
	t.Helper()

	job := &models.Job{Type: jobType, Status: status, Metadata: "{}"}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	return job
}

func TestBannerService_GetActive_Empty(t *testing.T) {
	service, _ := setupBannerServiceTest(t)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 0 {
		t.Errorf("expected 0 banners, got %d", len(banners))
	}
}

func TestBannerService_GetActive_InProgressSync(t *testing.T) {
	service, db := setupBannerServiceTest(t)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusInProgress)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 1 {
		t.Fatalf("expected 1 banner, got %d", len(banners))
	}

	banner := banners[0]
	if banner.Severity != models.BannerSeverityInfo {
		t.Errorf("expected info severity, got %s", banner.Severity)
	}
	if banner.Dismissible {
		t.Error("expected in-progress banner to be non-dismissible")
	}
	if banner.ID != "sync:bulk_data_import" {
		t.Errorf("unexpected banner id: %s", banner.ID)
	}
	if banner.Message == "" {
		t.Error("expected a non-empty message")
	}
}

func TestBannerService_GetActive_DedupesInProgressByType(t *testing.T) {
	service, db := setupBannerServiceTest(t)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusInProgress)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusPending)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 1 {
		t.Fatalf("expected 1 deduped banner, got %d", len(banners))
	}
}

func TestBannerService_GetActive_FailedLastJob(t *testing.T) {
	service, db := setupBannerServiceTest(t)
	job := createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusFailed)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 1 {
		t.Fatalf("expected 1 banner, got %d", len(banners))
	}

	banner := banners[0]
	if banner.Severity != models.BannerSeverityWarning {
		t.Errorf("expected warning severity, got %s", banner.Severity)
	}
	if !banner.Dismissible {
		t.Error("expected failed-job banner to be dismissible")
	}

	wantID := fmt.Sprintf("job-failed:%s:%d", models.JobTypeBulkDataImport, job.ID)
	if banner.ID != wantID {
		t.Errorf("expected banner id %s, got %s", wantID, banner.ID)
	}
}

func TestBannerService_GetActive_FailedThenSucceeded(t *testing.T) {
	service, db := setupBannerServiceTest(t)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusFailed)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusCompleted)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 0 {
		t.Errorf("expected 0 banners once a later run succeeded, got %d", len(banners))
	}
}

func TestBannerService_GetActive_FailedClearedWhenNewerJobSharesTimestamp(t *testing.T) {
	service, db := setupBannerServiceTest(t)

	// Two jobs with an identical created_at: only the id tie-breaker decides
	// which one is "last". The newer (higher-id) success must win, clearing
	// the failed banner.
	ts := time.Now()
	failed := &models.Job{Type: models.JobTypeBulkDataImport, Status: models.JobStatusFailed, Metadata: "{}"}
	failed.CreatedAt = ts
	if err := db.Create(failed).Error; err != nil {
		t.Fatalf("failed to create failed job: %v", err)
	}
	succeeded := &models.Job{Type: models.JobTypeBulkDataImport, Status: models.JobStatusCompleted, Metadata: "{}"}
	succeeded.CreatedAt = ts
	if err := db.Create(succeeded).Error; err != nil {
		t.Fatalf("failed to create succeeded job: %v", err)
	}

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 0 {
		t.Errorf("expected 0 banners when the newer same-timestamp job succeeded, got %d", len(banners))
	}
}

func TestBannerService_GetActive_BothBannerKinds(t *testing.T) {
	service, db := setupBannerServiceTest(t)
	createBannerTestJob(t, db, models.JobTypeBulkDataImport, models.JobStatusInProgress)
	createBannerTestJob(t, db, models.JobTypeSetDataImport, models.JobStatusFailed)

	banners, err := service.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if len(banners) != 2 {
		t.Fatalf("expected 2 banners, got %d", len(banners))
	}
}
