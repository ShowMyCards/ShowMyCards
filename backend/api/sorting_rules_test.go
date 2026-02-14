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

func setupSortingRulesTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.StorageLocation{}, &models.SortingRule{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewSortingRulesHandler(db)

	app.Get("/sorting-rules", handler.List)
	app.Get("/sorting-rules/:id", handler.Get)
	app.Post("/sorting-rules", handler.Create)
	app.Put("/sorting-rules/:id", handler.Update)
	app.Delete("/sorting-rules/:id", handler.Delete)

	return app, db
}

func createTestStorageLocation(t *testing.T, db *gorm.DB) models.StorageLocation {
	t.Helper()
	location := models.StorageLocation{
		Name:        "Test Box",
		StorageType: models.Box,
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to create test storage location: %v", err)
	}
	return location
}

func createTestRule(t *testing.T, db *gorm.DB, name string, priority int, expression string, locationID uint) models.SortingRule {
	t.Helper()
	rule := models.SortingRule{
		Name:              name,
		Priority:          priority,
		Expression:        expression,
		StorageLocationID: locationID,
		Enabled:           true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create test rule: %v", err)
	}
	return rule
}

// List endpoint tests

func TestSortingRulesList_Empty(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules", nil)
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

func TestSortingRulesList_WithItems(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	createTestRule(t, db, "Rule 1", 1, "prices.usd < 5.0", location.ID)
	createTestRule(t, db, "Rule 2", 2, "rarity == 'mythic'", location.ID)

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules", nil)
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

func TestSortingRulesList_OrderByPriority(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	// Create in reverse priority order
	createTestRule(t, db, "Rule High", 3, "test3", location.ID)
	createTestRule(t, db, "Rule Low", 1, "test1", location.ID)
	createTestRule(t, db, "Rule Mid", 2, "test2", location.ID)

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Extract rules from the Data field
	dataBytes, _ := json.Marshal(result.Data)
	var rules []models.SortingRule
	json.Unmarshal(dataBytes, &rules)

	// Verify order
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
	if rules[0].Priority != 1 || rules[1].Priority != 2 || rules[2].Priority != 3 {
		t.Errorf("rules not ordered by priority: got %d, %d, %d", rules[0].Priority, rules[1].Priority, rules[2].Priority)
	}
}

func TestSortingRulesList_FilterEnabled(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	createTestRule(t, db, "Enabled Rule", 1, "test1", location.ID)

	rule2 := createTestRule(t, db, "Disabled Rule", 2, "test2", location.ID)
	rule2.Enabled = false
	db.Save(&rule2)

	// Test enabled=true filter
	req := httptest.NewRequest(http.MethodGet, "/sorting-rules?enabled=true", nil)
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
		t.Errorf("expected 1 enabled rule, got %d", result.TotalItems)
	}

	// Test enabled=false filter
	req2 := httptest.NewRequest(http.MethodGet, "/sorting-rules?enabled=false", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	var result2 utils.PaginatedResponse[json.RawMessage]
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result2.TotalItems != 1 {
		t.Errorf("expected 1 disabled rule, got %d", result2.TotalItems)
	}
}

func TestSortingRulesList_Pagination(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	for i := 1; i <= 5; i++ {
		createTestRule(t, db, fmt.Sprintf("Rule %d", i), i, "test", location.ID)
	}

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules?page=2&page_size=2", nil)
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

func TestSortingRulesGet_Success(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test Rule", 1, "prices.usd < 10.0", location.ID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sorting-rules/%d", rule.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID != rule.ID {
		t.Errorf("expected ID %d, got %d", rule.ID, result.ID)
	}
	if result.Name != "Test Rule" {
		t.Errorf("expected name 'Test Rule', got '%s'", result.Name)
	}
	if result.Expression != "prices.usd < 10.0" {
		t.Errorf("expected expression 'prices.usd < 10.0', got '%s'", result.Expression)
	}
}

func TestSortingRulesGet_NotFound(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSortingRulesGet_InvalidID(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sorting-rules/invalid", nil)
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

func TestSortingRulesCreate_Success(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{
		"name": "Chaff Cards",
		"priority": 1,
		"expression": "prices.usd < 5.0",
		"storage_location_id": %d,
		"enabled": true
	}`, location.ID)

	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if result.Name != "Chaff Cards" {
		t.Errorf("expected name 'Chaff Cards', got '%s'", result.Name)
	}
	if result.Priority != 1 {
		t.Errorf("expected priority 1, got %d", result.Priority)
	}
	if result.Expression != "prices.usd < 5.0" {
		t.Errorf("expected expression 'prices.usd < 5.0', got '%s'", result.Expression)
	}
	if !result.Enabled {
		t.Error("expected rule to be enabled")
	}
}

func TestSortingRulesCreate_DefaultEnabled(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{
		"name": "Test Rule",
		"priority": 1,
		"expression": "prices.usd < 5.0",
		"storage_location_id": %d
	}`, location.ID)

	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Enabled {
		t.Errorf("expected rule to be enabled by default, got Enabled=%v (ID=%d, Name=%s)", result.Enabled, result.ID, result.Name)
	}
}

func TestSortingRulesCreate_InvalidStorageLocation(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	body := `{
		"name": "Test Rule",
		"priority": 1,
		"expression": "test",
		"storage_location_id": 999
	}`

	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
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

func TestSortingRulesCreate_EmptyName(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{
		"name": "",
		"priority": 1,
		"expression": "prices.usd < 5.0",
		"storage_location_id": %d
	}`, location.ID)

	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
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

func TestSortingRulesCreate_EmptyExpression(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)

	body := fmt.Sprintf(`{
		"name": "Test",
		"priority": 1,
		"expression": "",
		"storage_location_id": %d
	}`, location.ID)

	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
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

func TestSortingRulesCreate_InvalidJSON(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/sorting-rules", bytes.NewBufferString(body))
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

func TestSortingRulesUpdate_Success(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Original Name", 1, "original", location.ID)

	body := `{
		"name": "Updated Name",
		"priority": 2,
		"expression": "updated"
	}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sorting-rules/%d", rule.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
	if result.Priority != 2 {
		t.Errorf("expected priority 2, got %d", result.Priority)
	}
	if result.Expression != "updated" {
		t.Errorf("expected expression 'updated', got '%s'", result.Expression)
	}
}

func TestSortingRulesUpdate_PartialUpdate(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Original Name", 1, "original", location.ID)

	body := `{"name": "New Name"}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sorting-rules/%d", rule.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", result.Name)
	}
	if result.Priority != 1 {
		t.Errorf("expected priority unchanged at 1, got %d", result.Priority)
	}
	if result.Expression != "original" {
		t.Errorf("expected expression unchanged at 'original', got '%s'", result.Expression)
	}
}

func TestSortingRulesUpdate_DisableRule(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test", 1, "test", location.ID)

	body := `{"enabled": false}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sorting-rules/%d", rule.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Enabled {
		t.Error("expected rule to be disabled")
	}
}

func TestSortingRulesUpdate_ChangeStorageLocation(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location1 := createTestStorageLocation(t, db)
	location2 := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test", 1, "test", location1.ID)

	body := fmt.Sprintf(`{"storage_location_id": %d}`, location2.ID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sorting-rules/%d", rule.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.SortingRule
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationID != location2.ID {
		t.Errorf("expected storage location ID %d, got %d", location2.ID, result.StorageLocationID)
	}
}

func TestSortingRulesUpdate_InvalidStorageLocation(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test", 1, "test", location.ID)

	body := `{"storage_location_id": 999}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sorting-rules/%d", rule.ID), bytes.NewBufferString(body))
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

func TestSortingRulesUpdate_NotFound(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/sorting-rules/999", bytes.NewBufferString(body))
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

func TestSortingRulesDelete_Success(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test", 1, "test", location.ID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sorting-rules/%d", rule.ID), nil)
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
	db.Model(&models.SortingRule{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rules after delete, got %d", count)
	}
}

func TestSortingRulesDelete_NotFound(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/sorting-rules/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSortingRulesDelete_InvalidID(t *testing.T) {
	app, _ := setupSortingRulesTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/sorting-rules/invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestSortingRulesDelete_DoesNotDeleteStorageLocation(t *testing.T) {
	app, db := setupSortingRulesTestApp(t)

	location := createTestStorageLocation(t, db)
	rule := createTestRule(t, db, "Test", 1, "test", location.ID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sorting-rules/%d", rule.ID), nil)
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
