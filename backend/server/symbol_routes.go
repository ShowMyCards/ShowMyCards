package server

import (
	"backend/api"
	"backend/services"
	"context"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SymbolRoutes registers symbol routes
func SymbolRoutes(app *fiber.App, db *gorm.DB, symbolDataService *services.SymbolDataService, appCtx context.Context) {
	handler := api.NewSymbolHandler(db, symbolDataService)

	symbols := app.Group("/api/symbols")
	// Greedy "+" so hybrid/Phyrexian symbols containing slashes (e.g. "{W/U}")
	// are captured intact rather than split across path segments.
	symbols.Get("/+", handler.GetSVG)
	symbols.Post("/import", func(c fiber.Ctx) error {
		return handler.TriggerImport(c, appCtx)
	})
}
