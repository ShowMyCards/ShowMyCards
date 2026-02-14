package api

import (
	"backend/services"
	"backend/utils"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	service *services.SettingsService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(service *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// GetAll retrieves all settings
func (h *SettingsHandler) GetAll(c fiber.Ctx) error {
	settings, err := h.service.GetAll(c.RequestCtx())
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to retrieve settings", "settings query failed", err)
	}

	return c.JSON(settings)
}

// Get retrieves a single setting by key
func (h *SettingsHandler) Get(c fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Key is required")
	}

	value, err := h.service.Get(c.RequestCtx(), key)
	if err != nil {
		return utils.ReturnError(c, fiber.StatusNotFound, "Setting not found")
	}

	return c.JSON(fiber.Map{
		"key":   key,
		"value": value,
	})
}

// Update updates a single setting
func (h *SettingsHandler) Update(c fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Key is required")
	}

	if !services.ValidSettingKeys()[key] {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("invalid setting key: %s", key))
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.service.Set(c.RequestCtx(), key, req.Value); err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update setting", "setting update failed", err)
	}

	return c.JSON(fiber.Map{
		"key":   key,
		"value": req.Value,
	})
}

// UpdateBulk updates multiple settings at once
func (h *SettingsHandler) UpdateBulk(c fiber.Ctx) error {
	var req map[string]string

	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate keys against whitelist
	validKeys := services.ValidSettingKeys()
	for key := range req {
		if !validKeys[key] {
			return utils.ReturnError(c, fiber.StatusBadRequest,
				fmt.Sprintf("invalid setting key: %s", key))
		}
	}

	if err := h.service.SetBulk(c.RequestCtx(), req); err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update settings", "bulk setting update failed", err)
	}

	return c.JSON(fiber.Map{
		"message": "Settings updated successfully",
	})
}
