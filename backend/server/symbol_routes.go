package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SymbolRoutes registers symbol routes
func SymbolRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewSymbolHandler(db)

	symbols := app.Group("/api/symbols")
	// Greedy "+" so hybrid/Phyrexian symbols containing slashes (e.g. "{W/U}")
	// are captured intact rather than split across path segments.
	symbols.Get("/+", handler.GetSVG)
}
