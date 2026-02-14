package services

import (
	"backend/models"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJobServiceTest(t *testing.T) (*JobService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return NewJobService(db), db
}

// Create tests

func TestJobService_Create_Success(t *testing.T) {
	service, _ := setupJobServiceTest(t)

	ctx := context.Background()

	job, err := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if job.ID == 0 {
		t.Error("expected job ID to be set")
	}

	if job.Type != models.JobTypeBulkDataImport {
		t.Errorf("expected type %s, got %s", models.JobTypeBulkDataImport, job.Type)
	}

	if job.Status != models.JobStatusPending {
		t.Errorf("expected status %s, got %s", models.JobStatusPending, job.Status)
	}

	if job.Metadata != "{}" {
		t.Errorf("expected metadata '{}', got '%s'", job.Metadata)
	}
}

func TestJobService_Create_WithMetadata(t *testing.T) {
	service, _ := setupJobServiceTest(t)

	ctx := context.Background()

	metadata := `{"total_cards": 100, "phase": "importing"}`
	job, err := service.Create(ctx, models.JobTypeBulkDataImport, metadata)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if job.Metadata != metadata {
		t.Errorf("expected metadata %s, got %s", metadata, job.Metadata)
	}
}

// Get tests

func TestJobService_Get_Success(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	createdJob, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	job, err := service.Get(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if job.ID != createdJob.ID {
		t.Errorf("expected ID %d, got %d", createdJob.ID, job.ID)
	}
}

func TestJobService_Get_NotFound(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	job, err := service.Get(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent job")
	}

	if job != nil {
		t.Error("expected nil job for non-existent ID")
	}
}

// List tests

func TestJobService_List_Empty(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	jobs, total, err := service.List(ctx, 1, 10, nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestJobService_List_WithJobs(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create multiple jobs
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	jobs, total, err := service.List(ctx, 1, 10, nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestJobService_List_Pagination(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	}

	// Get page 2 with page size 2
	jobs, total, err := service.List(ctx, 2, 2, nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs on page 2, got %d", len(jobs))
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}

func TestJobService_List_FilterByStatus(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create jobs with different statuses
	job1, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	job2, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	// Mark some as completed
	service.Complete(ctx, job1.ID)
	service.Complete(ctx, job2.ID)

	// Filter by completed status
	status := models.JobStatusCompleted
	jobs, total, err := service.List(ctx, 1, 10, nil, &status)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 completed jobs, got %d", len(jobs))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	for _, job := range jobs {
		if job.Status != models.JobStatusCompleted {
			t.Errorf("expected status %s, got %s", models.JobStatusCompleted, job.Status)
		}
	}
}

func TestJobService_List_FilterByType(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create jobs of different types
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	// Filter by type
	jobType := models.JobTypeBulkDataImport
	jobs, total, err := service.List(ctx, 1, 10, &jobType, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 bulk import jobs, got %d", len(jobs))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

// Start tests

func TestJobService_Start_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	job, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	err := service.Start(ctx, job.ID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify in database
	var updated models.Job
	db.First(&updated, job.ID)

	if updated.Status != models.JobStatusInProgress {
		t.Errorf("expected status %s, got %s", models.JobStatusInProgress, updated.Status)
	}

	if updated.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
}

func TestJobService_Start_NonExistent(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	err := service.Start(ctx, 999)
	// Should not error, just update nothing
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify no job was created
	var count int64
	db.Model(&models.Job{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 jobs, got %d", count)
	}
}

// Complete tests

func TestJobService_Complete_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	job, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Start(ctx, job.ID)

	err := service.Complete(ctx, job.ID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify in database
	var updated models.Job
	db.First(&updated, job.ID)

	if updated.Status != models.JobStatusCompleted {
		t.Errorf("expected status %s, got %s", models.JobStatusCompleted, updated.Status)
	}

	if updated.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// Fail tests

func TestJobService_Fail_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	job, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Start(ctx, job.ID)

	errorMsg := "Download failed: network timeout"
	err := service.Fail(ctx, job.ID, errorMsg)
	if err != nil {
		t.Fatalf("Fail failed: %v", err)
	}

	// Verify in database
	var updated models.Job
	db.First(&updated, job.ID)

	if updated.Status != models.JobStatusFailed {
		t.Errorf("expected status %s, got %s", models.JobStatusFailed, updated.Status)
	}

	if updated.Error != errorMsg {
		t.Errorf("expected error '%s', got '%s'", errorMsg, updated.Error)
	}

	if updated.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// UpdateMetadata tests

func TestJobService_UpdateMetadata_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	job, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	newMetadata := `{"total_cards": 500, "processed_cards": 250}`
	err := service.UpdateMetadata(ctx, job.ID, newMetadata)
	if err != nil {
		t.Fatalf("UpdateMetadata failed: %v", err)
	}

	// Verify in database
	var updated models.Job
	db.First(&updated, job.ID)

	if updated.Metadata != newMetadata {
		t.Errorf("expected metadata %s, got %s", newMetadata, updated.Metadata)
	}
}

// CleanupOldJobs tests

func TestJobService_CleanupOldJobs_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)

	// Create an old job
	oldJob := &models.Job{
		Type:   models.JobTypeBulkDataImport,
		Status: models.JobStatusCompleted,
	}
	db.Create(oldJob)

	// Manually update created_at to be 31 days old
	db.Model(oldJob).Update("created_at", time.Now().AddDate(0, 0, -31))

	// Create a recent job
	ctx := context.Background()
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	// Cleanup jobs older than 30 days
	deleted, err := service.CleanupOldJobs(ctx, 30)
	if err != nil {
		t.Fatalf("CleanupOldJobs failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted job, got %d", deleted)
	}

	// Verify only recent job remains
	var count int64
	db.Model(&models.Job{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 remaining job, got %d", count)
	}
}

func TestJobService_CleanupOldJobs_NoOldJobs(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create only recent jobs
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Create(ctx, models.JobTypeBulkDataImport, "{}")

	deleted, err := service.CleanupOldJobs(ctx, 30)
	if err != nil {
		t.Fatalf("CleanupOldJobs failed: %v", err)
	}

	if deleted != 0 {
		t.Errorf("expected 0 deleted jobs, got %d", deleted)
	}
}

// CancelStaleJobs tests

func TestJobService_CancelStaleJobs_Success(t *testing.T) {
	service, db := setupJobServiceTest(t)
	ctx := context.Background()

	// Create jobs in various states
	pendingJob, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	inProgressJob, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Start(ctx, inProgressJob.ID)
	completedJob, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Complete(ctx, completedJob.ID)

	// Cancel stale jobs
	cancelled, err := service.CancelStaleJobs(ctx)
	if err != nil {
		t.Fatalf("CancelStaleJobs failed: %v", err)
	}

	if cancelled != 2 {
		t.Errorf("expected 2 cancelled jobs, got %d", cancelled)
	}

	// Verify pending job was cancelled
	var updatedPending models.Job
	db.First(&updatedPending, pendingJob.ID)
	if updatedPending.Status != models.JobStatusCancelled {
		t.Errorf("expected pending job to be cancelled, got %s", updatedPending.Status)
	}

	// Verify in-progress job was cancelled
	var updatedInProgress models.Job
	db.First(&updatedInProgress, inProgressJob.ID)
	if updatedInProgress.Status != models.JobStatusCancelled {
		t.Errorf("expected in-progress job to be cancelled, got %s", updatedInProgress.Status)
	}

	// Verify completed job was NOT cancelled
	var updatedCompleted models.Job
	db.First(&updatedCompleted, completedJob.ID)
	if updatedCompleted.Status != models.JobStatusCompleted {
		t.Errorf("expected completed job to remain completed, got %s", updatedCompleted.Status)
	}
}

func TestJobService_CancelStaleJobs_NoStaleJobs(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create only completed jobs
	job, _ := service.Create(ctx, models.JobTypeBulkDataImport, "{}")
	service.Complete(ctx, job.ID)

	cancelled, err := service.CancelStaleJobs(ctx)
	if err != nil {
		t.Fatalf("CancelStaleJobs failed: %v", err)
	}

	if cancelled != 0 {
		t.Errorf("expected 0 cancelled jobs, got %d", cancelled)
	}
}

// GetLastJobByType tests

func TestJobService_GetLastJobByType_Success(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// Create multiple jobs of same type
	service.Create(ctx, models.JobTypeBulkDataImport, `{"order": 1}`)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	service.Create(ctx, models.JobTypeBulkDataImport, `{"order": 2}`)
	time.Sleep(10 * time.Millisecond)
	lastJob, _ := service.Create(ctx, models.JobTypeBulkDataImport, `{"order": 3}`)

	// Get last job
	retrieved, err := service.GetLastJobByType(ctx, models.JobTypeBulkDataImport)
	if err != nil {
		t.Fatalf("GetLastJobByType failed: %v", err)
	}

	if retrieved.ID != lastJob.ID {
		t.Errorf("expected last job ID %d, got %d", lastJob.ID, retrieved.ID)
	}

	if retrieved.Metadata != `{"order": 3}` {
		t.Errorf("expected last job metadata, got %s", retrieved.Metadata)
	}
}

func TestJobService_GetLastJobByType_NotFound(t *testing.T) {
	service, _ := setupJobServiceTest(t)
	ctx := context.Background()

	// No jobs exist
	job, err := service.GetLastJobByType(ctx, models.JobTypeBulkDataImport)
	if err != nil {
		t.Fatalf("GetLastJobByType failed: %v", err)
	}

	if job != nil {
		t.Error("expected nil job when none exist")
	}
}
