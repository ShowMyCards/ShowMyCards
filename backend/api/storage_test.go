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

	app.Get("/api/storage", handler.List)
	app.Get("/api/storage/with-counts", handler.ListWithCounts)
	app.Get("/api/storage/:id", handler.Get)
	app.Post("/api/storage", handler.Create)
	app.Post("/api/storage/batch/sort-order", handler.BatchUpdateSortOrder)
	app.Put("/api/storage/:id", handler.Update)
	app.Delete("/api/storage/:id", handler.Delete)

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

	req := httptest.NewRequest(http.MethodGet, "/api/storage", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/storage", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/storage?page=2&page_size=2", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/storage?page_size=200", nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/storage/%d", location.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/storage/999", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/storage/invalid", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/api/storage", bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPost, "/api/storage", bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPost, "/api/storage", bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPost, "/api/storage", bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/storage/%d", location.ID), bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPut, "/api/storage/999", bytes.NewBufferString(body))
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
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/storage/%d", location.ID), bytes.NewBufferString(body))
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

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/storage/%d", location.ID), nil)
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

	req := httptest.NewRequest(http.MethodDelete, "/api/storage/999", nil)
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

	req := httptest.NewRequest(http.MethodDelete, "/api/storage/invalid", nil)
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
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/storage/%d", location.ID), nil)
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
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/storage/%d", location.ID), nil)
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
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/storage/%d", location.ID), nil)
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

