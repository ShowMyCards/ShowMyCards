package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HealthRoutes registers health check routes
func HealthRoutes(app *fiber.App, db *gorm.DB, version string) {
	handler := api.NewHealthHandler(db, version)
	app.Get("/health", handler.Check)
}
