package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/models"
	"backend/utils"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Inventory{}, &models.SortingRule{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewStorageHandler(db)

	app.Get("/storage", handler.List)
	app.Get("/storage/:id", handler.Get)
	app.Post("/storage", handler.Create)
	app.Put("/storage/:id", handler.Update)
	app.Delete("/storage/:id", handler.Delete)

	return app, db
}

func createTestLocation(t *testing.T, db *gorm.DB, storageType models.StorageType) models.StorageLocation {
	t.Helper()
	location := models.StorageLocation{
		Name:        "Test " + string(storageType),
		StorageType: storageType,
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to create test location: %v", err)
	}
	return location
}

// List endpoint tests

func TestList_Empty(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/storage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalItems != 0 {
		t.Errorf("expected 0 total items, got %d", result.TotalItems)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
}

func TestList_WithItems(t *testing.T) {
	app, db := setupTestApp(t)

	createTestLocation(t, db, models.Box)
	createTestLocation(t, db, models.Binder)

	req := httptest.NewRequest(http.MethodGet, "/storage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalItems != 2 {
		t.Errorf("expected 2 total items, got %d", result.TotalItems)
	}
}

func TestList_Pagination(t *testing.T) {
	app, db := setupTestApp(t)

	// Create 5 items
	for i := 0; i < 5; i++ {
		createTestLocation(t, db, models.Box)
	}

	req := httptest.NewRequest(http.MethodGet, "/storage?page=2&page_size=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Page)
	}
	if result.PageSize != 2 {
		t.Errorf("expected page_size 2, got %d", result.PageSize)
	}
	if result.TotalItems != 5 {
		t.Errorf("expected 5 total items, got %d", result.TotalItems)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", result.TotalPages)
	}
}

func TestList_PageSizeLimit(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/storage?page_size=200", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.PageSize != utils.MaxPageSize {
		t.Errorf("expected page_size capped at %d, got %d", utils.MaxPageSize, result.PageSize)
	}
}

// Get endpoint tests

func TestGet_Success(t *testing.T) {
	app, db := setupTestApp(t)

	location := createTestLocation(t, db, models.Box)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/storage/%d", location.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.StorageLocation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID != location.ID {
		t.Errorf("expected ID %d, got %d", location.ID, result.ID)
	}
	if result.StorageType != models.Box {
		t.Errorf("expected storage type %s, got %s", models.Box, result.StorageType)
	}
}