func TestCreate_AssignsIncrementingSortOrder(t *testing.T) {
	app, _ := setupTestApp(t)

	names := []string{"First", "Second", "Third"}
	for i, name := range names {
		body := fmt.Sprintf(`{"name": %q, "storage_type": "Box"}`, name)
		req := httptest.NewRequest(http.MethodPost, "/api/storage", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		var created models.StorageLocation
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		if created.SortOrder != i+1 {
			t.Errorf("created %q: expected sort_order %d, got %d", name, i+1, created.SortOrder)
		}
	}
}

func TestUpdate_PreservesSortOrder(t *testing.T) {
	app, db := setupTestApp(t)

	loc := models.StorageLocation{Name: "Original", StorageType: models.Box, SortOrder: 7}
	if err := db.Create(&loc).Error; err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	body := `{"name": "Renamed", "storage_type": "Binder"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/storage/%d", loc.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var updated models.StorageLocation
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if updated.SortOrder != 7 {
		t.Errorf("expected sort_order 7 to be preserved, got %d", updated.SortOrder)
	}
}

func TestListAndListWithCounts_OrderedBySortOrder(t *testing.T) {
	app, db := setupTestApp(t)

	// Mix Box and Binder intentionally with manual sort orders
	locs := []models.StorageLocation{
		{Name: "Third Box", StorageType: models.Box, SortOrder: 3},
		{Name: "First Binder", StorageType: models.Binder, SortOrder: 1},
		{Name: "Second Box", StorageType: models.Box, SortOrder: 2},
	}
	for i := range locs {
		if err := db.Create(&locs[i]).Error; err != nil {
			t.Fatalf("failed to create location: %v", err)
		}
	}

	// Check /api/storage
	req := httptest.NewRequest(http.MethodGet, "/api/storage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var listResult utils.PaginatedResponse[models.StorageLocation]
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	wantNames := []string{"First Binder", "Second Box", "Third Box"}
	if len(listResult.Data) != len(wantNames) {
		t.Fatalf("expected %d items from /api/storage, got %d", len(wantNames), len(listResult.Data))
	}
	for i, want := range wantNames {
		if listResult.Data[i].Name != want {
			t.Errorf("/api/storage[%d] = %q, want %q", i, listResult.Data[i].Name, want)
		}
	}

	// Check /api/storage/with-counts (unified order across Box and Binder)
	reqCounts := httptest.NewRequest(http.MethodGet, "/api/storage/with-counts", nil)
	respCounts, err := app.Test(reqCounts)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respCounts.Body.Close()

	if respCounts.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 from with-counts, got %d", respCounts.StatusCode)
	}

	var countResults []StorageLocationWithCount
	if err := json.NewDecoder(respCounts.Body).Decode(&countResults); err != nil {
		t.Fatalf("failed to decode with-counts response: %v", err)
	}
	if len(countResults) != len(wantNames) {
		t.Fatalf("expected %d items from with-counts, got %d", len(wantNames), len(countResults))
	}
	for i, want := range wantNames {
		if countResults[i].Name != want {
			t.Errorf("with-counts[%d] = %q, want %q", i, countResults[i].Name, want)
		}
		if countResults[i].SortOrder != i+1 {
			t.Errorf("with-counts[%d].SortOrder = %d, want %d", i, countResults[i].SortOrder, i+1)
		}
	}
}

func TestBatchUpdateSortOrder_Success(t *testing.T) {
	app, db := setupTestApp(t)

	a := models.StorageLocation{Name: "A", StorageType: models.Box, SortOrder: 1}
	b := models.StorageLocation{Name: "B", StorageType: models.Binder, SortOrder: 2}
	c := models.StorageLocation{Name: "C", StorageType: models.Box, SortOrder: 3}
	db.Create(&a)
	db.Create(&b)
	db.Create(&c)

	body := fmt.Sprintf(`{"updates": [{"id": %d, "sort_order": 1}, {"id": %d, "sort_order": 2}, {"id": %d, "sort_order": 3}]}`, c.ID, a.ID, b.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/storage/batch/sort-order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var res BatchUpdateSortOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.UpdatedCount != 3 {
		t.Errorf("expected updated_count 3, got %d", res.UpdatedCount)
	}

	var ordered []models.StorageLocation
	db.Order("sort_order ASC").Find(&ordered)
	want := []string{"C", "A", "B"}
	for i, w := range want {
		if ordered[i].Name != w {
			t.Errorf("pos %d: got %q, want %q", i, ordered[i].Name, w)
		}
	}
}

func TestBatchUpdateSortOrder_ValidationAndRollback(t *testing.T) {
	app, db := setupTestApp(t)

	a := models.StorageLocation{Name: "A", StorageType: models.Box, SortOrder: 1}
	b := models.StorageLocation{Name: "B", StorageType: models.Binder, SortOrder: 2}
	db.Create(&a)
	db.Create(&b)

	badRequests := []struct {
		name string
		body string
	}{
		{
			name: "empty updates",
			body: `{"updates": []}`,
		},
		{
			name: "zero id",
			body: `{"updates": [{"id": 0, "sort_order": 1}]}`,
		},
		{
			name: "sort_order zero",
			body: fmt.Sprintf(`{"updates": [{"id": %d, "sort_order": 0}]}`, a.ID),
		},
		{
			name: "negative sort_order",
			body: fmt.Sprintf(`{"updates": [{"id": %d, "sort_order": -1}]}`, a.ID),
		},
		{
			name: "duplicate ids",
			body: fmt.Sprintf(`{"updates": [{"id": %d, "sort_order": 5}, {"id": %d, "sort_order": 2}, {"id": %d, "sort_order": 3}]}`, b.ID, a.ID, a.ID),
		},
	}

	for _, tc := range badRequests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/storage/batch/sort-order", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}

			var reloadedA, reloadedB models.StorageLocation
			db.First(&reloadedA, a.ID)
			db.First(&reloadedB, b.ID)
			if reloadedA.SortOrder != 1 || reloadedB.SortOrder != 2 {
				t.Errorf("expected sort orders to remain (1, 2), got (%d, %d)", reloadedA.SortOrder, reloadedB.SortOrder)
			}
		})
	}

	// Nonexistent ID in batch -> 404 and rollback earlier update in transaction
	t.Run("nonexistent id rolls back", func(t *testing.T) {
		reqMissing := httptest.NewRequest(http.MethodPost, "/api/storage/batch/sort-order", bytes.NewBufferString(fmt.Sprintf(`{"updates": [{"id": %d, "sort_order": 9}, {"id": 99999, "sort_order": 2}]}`, a.ID)))
		reqMissing.Header.Set("Content-Type", "application/json")
		respMissing, err := app.Test(reqMissing)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer respMissing.Body.Close()
		if respMissing.StatusCode != http.StatusNotFound {
			t.Errorf("missing ID: expected 404, got %d", respMissing.StatusCode)
		}

		var reloaded models.StorageLocation
		db.First(&reloaded, a.ID)
		if reloaded.SortOrder != 1 {
			t.Errorf("expected transaction rollback to keep sort_order 1, got %d", reloaded.SortOrder)
		}
	})
}
