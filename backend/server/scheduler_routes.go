package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
)

// RegisterSchedulerRoutes registers scheduler-related routes
func (s *Server) RegisterSchedulerRoutes(app *fiber.App) {
	schedulerHandler := api.NewSchedulerHandler(s.settingsService, s.jobService)

	scheduler := app.Group("/api/scheduler")
	scheduler.Get("/tasks", schedulerHandler.GetScheduledTasks)
}
