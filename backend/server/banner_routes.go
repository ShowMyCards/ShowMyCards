package server

import (
	"backend/api"
	"backend/services"

	"github.com/gofiber/fiber/v3"
)

// BannerRoutes registers banner-related routes.
func BannerRoutes(app *fiber.App, jobService *services.JobService) {
	service := services.NewBannerService(jobService)
	handler := api.NewBannerHandler(service)

	app.Get("/api/banners", handler.GetAll)
}
