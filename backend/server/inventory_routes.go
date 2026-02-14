package server

import (
	"backend/api"
	"backend/services"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// InventoryRoutes registers inventory routes
func InventoryRoutes(app *fiber.App, db *gorm.DB) {
	autoSortSvc := services.NewAutoSortService(db)
	handler := api.NewInventoryHandler(db, autoSortSvc)

	inventory := app.Group("/inventory")
	inventory.Get("/", handler.List)
	inventory.Get("/cards", handler.ListAsCards)
	inventory.Get("/unassigned/count", handler.GetUnassignedCount)
	inventory.Get("/by-oracle/:oracle_id", handler.ByOracle)
	inventory.Post("/batch/move", handler.BatchMove)
	inventory.Delete("/batch", handler.BatchDelete)
	inventory.Post("/resort", handler.Resort)
	inventory.Get("/:id", handler.Get)
	inventory.Post("/", handler.Create)
	inventory.Put("/:id", handler.Update)
	inventory.Delete("/:id", handler.Delete)
}
