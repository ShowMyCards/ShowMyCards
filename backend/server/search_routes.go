package server

import (
	"backend/api"
	"backend/scryfall"
	"backend/services"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SearchRoutes registers card search routes
func SearchRoutes(app *fiber.App, client *scryfall.Client, db *gorm.DB, settingsService *services.SettingsService) {
	handler := api.NewSearchHandler(client, db, settingsService)

	app.Get("/api/search", handler.Search)
	app.Get("/api/search/autocomplete", handler.Autocomplete)
	app.Get("/api/cards/:id", handler.GetCard)
}
