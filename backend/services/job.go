package services

import (
	"backend/models"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JobService handles job operations
type JobService struct {
	db *gorm.DB
}

// NewJobService creates a new job service
func NewJobService(db *gorm.DB) *JobService {
	return &JobService{db: db}
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, jobType models.JobType, metadata string) (*models.Job, error) {
	job := &models.Job{
		Type:     jobType,
		Status:   models.JobStatusPending,
		Metadata: metadata,
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("creating %s job: %w", jobType, err)
	}

	return job, nil
}

// Get retrieves a job by ID
func (s *JobService) Get(ctx context.Context, id uint) (*models.Job, error) {
	var job models.Job
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, fmt.Errorf("getting job %d: %w", id, err)
	}
	return &job, nil
}

// List retrieves jobs with pagination and optional filtering
func (s *JobService) List(ctx context.Context, page, pageSize int, jobType *models.JobType, status *models.JobStatus) ([]models.Job, int64, error) {
	var jobs []models.Job
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Job{})

	// Apply filters
	if jobType != nil {
		query = query.Where("type = ?", *jobType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting jobs: %w", err)
	}

	// Get paginated results, ordered by created_at descending (newest first)
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, fmt.Errorf("listing jobs: %w", err)
	}

	return jobs, total, nil
}

// UpdateStatus updates a job's status
func (s *JobService) UpdateStatus(ctx context.Context, id uint, status models.JobStatus) error {
	if err := s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("updating job %d status to %s: %w", id, status, err)
	}
	return nil
}

// Start marks a job as in progress
func (s *JobService) Start(ctx context.Context, id uint) error {
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     models.JobStatusInProgress,
		"started_at": now,
	}).Error; err != nil {
		return fmt.Errorf("starting job %d: %w", id, err)
	}
	return nil
}

// Complete marks a job as completed
func (s *JobService) Complete(ctx context.Context, id uint) error {
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.JobStatusCompleted,
		"completed_at": now,
	}).Error; err != nil {
		return fmt.Errorf("completing job %d: %w", id, err)
	}
	return nil
}

// Fail marks a job as failed with an error message
func (s *JobService) Fail(ctx context.Context, id uint, errorMessage string) error {
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.JobStatusFailed,
		"completed_at": now,
		"error":        errorMessage,
	}).Error; err != nil {
		return fmt.Errorf("failing job %d: %w", id, err)
	}
	return nil
}

// CleanupOldJobs deletes jobs older than the specified retention period
func (s *JobService) CleanupOldJobs(ctx context.Context, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := s.db.WithContext(ctx).Where("created_at < ?", cutoffDate).Delete(&models.Job{})
	if result.Error != nil {
		return 0, fmt.Errorf("cleaning up jobs older than %d days: %w", retentionDays, result.Error)
	}

	return result.RowsAffected, nil
}

// UpdateMetadata updates a job's metadata
func (s *JobService) UpdateMetadata(ctx context.Context, id uint, metadata string) error {
	if err := s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Update("metadata", metadata).Error; err != nil {
		return fmt.Errorf("updating metadata for job %d: %w", id, err)
	}
	return nil
}

// CancelStaleJobs cancels any jobs that are stuck in pending or in_progress status
// This should be called on application startup to clean up jobs from previous runs
func (s *JobService) CancelStaleJobs(ctx context.Context) (int64, error) {
	now := time.Now()

	result := s.db.WithContext(ctx).Model(&models.Job{}).
		Where("status IN ?", []models.JobStatus{models.JobStatusPending, models.JobStatusInProgress}).
		Updates(map[string]interface{}{
			"status":       models.JobStatusCancelled,
			"completed_at": now,
			"error":        "Job cancelled on startup (stale from previous run)",
		})

	if result.Error != nil {
		return 0, fmt.Errorf("cancelling stale jobs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetLastJobByType retrieves the most recent job of a specific type
func (s *JobService) GetLastJobByType(ctx context.Context, jobType models.JobType) (*models.Job, error) {
	var job models.Job

	err := s.db.WithContext(ctx).Where("type = ?", jobType).
		Order("created_at DESC").
		First(&job).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No job found, return nil without error
		}
		return nil, fmt.Errorf("querying last job by type %s: %w", jobType, err)
	}

	return &job, nil
}
