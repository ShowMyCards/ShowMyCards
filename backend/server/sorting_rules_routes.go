package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SortingRulesRoutes registers sorting rule routes
func SortingRulesRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewSortingRulesHandler(db)

	rules := app.Group("/sorting-rules")
	rules.Get("/", handler.List)
	rules.Get("/:id", handler.Get)
	rules.Post("/", handler.Create)
	rules.Put("/:id", handler.Update)
	rules.Delete("/:id", handler.Delete)

	// Batch operations
	rules.Post("/batch/priorities", handler.BatchUpdatePriorities)

	// Evaluation endpoints
	rules.Post("/evaluate", handler.Evaluate)
	rules.Post("/validate", handler.ValidateExpression)
}
