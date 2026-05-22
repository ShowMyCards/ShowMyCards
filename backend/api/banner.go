package api

import (
	"backend/services"
	"backend/utils"

	"github.com/gofiber/fiber/v3"
)

// BannerHandler handles banner-related HTTP requests.
type BannerHandler struct {
	service *services.BannerService
}

// NewBannerHandler creates a new banner handler.
func NewBannerHandler(service *services.BannerService) *BannerHandler {
	return &BannerHandler{service: service}
}

// GetAll returns the banners that should currently be displayed.
func (h *BannerHandler) GetAll(c fiber.Ctx) error {
	banners, err := h.service.GetActive(c.RequestCtx())
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to retrieve banners", "banner query failed", err)
	}

	return c.JSON(banners)
}
