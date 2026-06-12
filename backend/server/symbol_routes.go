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
	symbols.Get("/:symbol", handler.GetSVG)
	symbols.Post("/import", func(c fiber.Ctx) error {
		return handler.TriggerImport(c, appCtx)
	})
}
