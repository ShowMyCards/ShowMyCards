package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SetRoutes registers set routes
func SetRoutes(app *fiber.App, db *gorm.DB, dataDir string) {
	handler := api.NewSetHandler(db, dataDir)

	sets := app.Group("/api/sets")
	sets.Get("/", handler.List)
	sets.Get("/id/:id", handler.GetByID)
	sets.Get("/code/:code", handler.GetByCode)
	sets.Get("/code/:code/icon", handler.GetIcon)
}
