package api

import (
	"backend/models"
	"backend/utils"
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SymbolHandler handles symbol endpoints
type SymbolHandler struct {
	db *gorm.DB
}

// NewSymbolHandler creates a new symbol handler
func NewSymbolHandler(db *gorm.DB) *SymbolHandler {
	return &SymbolHandler{db: db}
}

// GetSVG returns the cached SVG for a given symbol.
//
// The path parameter is URL-decoded and then normalized (braces stripped,
// uppercased) so that every Scryfall symbol resolves regardless of how the
// client spells it: bare ("T"), braced ("{T}"), lowercase ("t"),
// percent-encoded ("%7BT%7D"), hybrid/Phyrexian symbols containing slashes
// ("{W/U}", "{B/G/P}"), and Unicode symbols ("½", "∞"). A greedy "+" route
// captures the slashes; Fiber returns the parameter without decoding, so we
// percent-decode it ourselves.
func (h *SymbolHandler) GetSVG(c fiber.Ctx) error {
	raw := c.Params("+")
	if raw == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid symbol")
	}

	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
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
