package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DeckRoutes registers deck routes.
func DeckRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewDeckHandler(db)

	decks := app.Group("/api/decks")
	decks.Get("/", handler.List)
	decks.Get("/:id", handler.Get)
	decks.Post("/", handler.Create)
	decks.Put("/:id", handler.Update)
	decks.Delete("/:id", handler.Delete)

	// Deck item routes
	decks.Get("/:id/items", handler.ListItems)
	decks.Post("/:id/items/batch", handler.CreateItemsBatch)
	decks.Put("/:id/items/:item_id", handler.UpdateItem)
	decks.Delete("/:id/items/:item_id", handler.DeleteItem)
}
