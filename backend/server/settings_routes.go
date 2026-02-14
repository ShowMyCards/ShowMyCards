package server

import (
	"backend/api"
	"backend/services"

	"github.com/gofiber/fiber/v3"
)

// SettingsRoutes registers settings-related routes
func SettingsRoutes(app *fiber.App, service *services.SettingsService) {
	handler := api.NewSettingsHandler(service)

	settings := app.Group("/api/settings")
	settings.Get("/", handler.GetAll)
	settings.Put("/", handler.UpdateBulk)
	settings.Get("/:key", handler.Get)
	settings.Put("/:key", handler.Update)
}
