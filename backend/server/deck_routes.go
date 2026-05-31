package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DeckRoutes registers deck routes.
//
// Items endpoints beyond the stub GET land in FR98 Milestone 1b alongside the
// allocation service.
func DeckRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewDeckHandler(db)

	decks := app.Group("/api/decks")
	decks.Get("/", handler.List)
	decks.Get("/:id", handler.Get)
	decks.Post("/", handler.Create)
	decks.Put("/:id", handler.Update)
	decks.Delete("/:id", handler.Delete)

	// Stub — see DeckHandler.ListItems. Real implementation lands in 1b.
	decks.Get("/:id/items", handler.ListItems)
}
