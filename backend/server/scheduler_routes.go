package server

import (
	"backend/api"

	"github.com/gofiber/fiber/v3"
)

// RegisterSchedulerRoutes registers scheduler-related routes
func (s *Server) RegisterSchedulerRoutes(app *fiber.App) {
	schedulerHandler := api.NewSchedulerHandler(s.appCtx, s.scheduler, s.settingsService, s.jobService)

	scheduler := app.Group("/api/scheduler")
	scheduler.Get("/tasks", schedulerHandler.GetScheduledTasks)
	scheduler.Post("/tasks/:name/run", schedulerHandler.RunTask)

	// Legacy per-type import endpoints, kept as aliases of the generic run.
	app.Post("/api/bulk-data/import", schedulerHandler.RunNamedTask("bulk_data_update"))
	app.Post("/api/sets/import", schedulerHandler.RunNamedTask("set_data_update"))
	app.Post("/api/symbols/import", schedulerHandler.RunNamedTask("symbol_data_update"))
}
