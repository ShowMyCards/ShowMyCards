package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DataRoutes registers data import and export routes
func DataRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewDataHandler(db)

	data := app.Group("/api/data")
	data.Get("/export", handler.Export)
	data.Post("/import", handler.Import)
}
