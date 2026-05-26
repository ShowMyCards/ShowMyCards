package api

import (
	"backend/models"
	"backend/services"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSettingsTestApp(t *testing.T) (*fiber.App, *services.SettingsService) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	settingsService := services.NewSettingsService(db)
	handler := NewSettingsHandler(settingsService)

	app := fiber.New()
	app.Get("/api/settings", handler.GetAll)
	app.Get("/api/settings/:key", handler.Get)
	app.Put("/api/settings/:key", handler.Update)
	app.Put("/api/settings", handler.UpdateBulk)

	return app, settingsService
}

// GetAll tests

func TestSettingsGet_Success(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	req := httptest.NewRequest("GET", "/api/settings", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var settings map[string]string
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify default settings exist
	expectedDefaults := []string{
		"bulk_data_auto_update",
		"bulk_data_update_time",
		"bulk_data_url",
		"bulk_data_last_update",
		"bulk_data_last_update_status",
		"scryfall_default_search",
		"scryfall_unique_mode",
	}

	for _, key := range expectedDefaults {
		if _, exists := settings[key]; !exists {
			t.Errorf("expected default setting '%s' to exist", key)
		}
	}

	// Verify specific default values
	if settings["bulk_data_auto_update"] != "true" {
		t.Errorf("expected bulk_data_auto_update='true', got '%s'", settings["bulk_data_auto_update"])
	}

	if settings["bulk_data_update_time"] != "03:00" {
		t.Errorf("expected bulk_data_update_time='03:00', got '%s'", settings["bulk_data_update_time"])
	}
}

// Get single setting tests

func TestSettingsGetSingle_Success(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	req := httptest.NewRequest("GET", "/api/settings/bulk_data_auto_update", nil)
	req.SetPathValue("key", "bulk_data_auto_update")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["key"] != "bulk_data_auto_update" {
		t.Errorf("expected key 'bulk_data_auto_update', got '%s'", result["key"])
	}

	if result["value"] != "true" {
		t.Errorf("expected value 'true', got '%s'", result["value"])
	}
}

func TestSettingsGetSingle_NotFound(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	req := httptest.NewRequest("GET", "/api/settings/nonexistent_key", nil)
	req.SetPathValue("key", "nonexistent_key")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status %d, got %d", fiber.StatusNotFound, resp.StatusCode)
	}
}

// Update single setting tests

func TestSettingsUpdate_Success(t *testing.T) {
	app, service := setupSettingsTestApp(t)

	updateReq := map[string]string{
		"value": "false",
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings/bulk_data_auto_update", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("key", "bulk_data_auto_update")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(body))
	}

	// Verify update persisted
	value, err := service.Get(context.Background(),"bulk_data_auto_update")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}

	if value != "false" {
		t.Errorf("expected value 'false', got '%s'", value)
	}
}

func TestSettingsUpdate_InvalidJSON(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	req := httptest.NewRequest("PUT", "/api/settings/bulk_data_auto_update", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("key", "bulk_data_auto_update")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

func TestSettingsUpdate_InvalidKey(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	updateReq := map[string]string{
		"value": "custom_value",
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings/invalid_key", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("key", "invalid_key")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

// UpdateBulk tests

func TestSettingsUpdateBulk_Success(t *testing.T) {
	app, service := setupSettingsTestApp(t)

	updateReq := map[string]string{
		"bulk_data_auto_update": "false",
		"bulk_data_update_time": "04:00",
		"scryfall_unique_mode":  "prints",
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(body))
	}

	// Verify all updates persisted
	value1, _ := service.Get(context.Background(),"bulk_data_auto_update")
	if value1 != "false" {
		t.Errorf("expected bulk_data_auto_update='false', got '%s'", value1)
	}

	value2, _ := service.Get(context.Background(),"bulk_data_update_time")
	if value2 != "04:00" {
		t.Errorf("expected bulk_data_update_time='04:00', got '%s'", value2)
	}

	value3, _ := service.Get(context.Background(),"scryfall_unique_mode")
	if value3 != "prints" {
		t.Errorf("expected scryfall_unique_mode='prints', got '%s'", value3)
	}
}

func TestSettingsUpdateBulk_UnknownKeyIsDropped(t *testing.T) {
	app, service := setupSettingsTestApp(t)

	updateReq := map[string]string{
		"bulk_data_auto_update": "false",
		"invalid_key":           "some_value",
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	// Unknown keys are silently dropped; the request still succeeds
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// Valid key was updated
	value, _ := service.Get(context.Background(), "bulk_data_auto_update")
	if value != "false" {
		t.Errorf("expected bulk_data_auto_update='false', got '%s'", value)
	}

	// Unknown key was not stored
	unknown, err := service.Get(context.Background(), "invalid_key")
	if err == nil && unknown != "" {
		t.Errorf("expected invalid_key to not be stored, got '%s'", unknown)
	}
}

func TestSettingsUpdateBulk_AppVersionIsDropped(t *testing.T) {
	app, service := setupSettingsTestApp(t)

	// Seed app_version as the version system would
	if err := service.Set(context.Background(), "app_version", "0.1.0"); err != nil {
		t.Fatalf("failed to seed app_version: %v", err)
	}

	// Simulate the frontend round-tripping all settings including app_version
	updateReq := map[string]string{
		"bulk_data_auto_update": "false",
		"app_version":           "0.0.0", // should be ignored
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(body))
	}

	// app_version must not have been overwritten
	version, err := service.Get(context.Background(), "app_version")
	if err != nil {
		t.Fatalf("failed to get app_version: %v", err)
	}
	if version != "0.1.0" {
		t.Errorf("expected app_version='0.1.0' (unchanged), got '%s'", version)
	}
}

func TestSettingsUpdateBulk_InvalidJSON(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

func TestSettingsUpdateBulk_EmptyBody(t *testing.T) {
	app, _ := setupSettingsTestApp(t)

	updateReq := map[string]string{}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d (no error for empty update), got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestSettingsUpdateBulk_PartialUpdate(t *testing.T) {
	app, service := setupSettingsTestApp(t)

	// Update only some settings
	updateReq := map[string]string{
		"bulk_data_auto_update": "false",
		"scryfall_unique_mode":  "prints",
	}
	reqBody, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// Verify updated settings
	value1, _ := service.Get(context.Background(),"bulk_data_auto_update")
	if value1 != "false" {
		t.Errorf("expected bulk_data_auto_update='false', got '%s'", value1)
	}

	value2, _ := service.Get(context.Background(),"scryfall_unique_mode")
	if value2 != "prints" {
		t.Errorf("expected scryfall_unique_mode='prints', got '%s'", value2)
	}

	// Verify untouched settings remain unchanged
	value3, _ := service.Get(context.Background(),"bulk_data_update_time")
	if value3 != "03:00" {
		t.Errorf("expected bulk_data_update_time='03:00' (unchanged), got '%s'", value3)
	}
}
