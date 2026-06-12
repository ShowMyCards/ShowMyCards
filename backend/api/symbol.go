package api

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SymbolHandler handles symbol endpoints
type SymbolHandler struct {
	db                *gorm.DB
	symbolDataService *services.SymbolDataService
}

// NewSymbolHandler creates a new symbol handler
func NewSymbolHandler(db *gorm.DB, symbolDataService *services.SymbolDataService) *SymbolHandler {
	return &SymbolHandler{
		db:                db,
		symbolDataService: symbolDataService,
	}
}

// GetSVG returns the cached SVG for a given symbol.
//
// The path parameter is normalized (braces stripped, uppercased) so that
// "{T}", "t", and "T" all resolve to the same cached symbol.
func (h *SymbolHandler) GetSVG(c fiber.Ctx) error {
	raw := c.Params("symbol")
	if raw == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid symbol")
	}

	code := models.NormalizeSymbolCode(raw)
	if code == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid symbol")
	}

	var symbol models.Symbol
	if err := h.db.WithContext(c.RequestCtx()).Where("code = ?", code).First(&symbol).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "symbol not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch symbol", "database query failed", err)
	}

	if symbol.SVG == "" {
		return utils.ReturnError(c, fiber.StatusNotFound, "symbol svg not found")
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.SendString(symbol.SVG)
}

// TriggerImport triggers a symbol data import
func (h *SymbolHandler) TriggerImport(c fiber.Ctx, appCtx context.Context) error {
	job, err := h.symbolDataService.CreateImportJob(appCtx)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create import job", "job creation failed", err)
	}

	// Run import in background
	go func() {
		if err := h.symbolDataService.DownloadAndImport(appCtx, job.ID); err != nil {
			slog.Error("symbol data import failed", "job_id", job.ID, "error", err)
		}
	}()

	return c.Status(fiber.StatusAccepted).JSON(TriggerImportResponse{
		Message: "Symbol data import started",
		JobID:   job.ID,
	})
}
