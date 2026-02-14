package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HealthRoutes registers health check routes
func HealthRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewHealthHandler(db)
	app.Get("/health", handler.Check)
}
