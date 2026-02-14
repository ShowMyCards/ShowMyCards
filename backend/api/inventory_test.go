package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"backend/models"
	"backend/services"
	"backend/utils"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInventoryTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.StorageLocation{}, &models.Inventory{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewInventoryHandler(db, services.NewAutoSortService(db))

	app.Get("/inventory", handler.List)
	app.Get("/inventory/:id", handler.Get)
	app.Post("/inventory", handler.Create)
	app.Put("/inventory/:id", handler.Update)
	app.Delete("/inventory/:id", handler.Delete)

	return app, db
}

func createTestInventoryItem(t *testing.T, db *gorm.DB, scryfallID string, quantity int, locationID *uint) models.Inventory {
	t.Helper()
	item := models.Inventory{
		ScryfallID:        scryfallID,
		OracleID:          "test-oracle-" + scryfallID,
		Treatment:         "normal",
		Quantity:          quantity,
		StorageLocationID: locationID,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create test inventory item: %v", err)
	}
	return item
}

// List endpoint tests

func TestInventoryList_Empty(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
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
}

func TestInventoryList_WithItems(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	createTestInventoryItem(t, db, "card-1", 1, nil)
	createTestInventoryItem(t, db, "card-2", 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
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

func TestInventoryList_FilterByScryfallID(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	createTestInventoryItem(t, db, "card-1", 1, nil)
	createTestInventoryItem(t, db, "card-2", 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory?scryfall_id=card-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalItems != 1 {
		t.Errorf("expected 1 item with scryfall_id card-1, got %d", result.TotalItems)
	}
}

func TestInventoryList_FilterByStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	locationID := location.ID

	createTestInventoryItem(t, db, "card-1", 1, &locationID)
	createTestInventoryItem(t, db, "card-2", 2, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/inventory?storage_location_id=%d", locationID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalItems != 1 {
		t.Errorf("expected 1 item in storage location, got %d", result.TotalItems)
	}
}

func TestInventoryList_FilterByNullStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	locationID := location.ID

	createTestInventoryItem(t, db, "card-1", 1, &locationID)
	createTestInventoryItem(t, db, "card-2", 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory?storage_location_id=null", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalItems != 1 {
		t.Errorf("expected 1 item without storage location, got %d", result.TotalItems)
	}
}

func TestInventoryList_Pagination(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	for i := 1; i <= 5; i++ {
		createTestInventoryItem(t, db, fmt.Sprintf("card-%d", i), i, nil)
	}

	req := httptest.NewRequest(http.MethodGet, "/inventory?page=2&page_size=2", nil)
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
}

// Get endpoint tests

func TestInventoryGet_Success(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "test-card", 3, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/inventory/%d", item.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID != item.ID {
		t.Errorf("expected ID %d, got %d", item.ID, result.ID)
	}
	if result.ScryfallID != "test-card" {
		t.Errorf("expected scryfall_id 'test-card', got '%s'", result.ScryfallID)
	}
	if result.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", result.Quantity)
	}
}

func TestInventoryGet_WithStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	locationID := location.ID
	item := createTestInventoryItem(t, db, "test-card", 1, &locationID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/inventory/%d", item.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocation == nil {
		t.Fatal("expected storage location to be preloaded")
	}
	if result.StorageLocation.ID != locationID {
		t.Errorf("expected storage location ID %d, got %d", locationID, result.StorageLocation.ID)
	}
}

