package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
)

func TestHandleScryfallError_NotFound(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		scryfallErr := &scryfall.Error{
			Status:  404,
			Code:    "not_found",
			Details: "Card not found",
		}
		return HandleScryfallError(c, scryfallErr, "failed to find card")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	var result EnhancedErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "failed to find card" {
		t.Errorf("expected error 'failed to find card', got '%s'", result.Error)
	}
	if result.Code != "not_found" {
		t.Errorf("expected code 'not_found', got '%s'", result.Code)
	}
}

func TestHandleScryfallError_BadRequest(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		scryfallErr := &scryfall.Error{
			Status:  400,
			Code:    "bad_request",
			Details: "Invalid query syntax",
		}
		return HandleScryfallError(c, scryfallErr, "failed to search cards")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var result EnhancedErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "failed to search cards" {
		t.Errorf("expected error 'failed to search cards', got '%s'", result.Error)
	}
	if result.Code != "bad_request" {
		t.Errorf("expected code 'bad_request', got '%s'", result.Code)
	}
}

func TestHandleScryfallError_RateLimit(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		scryfallErr := &scryfall.Error{
			Status:  429,
			Code:    "rate_limit",
			Details: "Too many requests",
		}
		return HandleScryfallError(c, scryfallErr, "rate limited")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", resp.StatusCode)
	}

	var result EnhancedErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "rate limited" {
		t.Errorf("expected error 'rate limited', got '%s'", result.Error)
	}
	if result.Code != "rate_limit" {
		t.Errorf("expected code 'rate_limit', got '%s'", result.Code)
	}
}

func TestHandleScryfallError_ServerError(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		scryfallErr := &scryfall.Error{
			Status:  503,
			Code:    "service_unavailable",
			Details: "Scryfall is down",
		}
		return HandleScryfallError(c, scryfallErr, "service error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var result EnhancedErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "service error" {
		t.Errorf("expected error 'service error', got '%s'", result.Error)
	}
	if result.Code != "service_unavailable" {
		t.Errorf("expected code 'service_unavailable', got '%s'", result.Code)
	}
}

func TestHandleScryfallError_WithWarnings(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		scryfallErr := &scryfall.Error{
			Status:   400,
			Code:     "bad_request",
			Details:  "Query has issues",
			Warnings: []string{"Consider using 'is:spell' instead", "Use quotes for exact names"},
		}
		return HandleScryfallError(c, scryfallErr, "query error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	var result EnhancedErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "query error" {
		t.Errorf("expected error 'query error', got '%s'", result.Error)
	}
	if result.Code != "bad_request" {
		t.Errorf("expected code 'bad_request', got '%s'", result.Code)
	}
	if len(result.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result.Warnings))
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if w == "Consider using 'is:spell' instead" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected warning 'Consider using 'is:spell' instead' not found")
	}
}

func TestHandleScryfallError_GenericError(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		// Non-Scryfall error
		genericErr := errors.New("something went wrong")
		return HandleScryfallError(c, genericErr, "generic error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var result ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "generic error" {
		t.Errorf("expected error 'generic error', got '%s'", result.Error)
	}
}

func TestMapScryfallStatus(t *testing.T) {
	tests := []struct {
		name           string
		scryfallStatus int
		expectedStatus int
	}{
		{"400 Bad Request", 400, 400},
		{"404 Not Found", 404, 404},
		{"429 Rate Limit", 429, 429},
		{"500 Server Error", 500, 503},
		{"503 Service Unavailable", 503, 503},
		{"Unknown Status", 999, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapScryfallStatus(tt.scryfallStatus)
			if result != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, result)
			}
		})
	}
}

