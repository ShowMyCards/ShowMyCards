package server

import (
	"backend/api"
	"backend/services"
	"context"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SetRoutes registers set routes
func SetRoutes(app *fiber.App, db *gorm.DB, setDataService *services.SetDataService, dataDir string, appCtx context.Context) {
	handler := api.NewSetHandler(db, setDataService, dataDir)

	sets := app.Group("/sets")
	sets.Get("/", handler.List)
	sets.Get("/id/:id", handler.GetByID)
	sets.Get("/code/:code", handler.GetByCode)
	sets.Get("/code/:code/icon", handler.GetIcon)
	sets.Post("/import", func(c fiber.Ctx) error {
		return handler.TriggerImport(c, appCtx)
	})
}