func TestInventoryGet_NotFound(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/inventory/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestInventoryGet_InvalidID(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/inventory/invalid", nil)
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

func TestInventoryCreate_Success(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{
		"scryfall_id": "test-card-123",
		"oracle_id": "test-oracle-123",
		"treatment": "foil",
		"quantity": 2
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if result.ScryfallID != "test-card-123" {
		t.Errorf("expected scryfall_id 'test-card-123', got '%s'", result.ScryfallID)
	}
	if result.Treatment != "foil" {
		t.Errorf("expected treatment 'foil', got '%s'", result.Treatment)
	}
	if result.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", result.Quantity)
	}
}

func TestInventoryCreate_DefaultQuantity(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{
		"scryfall_id": "test-card",
		"oracle_id": "test-oracle"
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Quantity != 1 {
		t.Errorf("expected default quantity 1, got %d", result.Quantity)
	}
}

func TestInventoryCreate_WithStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{
		"scryfall_id": "test-card",
		"oracle_id": "test-oracle",
		"quantity": 1,
		"storage_location_id": %d
	}`, location.ID)

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID == nil {
		t.Fatal("expected storage location ID to be set")
	}
	if *result.StorageLocationID != location.ID {
		t.Errorf("expected storage location ID %d, got %d", location.ID, *result.StorageLocationID)
	}
}

func TestInventoryCreate_InvalidStorageLocation(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{
		"scryfall_id": "test-card",
		"quantity": 1,
		"storage_location_id": 999
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
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

func TestInventoryCreate_EmptyScryfallID(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{
		"scryfall_id": "",
		"quantity": 1
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
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

func TestInventoryCreate_NegativeQuantity(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{
		"scryfall_id": "test-card",
		"quantity": -1
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
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

func TestInventoryCreate_InvalidJSON(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
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

func TestInventoryUpdate_Success(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "original-card", 1, nil)

	body := `{
		"scryfall_id": "updated-card",
		"treatment": "foil",
		"quantity": 5
	}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/inventory/%d", item.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ScryfallID != "updated-card" {
		t.Errorf("expected scryfall_id 'updated-card', got '%s'", result.ScryfallID)
	}
	if result.Treatment != "foil" {
		t.Errorf("expected treatment 'foil', got '%s'", result.Treatment)
	}
	if result.Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", result.Quantity)
	}
}

func TestInventoryUpdate_PartialUpdate(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "test-card", 1, nil)

	body := `{"quantity": 10}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/inventory/%d", item.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ScryfallID != "test-card" {
		t.Errorf("expected scryfall_id unchanged at 'test-card', got '%s'", result.ScryfallID)
	}
	if result.Quantity != 10 {
		t.Errorf("expected quantity updated to 10, got %d", result.Quantity)
	}
}

func TestInventoryUpdate_SetStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "test-card", 1, nil)
	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{"storage_location_id": %d}`, location.ID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/inventory/%d", item.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID == nil {
		t.Fatal("expected storage location to be set")
	}
	if *result.StorageLocationID != location.ID {
		t.Errorf("expected storage location ID %d, got %d", location.ID, *result.StorageLocationID)
	}
}

func TestInventoryUpdate_ClearStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	locationID := location.ID
	item := createTestInventoryItem(t, db, "test-card", 1, &locationID)

	body := `{"clear_storage": true}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/inventory/%d", item.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID != nil {
		t.Errorf("expected storage location to be cleared, got %d", *result.StorageLocationID)
	}
}

func TestInventoryUpdate_InvalidStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "test-card", 1, nil)

	body := `{"storage_location_id": 999}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/inventory/%d", item.ID), bytes.NewBufferString(body))
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

