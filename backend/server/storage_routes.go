package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// StorageRoutes registers storage location routes
func StorageRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewStorageHandler(db)

	storage := app.Group("/storage")
	storage.Get("/", handler.List)
	storage.Get("/with-counts", handler.ListWithCounts)
	storage.Get("/:id", handler.Get)
	storage.Post("/", handler.Create)
	storage.Put("/:id", handler.Update)
	storage.Delete("/:id", handler.Delete)
}
