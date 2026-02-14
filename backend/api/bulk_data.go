package api

import (
	"backend/services"
	"backend/utils"
	"context"

	"github.com/gofiber/fiber/v3"
)

// BulkDataHandler handles bulk data-related HTTP requests
type BulkDataHandler struct {
	service *services.BulkDataService
}

// NewBulkDataHandler creates a new bulk data handler
func NewBulkDataHandler(service *services.BulkDataService) *BulkDataHandler {
	return &BulkDataHandler{service: service}
}

// TriggerImport triggers a bulk data download and import
func (h *BulkDataHandler) TriggerImport(c fiber.Ctx, appCtx context.Context) error {
	// Create a job for this import
	job, err := h.service.CreateImportJob(appCtx)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create import job", "job creation failed", err)
	}

	// Start the import in a goroutine (async)
	go func() {
		if err := h.service.DownloadAndImport(appCtx, job.ID); err != nil {
			// Error is already logged and job is marked as failed in the service
			return
		}
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "Import job started",
		"job_id":  job.ID,
	})
}