func TestInventoryUpdate_NotFound(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	body := `{"quantity": 5}`
	req := httptest.NewRequest(http.MethodPut, "/inventory/999", bytes.NewBufferString(body))
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

// Delete endpoint tests

func TestInventoryDelete_Success(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "test-card", 1, nil)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/inventory/%d", item.ID), nil)
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
	db.Model(&models.Inventory{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 inventory items after delete, got %d", count)
	}
}

func TestInventoryDelete_NotFound(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/inventory/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestInventoryDelete_InvalidID(t *testing.T) {
	app, _ := setupInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/inventory/invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestInventoryDelete_DoesNotDeleteStorageLocation(t *testing.T) {
	app, db := setupInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	locationID := location.ID
	item := createTestInventoryItem(t, db, "test-card", 1, &locationID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/inventory/%d", item.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify storage location still exists
	var count int64
	db.Model(&models.StorageLocation{}).Count(&count)
	if count != 1 {
		t.Errorf("expected storage location to remain, got count %d", count)
	}
}

// --- Auto-sort and Resort integration tests ---

func setupInventoryTestAppWithRules(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.StorageLocation{},
		&models.Inventory{},
		&models.Card{},
		&models.SortingRule{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewInventoryHandler(db, services.NewAutoSortService(db))

	app.Get("/inventory", handler.List)
	app.Get("/inventory/:id", handler.Get)
	app.Post("/inventory", handler.Create)
	app.Put("/inventory/:id", handler.Update)
	app.Delete("/inventory/:id", handler.Delete)
	app.Post("/inventory/resort", handler.Resort)

	return app, db
}

func createTestCard(t *testing.T, db *gorm.DB, scryfallID, name, set, rarity, usdPrice string) models.Card {
	t.Helper()
	rawJSON := fmt.Sprintf(`{
		"id": "%s", "name": "%s", "set": "%s", "rarity": "%s",
		"prices": {"usd": "%s", "usd_foil": "", "usd_etched": ""},
		"colors": ["R"], "color_identity": ["R"], "keywords": [],
		"finishes": ["nonfoil"], "promo_types": [],
		"type_line": "Instant", "mana_cost": "{R}", "cmc": 1.0,
		"layout": "normal", "released_at": "1993-08-05"
	}`, scryfallID, name, set, rarity, usdPrice)
	card := models.Card{
		ScryfallID: scryfallID,
		OracleID:   "oracle-" + scryfallID,
		RawJSON:    rawJSON,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}
	return card
}

func createTestSortingRule(t *testing.T, db *gorm.DB, name string, priority int, expression string, locationID uint) models.SortingRule {
	t.Helper()
	rule := models.SortingRule{
		Name:              name,
		Priority:          priority,
		Expression:        expression,
		StorageLocationID: locationID,
		Enabled:           true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create test sorting rule: %v", err)
	}
	return rule
}

// Auto-sort tests

func TestInventoryCreate_AutoSort_RuleMatches(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID)

	body := `{
		"scryfall_id": "bolt-id",
		"oracle_id": "oracle-bolt-id",
		"treatment": "nonfoil"
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID == nil {
		t.Fatal("expected storage location to be auto-assigned by rule")
	}
	if *result.StorageLocationID != location.ID {
		t.Errorf("expected storage location ID %d, got %d", location.ID, *result.StorageLocationID)
	}
}

func TestInventoryCreate_AutoSort_CardNotInDB(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID)
	// No card created in cards table

	body := `{
		"scryfall_id": "missing-card",
		"oracle_id": "oracle-missing",
		"treatment": "nonfoil"
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID != nil {
		t.Errorf("expected nil storage location when card not in DB, got %d", *result.StorageLocationID)
	}
}

func TestInventoryCreate_AutoSort_NoMatchingRule(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestCard(t, db, "expensive-id", "Black Lotus", "lea", "rare", "50000.00")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 1.0", location.ID)

	body := `{
		"scryfall_id": "expensive-id",
		"oracle_id": "oracle-expensive",
		"treatment": "nonfoil"
	}`

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID != nil {
		t.Errorf("expected nil storage location when no rule matches, got %d", *result.StorageLocationID)
	}
}

func TestInventoryCreate_ExplicitLocation_SkipsRules(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	manualBox := models.StorageLocation{Name: "Manual Box", StorageType: models.Box}
	db.Create(&manualBox)
	autoBox := models.StorageLocation{Name: "Auto Box", StorageType: models.Box}
	db.Create(&autoBox)

	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", autoBox.ID)

	body := fmt.Sprintf(`{
		"scryfall_id": "bolt-id",
		"oracle_id": "oracle-bolt-id",
		"treatment": "nonfoil",
		"storage_location_id": %d
	}`, manualBox.ID)

	req := httptest.NewRequest(http.MethodPost, "/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.Inventory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID == nil {
		t.Fatal("expected storage location to be set")
	}
	if *result.StorageLocationID != manualBox.ID {
		t.Errorf("expected manual box ID %d (explicit), got %d", manualBox.ID, *result.StorageLocationID)
	}
}

// Resort tests

func TestResort_RuleMatches_MovedToLocation(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID)

	// Create inventory item with no storage location
	item := createTestInventoryItem(t, db, "bolt-id", 1, nil)

	body := fmt.Sprintf(`{"ids": [%d]}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed 1, got %d", result.Processed)
	}
	if result.Updated != 1 {
		t.Errorf("expected updated 1, got %d", result.Updated)
	}
	if result.Errors != 0 {
		t.Errorf("expected errors 0, got %d", result.Errors)
	}
	if len(result.Movements) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(result.Movements))
	}
	if result.Movements[0].FromLocation != nil {
		t.Errorf("expected from_location nil, got %v", *result.Movements[0].FromLocation)
	}
	if result.Movements[0].ToLocation == nil || *result.Movements[0].ToLocation != location.Name {
		t.Errorf("expected to_location '%s', got %v", location.Name, result.Movements[0].ToLocation)
	}

	// Verify in DB
	var updated models.Inventory
	db.First(&updated, item.ID)
	if updated.StorageLocationID == nil || *updated.StorageLocationID != location.ID {
		t.Errorf("expected DB storage_location_id %d, got %v", location.ID, updated.StorageLocationID)
	}
}

func TestResort_NoMatchingRule_ClearsLocation(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	oldBox := models.StorageLocation{Name: "Old Box", StorageType: models.Box}
	db.Create(&oldBox)

	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	// Rule that won't match (price > 100)
	createTestSortingRule(t, db, "Expensive Only", 1, "prices.usd > 100.0", oldBox.ID)

	// Item currently in "Old Box"
	oldBoxID := oldBox.ID
	item := createTestInventoryItem(t, db, "bolt-id", 1, &oldBoxID)

	body := fmt.Sprintf(`{"ids": [%d]}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed 1, got %d", result.Processed)
	}
	if result.Updated != 1 {
		t.Errorf("expected updated 1, got %d", result.Updated)
	}
	if result.Errors != 0 {
		t.Errorf("expected errors 0, got %d", result.Errors)
	}
	if len(result.Movements) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(result.Movements))
	}
	if result.Movements[0].FromLocation == nil || *result.Movements[0].FromLocation != "Old Box" {
		t.Errorf("expected from_location 'Old Box', got %v", result.Movements[0].FromLocation)
	}
	if result.Movements[0].ToLocation != nil {
		t.Errorf("expected to_location nil (cleared), got %v", *result.Movements[0].ToLocation)
	}

	// Verify in DB
	var updated models.Inventory
	db.First(&updated, item.ID)
	if updated.StorageLocationID != nil {
		t.Errorf("expected DB storage_location_id nil, got %v", *updated.StorageLocationID)
	}
}

func TestResort_CardNotInDB_CountedAsError(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestSortingRule(t, db, "Catch All", 1, "rarity == \"common\"", location.ID)

	// No card in cards table for this scryfall_id
	item := createTestInventoryItem(t, db, "missing-card", 1, nil)

	body := fmt.Sprintf(`{"ids": [%d]}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed 1, got %d", result.Processed)
	}
	if result.Updated != 0 {
		t.Errorf("expected updated 0, got %d", result.Updated)
	}
	if result.Errors != 1 {
		t.Errorf("expected errors 1, got %d", result.Errors)
	}
}

func TestResort_AlreadyInCorrectLocation_NotUpdated(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID)

	// Item already in the correct location
	locationID := location.ID
	item := createTestInventoryItem(t, db, "bolt-id", 1, &locationID)

	body := fmt.Sprintf(`{"ids": [%d]}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed 1, got %d", result.Processed)
	}
	if result.Updated != 0 {
		t.Errorf("expected updated 0 (already correct), got %d", result.Updated)
	}
	if result.Errors != 0 {
		t.Errorf("expected errors 0, got %d", result.Errors)
	}
	if len(result.Movements) != 0 {
		t.Errorf("expected 0 movements, got %d", len(result.Movements))
	}
}

func TestResort_EmptyIDs_ProcessesAll(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	location := createTestStorageLocation(t, db)
	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestCard(t, db, "shock-id", "Shock", "m21", "common", "0.10")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID)

	createTestInventoryItem(t, db, "bolt-id", 1, nil)
	createTestInventoryItem(t, db, "shock-id", 1, nil)

	body := `{"ids": []}`
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Processed != 2 {
		t.Errorf("expected processed 2, got %d", result.Processed)
	}
	if result.Updated != 2 {
		t.Errorf("expected updated 2, got %d", result.Updated)
	}
}

func TestResort_MovementTracking(t *testing.T) {
	app, db := setupInventoryTestAppWithRules(t)

	oldBox := models.StorageLocation{Name: "Old Box", StorageType: models.Box}
	db.Create(&oldBox)
	newBox := models.StorageLocation{Name: "New Box", StorageType: models.Box}
	db.Create(&newBox)

	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "0.25")
	createTestSortingRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", newBox.ID)

	// Item currently in "Old Box"
	oldBoxID := oldBox.ID
	item := createTestInventoryItem(t, db, "bolt-id", 1, &oldBoxID)

	body := fmt.Sprintf(`{"ids": [%d]}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/resort", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ResortResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Movements) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(result.Movements))
	}

	m := result.Movements[0]
	if m.CardName != "Lightning Bolt" {
		t.Errorf("expected card_name 'Lightning Bolt', got '%s'", m.CardName)
	}
	if m.FromLocation == nil || *m.FromLocation != "Old Box" {
		t.Errorf("expected from_location 'Old Box', got %v", m.FromLocation)
	}
	if m.ToLocation == nil || *m.ToLocation != "New Box" {
		t.Errorf("expected to_location 'New Box', got %v", m.ToLocation)
	}
}

// --- ListAsCards, BatchMove, BatchDelete tests ---

func setupFullInventoryTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.StorageLocation{},
		&models.Inventory{},
		&models.Card{},
		&models.SortingRule{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewInventoryHandler(db, services.NewAutoSortService(db))

	// Register all inventory routes matching server/inventory_routes.go
	inventory := app.Group("/inventory")
	inventory.Get("/", handler.List)
	inventory.Get("/cards", handler.ListAsCards)
	inventory.Get("/unassigned/count", handler.GetUnassignedCount)
	inventory.Get("/by-oracle/:oracle_id", handler.ByOracle)
	inventory.Post("/batch/move", handler.BatchMove)
	inventory.Delete("/batch", handler.BatchDelete)
	inventory.Post("/resort", handler.Resort)
	inventory.Get("/:id", handler.Get)
	inventory.Post("/", handler.Create)
	inventory.Put("/:id", handler.Update)
	inventory.Delete("/:id", handler.Delete)

	return app, db
}

func loadTestCardJSON(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to load test card JSON %s: %v", filename, err)
	}
	return string(data)
}

func createTestCardFromFixture(t *testing.T, db *gorm.DB, scryfallID, oracleID, fixture string) models.Card {
	t.Helper()
	rawJSON := loadTestCardJSON(t, fixture)
	card := models.Card{
		ScryfallID: scryfallID,
		OracleID:   oracleID,
		RawJSON:    rawJSON,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("failed to create test card from fixture: %v", err)
	}
	return card
}

// ListAsCards tests

func TestListAsCards_BasicResponse(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "2.00")
	createTestCard(t, db, "shock-id", "Shock", "m21", "common", "0.10")
	createTestInventoryItem(t, db, "bolt-id", 2, nil)
	createTestInventoryItem(t, db, "shock-id", 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalCards != 2 {
		t.Errorf("expected total_cards 2, got %d", result.TotalCards)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
	if result.TotalPages != 1 {
		t.Errorf("expected total_pages 1, got %d", result.TotalPages)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 items in data, got %d", len(result.Data))
	}
}

func TestListAsCards_EnhancedCardFields(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	// Use real fixture for full field coverage
	createTestCardFromFixture(t, db,
		"d573ef03-4730-45aa-93dd-e45ac1dbaf4a",
		"4457ed35-7c10-48c8-9776-456485fdf070",
		"card_lightning_bolt.json")
	createTestInventoryItem(t, db, "d573ef03-4730-45aa-93dd-e45ac1dbaf4a", 3, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Data))
	}

	card := result.Data[0]
	if card.Name != "Lightning Bolt" {
		t.Errorf("expected name 'Lightning Bolt', got '%s'", card.Name)
	}
	if card.SetCode != "lea" {
		t.Errorf("expected set_code 'lea', got '%s'", card.SetCode)
	}
	if card.SetName != "Limited Edition Alpha" {
		t.Errorf("expected set_name 'Limited Edition Alpha', got '%s'", card.SetName)
	}
	if card.CollectorNumber != "161" {
		t.Errorf("expected collector_number '161', got '%s'", card.CollectorNumber)
	}
	if len(card.ColorIdentity) != 1 || card.ColorIdentity[0] != "R" {
		t.Errorf("expected color_identity [R], got %v", card.ColorIdentity)
	}
	if len(card.Finishes) != 1 || card.Finishes[0] != "nonfoil" {
		t.Errorf("expected finishes [nonfoil], got %v", card.Finishes)
	}
	if card.Prices.USD == "" {
		t.Error("expected USD price to be set")
	}
	if card.ImageURI == nil {
		t.Error("expected image_uri to be set")
	}
	if card.Inventory.TotalQuantity != 3 {
		t.Errorf("expected total_quantity 3, got %d", card.Inventory.TotalQuantity)
	}
	if len(card.Inventory.ThisPrinting) != 1 {
		t.Errorf("expected 1 item in this_printing, got %d", len(card.Inventory.ThisPrinting))
	}
}

func TestListAsCards_Pagination(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	// Create 5 distinct cards with inventory
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("card-%d", i)
		createTestCard(t, db, id, fmt.Sprintf("Card %d", i), "tst", "common", "1.00")
		createTestInventoryItem(t, db, id, 1, nil)
	}

	// Page 1 with page_size=2
	req := httptest.NewRequest(http.MethodGet, "/inventory/cards?page=1&page_size=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalCards != 5 {
		t.Errorf("expected total_cards 5, got %d", result.TotalCards)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected total_pages 3, got %d", result.TotalPages)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(result.Data))
	}

	// Page 3 should have 1 item
	req2 := httptest.NewRequest(http.MethodGet, "/inventory/cards?page=3&page_size=2", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	var result2 InventoryCardsResponse
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result2.Data) != 1 {
		t.Errorf("expected 1 item on page 3, got %d", len(result2.Data))
	}
}

func TestListAsCards_FilterByStorageLocation(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	boxA := models.StorageLocation{Name: "Box A", StorageType: models.Box}
	db.Create(&boxA)
	boxB := models.StorageLocation{Name: "Box B", StorageType: models.Box}
	db.Create(&boxB)

	createTestCard(t, db, "card-a1", "Card A1", "tst", "common", "1.00")
	createTestCard(t, db, "card-a2", "Card A2", "tst", "common", "1.00")
	createTestCard(t, db, "card-b1", "Card B1", "tst", "common", "1.00")
	createTestInventoryItem(t, db, "card-a1", 1, &boxA.ID)
	createTestInventoryItem(t, db, "card-a2", 1, &boxA.ID)
	createTestInventoryItem(t, db, "card-b1", 1, &boxB.ID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/inventory/cards?storage_location_id=%d", boxA.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalCards != 2 {
		t.Errorf("expected 2 cards in Box A, got %d", result.TotalCards)
	}
}

func TestListAsCards_FilterByNull(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	box := models.StorageLocation{Name: "Box", StorageType: models.Box}
	db.Create(&box)

	createTestCard(t, db, "assigned-card", "Assigned", "tst", "common", "1.00")
	createTestCard(t, db, "unassigned-card", "Unassigned", "tst", "common", "1.00")
	createTestInventoryItem(t, db, "assigned-card", 1, &box.ID)
	createTestInventoryItem(t, db, "unassigned-card", 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards?storage_location_id=null", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalCards != 1 {
		t.Errorf("expected 1 unassigned card, got %d", result.TotalCards)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 item in data, got %d", len(result.Data))
	}
	if result.Data[0].Name != "Unassigned" {
		t.Errorf("expected card name 'Unassigned', got '%s'", result.Data[0].Name)
	}
}

func TestListAsCards_InvalidLocationID(t *testing.T) {
	app, _ := setupFullInventoryTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards?storage_location_id=abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid location ID, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestListAsCards_CardNotInCardsTable(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	createTestCard(t, db, "good-card", "Good Card", "tst", "common", "1.00")
	createTestInventoryItem(t, db, "good-card", 1, nil)
	// Inventory item with no matching card record
	createTestInventoryItem(t, db, "missing-card", 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// total_cards counts inventory items (2), but data only includes those with card records (1)
	if len(result.Data) != 1 {
		t.Errorf("expected 1 card in data (missing card skipped), got %d", len(result.Data))
	}
	if result.Data[0].Name != "Good Card" {
		t.Errorf("expected card name 'Good Card', got '%s'", result.Data[0].Name)
	}
}

func TestListAsCards_MultipleInventoryPerCard(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	createTestCard(t, db, "bolt-id", "Lightning Bolt", "lea", "common", "2.00")
	// Two inventory items for the same card (different quantities)
	createTestInventoryItem(t, db, "bolt-id", 3, nil)
	createTestInventoryItem(t, db, "bolt-id", 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory/cards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result InventoryCardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 card (grouped), got %d", len(result.Data))
	}

	card := result.Data[0]
	if card.Inventory.TotalQuantity != 5 {
		t.Errorf("expected total_quantity 5 (3+2), got %d", card.Inventory.TotalQuantity)
	}
	if len(card.Inventory.ThisPrinting) != 2 {
		t.Errorf("expected 2 items in this_printing, got %d", len(card.Inventory.ThisPrinting))
	}
}

// BatchMove tests

func TestBatchMove_Success(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	item1 := createTestInventoryItem(t, db, "card-1", 1, nil)
	item2 := createTestInventoryItem(t, db, "card-2", 1, nil)
	createTestInventoryItem(t, db, "card-3", 1, nil) // not moved

	body := fmt.Sprintf(`{"ids": [%d, %d], "storage_location_id": %d}`, item1.ID, item2.ID, location.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result BatchMoveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Updated != 2 {
		t.Errorf("expected updated 2, got %d", result.Updated)
	}

	// Verify DB state
	var updated models.Inventory
	db.First(&updated, item1.ID)
	if updated.StorageLocationID == nil || *updated.StorageLocationID != location.ID {
		t.Errorf("item1 should be in location %d, got %v", location.ID, updated.StorageLocationID)
	}
}

func TestBatchMove_ClearLocation(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	item := createTestInventoryItem(t, db, "card-1", 1, &location.ID)

	body := fmt.Sprintf(`{"ids": [%d], "storage_location_id": null}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result BatchMoveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Updated != 1 {
		t.Errorf("expected updated 1, got %d", result.Updated)
	}

	// Verify location cleared in DB
	var updated models.Inventory
	db.First(&updated, item.ID)
	if updated.StorageLocationID != nil {
		t.Errorf("expected nil storage_location_id, got %v", *updated.StorageLocationID)
	}
}

func TestBatchMove_EmptyIDs(t *testing.T) {
	app, _ := setupFullInventoryTestApp(t)

	body := `{"ids": [], "storage_location_id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for empty IDs, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestBatchMove_LocationNotFound(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	item := createTestInventoryItem(t, db, "card-1", 1, nil)

	body := fmt.Sprintf(`{"ids": [%d], "storage_location_id": 99999}`, item.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for nonexistent location, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestBatchMove_NonexistentItemIDs(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{"ids": [99998, 99999], "storage_location_id": %d}`, location.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result BatchMoveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Updated != 0 {
		t.Errorf("expected updated 0 for nonexistent IDs, got %d", result.Updated)
	}
}

func TestBatchMove_PartialMatch(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	location := createTestStorageLocation(t, db)
	item1 := createTestInventoryItem(t, db, "card-1", 1, nil)
	item2 := createTestInventoryItem(t, db, "card-2", 1, nil)

	// Include one real ID and one nonexistent
	body := fmt.Sprintf(`{"ids": [%d, %d, 99999], "storage_location_id": %d}`, item1.ID, item2.ID, location.ID)
	req := httptest.NewRequest(http.MethodPost, "/inventory/batch/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result BatchMoveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Updated != 2 {
		t.Errorf("expected updated 2 (partial match), got %d", result.Updated)
	}
}

// BatchDelete tests

func TestBatchDelete_Success(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	item1 := createTestInventoryItem(t, db, "card-1", 1, nil)
	item2 := createTestInventoryItem(t, db, "card-2", 1, nil)
	item3 := createTestInventoryItem(t, db, "card-3", 1, nil)

	body := fmt.Sprintf(`{"ids": [%d, %d]}`, item1.ID, item2.ID)
	req := httptest.NewRequest(http.MethodDelete, "/inventory/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result BatchDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Deleted != 2 {
		t.Errorf("expected deleted 2, got %d", result.Deleted)
	}

	// Verify item3 still exists
	var remaining int64
	db.Model(&models.Inventory{}).Count(&remaining)
	if remaining != 1 {
		t.Errorf("expected 1 remaining item, got %d", remaining)
	}
	var item models.Inventory
	db.First(&item, item3.ID)
	if item.ID != item3.ID {
		t.Errorf("expected item3 to survive, but it was deleted")
	}
}

func TestBatchDelete_EmptyIDs(t *testing.T) {
	app, _ := setupFullInventoryTestApp(t)

	body := `{"ids": []}`
	req := httptest.NewRequest(http.MethodDelete, "/inventory/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for empty IDs, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestBatchDelete_NonexistentIDs(t *testing.T) {
	app, _ := setupFullInventoryTestApp(t)

	body := `{"ids": [99998, 99999]}`
	req := httptest.NewRequest(http.MethodDelete, "/inventory/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result BatchDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Deleted != 0 {
		t.Errorf("expected deleted 0, got %d", result.Deleted)
	}
}

func TestBatchDelete_DoesNotAffectOtherItems(t *testing.T) {
	app, db := setupFullInventoryTestApp(t)

	item1 := createTestInventoryItem(t, db, "card-1", 1, nil)
	item2 := createTestInventoryItem(t, db, "card-2", 1, nil)
	item3 := createTestInventoryItem(t, db, "card-3", 1, nil)

	// Only delete item2
	body := fmt.Sprintf(`{"ids": [%d]}`, item2.ID)
	req := httptest.NewRequest(http.MethodDelete, "/inventory/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result BatchDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Deleted != 1 {
		t.Errorf("expected deleted 1, got %d", result.Deleted)
	}

	// Verify items 1 and 3 still exist
	var count int64
	db.Model(&models.Inventory{}).Where("id IN ?", []uint{item1.ID, item3.ID}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 surviving items, got %d", count)
	}

	// Verify item2 is gone
	var deleted models.Inventory
	err = db.First(&deleted, item2.ID).Error
	if err == nil {
		t.Error("expected item2 to be deleted, but it still exists")
	}
}
