package server

import (
	"backend/api"
	"backend/services"

	"github.com/gofiber/fiber/v3"
)

// JobsRoutes registers job-related routes
func JobsRoutes(app *fiber.App, service *services.JobService) {
	handler := api.NewJobsHandler(service)

	jobs := app.Group("/api/jobs")
	jobs.Get("/", handler.GetAll)
	jobs.Get("/:id", handler.Get)
	jobs.Delete("/cleanup", handler.Cleanup)
}
