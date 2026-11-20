package server

import (
	"backend/api"
	"backend/services"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DeckRoutes registers deck routes.
func DeckRoutes(app *fiber.App, db *gorm.DB, settingsService *services.SettingsService) {
	handler := api.NewDeckHandler(db, settingsService)

	decks := app.Group("/api/decks")
	decks.Get("/", handler.List)
	decks.Get("/for-card/:scryfall_id", handler.DecksForCard)
	decks.Get("/:id", handler.Get)
	decks.Get("/:id/legality", handler.Legality)
	decks.Post("/", handler.Create)
	decks.Put("/:id", handler.Update)
	decks.Delete("/:id", handler.Delete)

	// Deck item routes
	decks.Get("/:id/items", handler.ListItems)
	decks.Post("/:id/items/batch", handler.CreateItemsBatch)
	decks.Put("/:id/items/:item_id", handler.UpdateItem)
	decks.Post("/:id/items/:item_id/move", handler.MoveItem)
	decks.Delete("/:id/items/:item_id", handler.DeleteItem)
}
