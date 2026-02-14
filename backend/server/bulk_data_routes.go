package server

import (
	"backend/api"
	"backend/services"
	"context"

	"github.com/gofiber/fiber/v3"
)

// BulkDataRoutes registers bulk data-related routes
func BulkDataRoutes(app *fiber.App, service *services.BulkDataService, appCtx context.Context) {
	handler := api.NewBulkDataHandler(service)

	bulkData := app.Group("/api/bulk-data")
	bulkData.Post("/import", func(c fiber.Ctx) error {
		return handler.TriggerImport(c, appCtx)
	})
}
