package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DashboardRoutes registers dashboard-related routes
func DashboardRoutes(app *fiber.App, db *gorm.DB) {
	handler := api.NewDashboardHandler(db)
	app.Get("/api/dashboard/stats", handler.GetStats)
}
