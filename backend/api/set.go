package api

import (
	"backend/models"
	"backend/utils"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SetHandler handles set endpoints
type SetHandler struct {
	db      *gorm.DB
	dataDir string
}

// NewSetHandler creates a new set handler
func NewSetHandler(db *gorm.DB, dataDir string) *SetHandler {
	return &SetHandler{
		db:      db,
		dataDir: dataDir,
	}
}

// List returns sets with pagination
func (h *SetHandler) List(c fiber.Ctx) error {
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, utils.MaxPageSize)

	var total int64
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.Set{}).Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count sets", "database count failed", err)
	}

	var sets []models.Set
	offset := utils.CalculateOffset(params.Page, params.PageSize)
	if err := h.db.WithContext(c.RequestCtx()).Order("released_at DESC, name ASC").Offset(offset).Limit(params.PageSize).Find(&sets).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch sets", "database query failed", err)
	}

	response := utils.NewPaginatedResponse(sets, params.Page, params.PageSize, total)
	return c.JSON(response)
}

// GetByID returns a single set by Scryfall ID
func (h *SetHandler) GetByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var set models.Set
	if err := h.db.WithContext(c.RequestCtx()).Where("scryfall_id = ?", id).First(&set).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "set not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch set", "database query failed", err)
	}
	return c.JSON(set)
}

// GetByCode returns a single set by set code
func (h *SetHandler) GetByCode(c fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid code")
	}

	var set models.Set
	if err := h.db.WithContext(c.RequestCtx()).Where("code = ?", code).First(&set).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "set not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch set", "database query failed", err)
	}
	return c.JSON(set)
}

// GetIcon returns the SVG icon for a set
func (h *SetHandler) GetIcon(c fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid code")
	}

	iconPath := filepath.Join(h.dataDir, "set-icons", code+".svg")

	// Ensure the resolved path stays within the expected directory
	absIcon, err := filepath.Abs(iconPath)
	if err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid code")
	}
	absBase, err := filepath.Abs(filepath.Join(h.dataDir, "set-icons"))
	if err != nil {
		return utils.ReturnError(c, fiber.StatusInternalServerError, "internal error")
	}
	if !strings.HasPrefix(absIcon, absBase+string(filepath.Separator)) {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid code")
	}

	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		return utils.ReturnError(c, fiber.StatusNotFound, "icon not found")
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.SendFile(iconPath)
}
