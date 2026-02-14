package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ListRoutes registers list routes
func ListRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewListHandler(db)

	lists := app.Group("/lists")
	lists.Get("/", handler.List)
	lists.Get("/:id", handler.Get)
	lists.Post("/", handler.Create)
	lists.Put("/:id", handler.Update)
	lists.Delete("/:id", handler.Delete)

	// List item routes
	lists.Get("/:id/items", handler.ListItems)
	lists.Post("/:id/items/batch", handler.CreateItemsBatch)
	lists.Put("/:id/items/:item_id", handler.UpdateItem)
	lists.Delete("/:id/items/:item_id", handler.DeleteItem)
}
