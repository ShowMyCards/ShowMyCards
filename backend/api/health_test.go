package api

import (
	"backend/models"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHealthTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestHealth_Success(t *testing.T) {
	db := setupHealthTest(t)
	handler := NewHealthHandler(db, "test")

	app := fiber.New()
	app.Get("/health", handler.Check)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := result["status"].(string)
	if !ok {
		t.Fatal("expected status field in response")
	}

	if status != "OK" {
		t.Errorf("expected status 'OK', got '%s'", status)
	}

	// Verify database check is included
	checks, ok := result["checks"].(map[string]interface{})
	if !ok {
		t.Fatal("expected checks field in response")
	}

	dbStatus, ok := checks["database"].(string)
	if !ok {
		t.Fatal("expected database check in response")
	}

	if dbStatus != "connected" {
		t.Errorf("expected database status 'connected', got '%s'", dbStatus)
	}
}