func TestGet_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/storage/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestGet_InvalidID(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/storage/invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// Create endpoint tests

func TestCreate_Success(t *testing.T) {
	app, _ := setupTestApp(t)

	body := `{"name": "Main Storage Box", "storage_type": "Box"}`
	req := httptest.NewRequest(http.MethodPost, "/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.StorageLocation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if result.Name != "Main Storage Box" {
		t.Errorf("expected name 'Main Storage Box', got '%s'", result.Name)
	}
	if result.StorageType != models.Box {
		t.Errorf("expected storage type %s, got %s", models.Box, result.StorageType)
	}
}

func TestCreate_Binder(t *testing.T) {
	app, _ := setupTestApp(t)

	body := `{"name": "Foil Binder", "storage_type": "Binder"}`
	req := httptest.NewRequest(http.MethodPost, "/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.StorageLocation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageType != models.Binder {
		t.Errorf("expected storage type %s, got %s", models.Binder, result.StorageType)
	}
}

func TestCreate_InvalidType(t *testing.T) {
	app, _ := setupTestApp(t)

	body := `{"name": "Invalid Storage", "storage_type": "Drawer"}`
	req := httptest.NewRequest(http.MethodPost, "/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	app, _ := setupTestApp(t)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// Update endpoint tests

func TestUpdate_Success(t *testing.T) {
	app, db := setupTestApp(t)

	location := createTestLocation(t, db, models.Box)

	body := `{"storage_type": "Binder"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/storage/%d", location.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.StorageLocation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageType != models.Binder {
		t.Errorf("expected storage type %s, got %s", models.Binder, result.StorageType)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	body := `{"storage_type": "Binder"}`
	req := httptest.NewRequest(http.MethodPut, "/storage/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestUpdate_InvalidType(t *testing.T) {
	app, db := setupTestApp(t)

	location := createTestLocation(t, db, models.Box)

	body := `{"storage_type": "Drawer"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/storage/%d", location.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// Delete endpoint tests

func TestDelete_Success(t *testing.T) {
	app, db := setupTestApp(t)

	location := createTestLocation(t, db, models.Box)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/storage/%d", location.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify deletion
	var count int64
	db.Model(&models.StorageLocation{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 locations after delete, got %d", count)
	}
}

func TestDelete_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/storage/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestDelete_InvalidID(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/storage/invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestDelete_WithInventoryReferences(t *testing.T) {
	app, db := setupTestApp(t)

	// Create a storage location
	location := createTestLocation(t, db, models.Box)

	// Create an inventory item referencing this location
	inventory := models.Inventory{
		ScryfallID:        "test-id-1",
		OracleID:          "oracle-1",
		Treatment:         "normal",
		Quantity:          1,
		StorageLocationID: &location.ID,
	}
	if err := db.Create(&inventory).Error; err != nil {
		t.Fatalf("failed to create test inventory: %v", err)
	}

	// Attempt to delete the storage location
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/storage/%d", location.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 409 Conflict
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, resp.StatusCode)
	}

	// Verify location still exists
	var count int64
	db.Model(&models.StorageLocation{}).Where("id = ?", location.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected location to still exist, got count %d", count)
	}
}

func TestDelete_WithSortingRuleReferences(t *testing.T) {
	app, db := setupTestApp(t)

	// Create a storage location
	location := createTestLocation(t, db, models.Box)

	// Create a sorting rule referencing this location
	rule := models.SortingRule{
		Name:              "Test Rule",
		Priority:          1,
		Expression:        "true",
		StorageLocationID: location.ID,
		Enabled:           true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create test rule: %v", err)
	}

	// Attempt to delete the storage location
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/storage/%d", location.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 409 Conflict
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, resp.StatusCode)
	}

	// Verify location still exists
	var count int64
	db.Model(&models.StorageLocation{}).Where("id = ?", location.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected location to still exist, got count %d", count)
	}
}

func TestDelete_WithBothReferences(t *testing.T) {
	app, db := setupTestApp(t)

	// Create a storage location
	location := createTestLocation(t, db, models.Box)

	// Create an inventory item referencing this location
	inventory := models.Inventory{
		ScryfallID:        "test-id-1",
		OracleID:          "oracle-1",
		Treatment:         "normal",
		Quantity:          1,
		StorageLocationID: &location.ID,
	}
	if err := db.Create(&inventory).Error; err != nil {
		t.Fatalf("failed to create test inventory: %v", err)
	}

	// Create a sorting rule referencing this location
	rule := models.SortingRule{
		Name:              "Test Rule",
		Priority:          1,
		Expression:        "true",
		StorageLocationID: location.ID,
		Enabled:           true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create test rule: %v", err)
	}

	// Attempt to delete the storage location
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/storage/%d", location.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 409 Conflict
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, resp.StatusCode)
	}

	// Parse response to verify both counts are reported
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if inventoryCount, ok := result["inventory_count"].(float64); !ok || inventoryCount != 1 {
		t.Errorf("expected inventory_count 1, got %v", result["inventory_count"])
	}

	if ruleCount, ok := result["sorting_rule_count"].(float64); !ok || ruleCount != 1 {
		t.Errorf("expected sorting_rule_count 1, got %v", result["sorting_rule_count"])
	}

	// Verify location still exists
	var count int64
	db.Model(&models.StorageLocation{}).Where("id = ?", location.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected location to still exist, got count %d", count)
	}
}
