package api

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// JobsHandler handles job-related HTTP requests
type JobsHandler struct {
	service *services.JobService
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(service *services.JobService) *JobsHandler {
	return &JobsHandler{service: service}
}

// GetAll retrieves all jobs with pagination and optional filtering
func (h *JobsHandler) GetAll(c fiber.Ctx) error {
	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, utils.MaxPageSize)

	// Parse filter parameters
	var jobType *models.JobType
	var status *models.JobStatus

	if typeStr := c.Query("type"); typeStr != "" {
		t := models.JobType(typeStr)
		if t.Valid() {
			jobType = &t
		}
	}

	if statusStr := c.Query("status"); statusStr != "" {
		s := models.JobStatus(statusStr)
		if s.Valid() {
			status = &s
		}
	}

	// Get jobs
	jobs, total, err := h.service.List(c.RequestCtx(), params.Page, params.PageSize, jobType, status)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to retrieve jobs", "job list query failed", err)
	}

	response := utils.NewPaginatedResponse(jobs, params.Page, params.PageSize, total)
	return c.JSON(response)
}

// Get retrieves a single job by ID
func (h *JobsHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Invalid job ID")
	}

	job, err := h.service.Get(c.RequestCtx(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "Job not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to retrieve job", "job query failed", err)
	}

	return c.JSON(job)
}

// Cleanup removes old jobs based on retention period
func (h *JobsHandler) Cleanup(c fiber.Ctx) error {
	// Default to 30 days retention
	retentionDays, _ := strconv.Atoi(c.Query("retention_days", strconv.Itoa(DefaultJobRetentionDays)))

	if retentionDays < 1 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Retention days must be at least 1")
	}

	deletedCount, err := h.service.CleanupOldJobs(c.RequestCtx(), retentionDays)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to cleanup jobs", "job cleanup failed", err)
	}

	return c.JSON(fiber.Map{
		"deleted_count":  deletedCount,
		"retention_days": retentionDays,
	})
}
