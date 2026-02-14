package api

import (
	"backend/models"
	"backend/services"
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ScheduledTaskInfo represents information about a scheduled task
// tygo:export
type ScheduledTaskInfo struct {
	Name          string     `json:"name"`
	Type          string     `json:"type"` // "bulk_data_update" | "job_cleanup"
	Enabled       bool       `json:"enabled"`
	Schedule      string     `json:"schedule"` // e.g., "03:00 daily"
	NextRun       time.Time  `json:"next_run"`
	LastRun       *time.Time `json:"last_run,omitempty"`
	LastJobID     *uint      `json:"last_job_id,omitempty"`
	LastJobStatus *string    `json:"last_job_status,omitempty"`
}

// SchedulerHandler handles scheduler-related API requests
type SchedulerHandler struct {
	settingsService *services.SettingsService
	jobService      *services.JobService
}

// NewSchedulerHandler creates a new scheduler handler
func NewSchedulerHandler(settingsService *services.SettingsService, jobService *services.JobService) *SchedulerHandler {
	return &SchedulerHandler{
		settingsService: settingsService,
		jobService:      jobService,
	}
}

// GetScheduledTasks returns information about all scheduled tasks
func (h *SchedulerHandler) GetScheduledTasks(c fiber.Ctx) error {
	tasks := []ScheduledTaskInfo{}

	// Bulk Data Update Task
	bulkDataTask := h.getBulkDataTaskInfo(c.RequestCtx())
	tasks = append(tasks, bulkDataTask)

	// Job Cleanup Task
	cleanupTask := h.getJobCleanupTaskInfo()
	tasks = append(tasks, cleanupTask)

	return c.JSON(tasks)
}

// getBulkDataTaskInfo returns info about the bulk data update task
func (h *SchedulerHandler) getBulkDataTaskInfo(ctx context.Context) ScheduledTaskInfo {
	// Check if auto-update is enabled
	enabled := h.settingsService.GetBool(ctx, "bulk_data_auto_update", false)

	// Get configured update time
	updateTime, err := h.settingsService.Get(ctx, "bulk_data_update_time")
	if err != nil || updateTime == "" {
		updateTime = "03:00" // Default to 3 AM
	}

	// Calculate next run time
	nextRun := calculateNextRun(updateTime)

	// Get last job
	lastJob, err := h.jobService.GetLastJobByType(ctx, models.JobTypeBulkDataImport)
	if err != nil {
		slog.Warn("failed to get last bulk data job", "component", "scheduler", "error", err)
	}

	task := ScheduledTaskInfo{
		Name:     "Bulk Data Auto-Update",
		Type:     "bulk_data_update",
		Enabled:  enabled,
		Schedule: updateTime + " daily",
		NextRun:  nextRun,
	}

	// Add last job info if it exists
	if lastJob != nil {
		task.LastJobID = &lastJob.ID

		if lastJob.CompletedAt != nil {
			task.LastRun = lastJob.CompletedAt
		} else if lastJob.StartedAt != nil {
			task.LastRun = lastJob.StartedAt
		}

		statusStr := string(lastJob.Status)
		task.LastJobStatus = &statusStr
	}

	return task
}

// getJobCleanupTaskInfo returns info about the job cleanup task
func (h *SchedulerHandler) getJobCleanupTaskInfo() ScheduledTaskInfo {
	// Job cleanup is always enabled and runs at midnight
	cleanupTime := "00:00"
	nextRun := calculateNextRun(cleanupTime)

	task := ScheduledTaskInfo{
		Name:     "Job History Cleanup",
		Type:     "job_cleanup",
		Enabled:  true,
		Schedule: cleanupTime + " daily",
		NextRun:  nextRun,
	}

	// Note: Job cleanup doesn't create a Job record, so no last job info

	return task
}

// calculateNextRun calculates the next run time for a given schedule (HH:MM format)
func calculateNextRun(schedule string) time.Time {
	// Parse schedule (format: "HH:MM")
	targetTime, err := time.Parse("15:04", schedule)
	if err != nil {
		// Fallback to midnight if parsing fails
		targetTime, _ = time.Parse("15:04", "00:00")
	}

	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(),
		targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())

	// If time already passed today, schedule for tomorrow
	if next.Before(now) || next.Equal(now) {
		next = next.AddDate(0, 0, 1)
	}

	return next
}
