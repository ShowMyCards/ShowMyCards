package api

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db      *gorm.DB
	version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// Check returns health status including database connectivity
func (h *HealthHandler) Check(c fiber.Ctx) error {
	dbStatus := "connected"
	httpStatus := fiber.StatusOK

	sqlDB, err := h.db.DB()
	if err != nil {
		dbStatus = "disconnected"
		httpStatus = fiber.StatusServiceUnavailable
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "unreachable"
		httpStatus = fiber.StatusServiceUnavailable
	}

	status := "OK"
	if httpStatus != fiber.StatusOK {
		status = "unhealthy"
	}

	return c.Status(httpStatus).JSON(fiber.Map{
		"status":  status,
		"version": h.version,
		"checks": fiber.Map{
			"database": dbStatus,
		},
	})
}
