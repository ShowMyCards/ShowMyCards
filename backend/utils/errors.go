// Package utils provides utility functions for error handling, validation, and pagination.
// It contains helpers used across the API layer for consistent behavior.
package utils

import (
	"log/slog"

	scryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
)

// ErrorResponse standard error format
type ErrorResponse struct {
	Error string `json:"error"`
}

// LogAndReturnError logs server-side and returns user-friendly error
func LogAndReturnError(c fiber.Ctx, statusCode int, userMsg, logMsg string, err error) error {
	if err != nil {
		slog.Error("request failed", "method", c.Method(), "path", c.Path(), "message", logMsg, "error", err)
	} else {
		slog.Error("request failed", "method", c.Method(), "path", c.Path(), "message", logMsg)
	}
	return c.Status(statusCode).JSON(ErrorResponse{Error: userMsg})
}

// ReturnError returns error without logging (for validation errors)
func ReturnError(c fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(ErrorResponse{Error: message})
}

// EnhancedErrorResponse includes additional error context
// tygo:export
type EnhancedErrorResponse struct {
	Error    string   `json:"error"`
	Code     string   `json:"code,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// HandleScryfallError extracts rich error details from Scryfall API errors
func HandleScryfallError(c fiber.Ctx, err error, userMsg string) error {
	// Type assert to *scryfall.Error
	if scryfallErr, ok := err.(*scryfall.Error); ok {
		// Map Scryfall status to appropriate HTTP status
		httpStatus := mapScryfallStatus(scryfallErr.Status)

		slog.Error("scryfall API error", "status", scryfallErr.Status, "code", scryfallErr.Code, "details", scryfallErr.Details)

		return c.Status(httpStatus).JSON(EnhancedErrorResponse{
			Error:    userMsg,
			Code:     scryfallErr.Code,
			Warnings: scryfallErr.Warnings,
		})
	}

	// Fallback for non-Scryfall errors
	slog.Error("non-scryfall error", "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error: userMsg,
	})
}

// mapScryfallStatus maps Scryfall HTTP status to our response status
func mapScryfallStatus(scryfallStatus int) int {
	switch {
	case scryfallStatus == 429:
		return fiber.StatusTooManyRequests
	case scryfallStatus >= 400 && scryfallStatus < 500:
		return scryfallStatus // Pass through client errors
	case scryfallStatus >= 500 && scryfallStatus < 600:
		return fiber.StatusServiceUnavailable
	default:
		return fiber.StatusInternalServerError
	}
}
