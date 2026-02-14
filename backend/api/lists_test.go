package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/models"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupListTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Enable foreign key constraints in SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	if err := db.AutoMigrate(&models.List{}, &models.ListItem{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewListHandler(db)

	app.Get("/lists", handler.List)
	app.Get("/lists/:id", handler.Get)
	app.Post("/lists", handler.Create)
	app.Put("/lists/:id", handler.Update)
	app.Delete("/lists/:id", handler.Delete)

	return app, db
}

func createTestList(t *testing.T, db *gorm.DB, name string) models.List {
	t.Helper()
	list := models.List{
		Name:        name,
		Description: "Test list",
	}

	if err := db.Create(&list).Error; err != nil {
		t.Fatalf("failed to create test list: %v", err)
	}
	return list
}

// List endpoint tests

func TestListList_Empty(t *testing.T) {
	app, _ := setupListTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/lists", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result []ListSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 lists, got %d", len(result))
	}
}

func TestListList_WithLists(t *testing.T) {
	app, db := setupListTestApp(t)

	createTestList(t, db, "List 1")
	createTestList(t, db, "List 2")

	req := httptest.NewRequest(http.MethodGet, "/lists", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result []ListSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 lists, got %d", len(result))
	}
}

// Get endpoint tests

func TestListGet_Success(t *testing.T) {
	app, db := setupListTestApp(t)

	list := createTestList(t, db, "Test List")

	req := httptest.NewRequest(http.MethodGet, "/lists/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.List
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != list.Name {
		t.Errorf("expected name %s, got %s", list.Name, result.Name)
	}
}

func TestListGet_NotFound(t *testing.T) {
	app, _ := setupListTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/lists/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// Create endpoint tests

func TestListCreate_Success(t *testing.T) {
	app, _ := setupListTestApp(t)

	reqBody := CreateListRequest{
		Name:        "My Commander Deck",
		Description: "Test deck",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lists", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result models.List
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != reqBody.Name {
		t.Errorf("expected name %s, got %s", reqBody.Name, result.Name)
	}
}

func TestListCreate_MissingName(t *testing.T) {
	app, _ := setupListTestApp(t)

	reqBody := CreateListRequest{
		Description: "Test",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lists", bytes.NewReader(body))
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

func TestListUpdate_Success(t *testing.T) {
	app, db := setupListTestApp(t)

	list := createTestList(t, db, "Original Name")

	reqBody := UpdateListRequest{
		Name: "Updated Name",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lists/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result models.List
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != reqBody.Name {
		t.Errorf("expected name %s, got %s", reqBody.Name, result.Name)
	}

	if result.ID != list.ID {
		t.Errorf("expected ID %d, got %d", list.ID, result.ID)
	}
}

// Delete endpoint tests

func TestListDelete_Success(t *testing.T) {
	app, db := setupListTestApp(t)

	createTestList(t, db, "Test List")

	req := httptest.NewRequest(http.MethodDelete, "/lists/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify list is deleted
	var count int64
	db.Model(&models.List{}).Count(&count)
	if count != 0 {
		t.Errorf("expected list to be deleted, but found %d lists", count)
	}
}

func TestListDelete_NotFound(t *testing.T) {
	app, _ := setupListTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/lists/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// --- ListItems value calculation and completion percentage tests ---

func setupListTestAppWithCards(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	if err := db.AutoMigrate(&models.List{}, &models.ListItem{}, &models.Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewListHandler(db)

	app.Get("/lists", handler.List)
	app.Get("/lists/:id", handler.Get)
	app.Post("/lists", handler.Create)
	app.Put("/lists/:id", handler.Update)
	app.Delete("/lists/:id", handler.Delete)
	app.Get("/lists/:id/items", handler.ListItems)

	return app, db
}

func createTestCardForList(t *testing.T, db *gorm.DB, scryfallID, name, usdPrice, usdFoilPrice string) models.Card {
	t.Helper()
	rawJSON := fmt.Sprintf(`{
		"id": "%s", "name": "%s", "set": "tst", "set_name": "Test Set",
		"rarity": "rare", "collector_number": "1", "released_at": "2020-01-01",
		"type_line": "Creature", "mana_cost": "{1}", "cmc": 1.0,
		"layout": "normal",
		"prices": {"usd": "%s", "usd_foil": "%s", "usd_etched": ""},
		"colors": [], "color_identity": [], "keywords": [],
		"finishes": ["nonfoil", "foil"], "promo_types": []
	}`, scryfallID, name, usdPrice, usdFoilPrice)
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

func createTestListItem(t *testing.T, db *gorm.DB, listID uint, scryfallID, oracleID, treatment string, desired, collected int) models.ListItem {
	t.Helper()
	item := models.ListItem{
		ListID:            listID,
		ScryfallID:        scryfallID,
		OracleID:          oracleID,
		Treatment:         treatment,
		DesiredQuantity:   desired,
		CollectedQuantity: collected,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create test list item: %v", err)
	}
	return item
}

func TestListItems_ValueCalculation_BasicPrices(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	createTestCardForList(t, db, "bolt-id", "Lightning Bolt", "2.00", "8.00")
	createTestCardForList(t, db, "counterspell-id", "Counterspell", "5.00", "15.00")

	list := createTestList(t, db, "My Deck")
	createTestListItem(t, db, list.ID, "bolt-id", "oracle-bolt-id", "nonfoil", 4, 2)
	createTestListItem(t, db, list.ID, "counterspell-id", "oracle-counterspell-id", "nonfoil", 2, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Collected: (2.00 * 2) + (5.00 * 1) = 9.0
	expectedCollected := 9.0
	if result.TotalCollectedValue != expectedCollected {
		t.Errorf("expected total_collected_value %.2f, got %.2f", expectedCollected, result.TotalCollectedValue)
	}

	// Remaining: (2.00 * 2) + (5.00 * 1) = 9.0
	expectedRemaining := 9.0
	if result.TotalRemainingValue != expectedRemaining {
		t.Errorf("expected total_remaining_value %.2f, got %.2f", expectedRemaining, result.TotalRemainingValue)
	}
}

func TestListItems_ValueCalculation_CardMissingFromDB(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	list := createTestList(t, db, "My Deck")
	// Item references a card that doesn't exist in the cards table
	createTestListItem(t, db, list.ID, "nonexistent-card", "oracle-none", "nonfoil", 4, 2)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.TotalCollectedValue != 0.0 {
		t.Errorf("expected total_collected_value 0.0 for missing card, got %.2f", result.TotalCollectedValue)
	}
	if result.TotalRemainingValue != 0.0 {
		t.Errorf("expected total_remaining_value 0.0 for missing card, got %.2f", result.TotalRemainingValue)
	}
}

func TestListItems_ValueCalculation_FullyCollected(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	createTestCardForList(t, db, "bolt-id", "Lightning Bolt", "10.00", "20.00")

	list := createTestList(t, db, "Complete Deck")
	createTestListItem(t, db, list.ID, "bolt-id", "oracle-bolt-id", "nonfoil", 2, 2)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Collected: 10.00 * 2 = 20.0
	if result.TotalCollectedValue != 20.0 {
		t.Errorf("expected total_collected_value 20.0, got %.2f", result.TotalCollectedValue)
	}
	// Remaining: desired - collected = 0, so no remaining value
	if result.TotalRemainingValue != 0.0 {
		t.Errorf("expected total_remaining_value 0.0 when fully collected, got %.2f", result.TotalRemainingValue)
	}
}

func TestListItems_ValueCalculation_FoilTreatment(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	createTestCardForList(t, db, "bolt-id", "Lightning Bolt", "2.00", "8.00")

	list := createTestList(t, db, "Foil Deck")
	createTestListItem(t, db, list.ID, "bolt-id", "oracle-bolt-id", "foil", 1, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should use foil price (8.00), not nonfoil (2.00)
	if result.TotalCollectedValue != 8.0 {
		t.Errorf("expected total_collected_value 8.0 (foil price), got %.2f", result.TotalCollectedValue)
	}
}

func TestListItems_CompletionPercentage(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	list := createTestList(t, db, "Partial Deck")
	createTestListItem(t, db, list.ID, "card-a", "oracle-a", "nonfoil", 4, 1)
	createTestListItem(t, db, list.ID, "card-b", "oracle-b", "nonfoil", 6, 2)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Total wanted: 4 + 6 = 10, collected: 1 + 2 = 3
	// Completion: (3 * 100) / 10 = 30
	if result.TotalWanted != 10 {
		t.Errorf("expected total_wanted 10, got %d", result.TotalWanted)
	}
	if result.TotalCollected != 3 {
		t.Errorf("expected total_collected 3, got %d", result.TotalCollected)
	}
	if result.CompletionPercent != 30 {
		t.Errorf("expected completion_percent 30, got %d", result.CompletionPercent)
	}
}

func TestListItems_CompletionPercentage_EmptyList(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	list := createTestList(t, db, "Empty Deck")
	// No items added

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.CompletionPercent != 0 {
		t.Errorf("expected completion_percent 0 for empty list, got %d", result.CompletionPercent)
	}
	if result.TotalCollectedValue != 0.0 {
		t.Errorf("expected total_collected_value 0.0, got %.2f", result.TotalCollectedValue)
	}
	if result.TotalRemainingValue != 0.0 {
		t.Errorf("expected total_remaining_value 0.0, got %.2f", result.TotalRemainingValue)
	}
}

func TestListItems_ValueCalculation_MixedCompletion(t *testing.T) {
	app, db := setupListTestAppWithCards(t)

	createTestCardForList(t, db, "bolt-id", "Lightning Bolt", "5.00", "10.00")
	createTestCardForList(t, db, "counter-id", "Counterspell", "3.00", "6.00")

	list := createTestList(t, db, "Mixed Deck")
	// Bolt: fully collected (remaining = 0)
	createTestListItem(t, db, list.ID, "bolt-id", "oracle-bolt-id", "nonfoil", 2, 2)
	// Counterspell: partially collected (remaining = 2)
	createTestListItem(t, db, list.ID, "counter-id", "oracle-counter-id", "nonfoil", 3, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Collected: (5.00 * 2) + (3.00 * 1) = 13.0
	if result.TotalCollectedValue != 13.0 {
		t.Errorf("expected total_collected_value 13.0, got %.2f", result.TotalCollectedValue)
	}
	// Remaining: Bolt has 0 remaining, Counterspell has 2 remaining at 3.00 each = 6.0
	if result.TotalRemainingValue != 6.0 {
		t.Errorf("expected total_remaining_value 6.0, got %.2f", result.TotalRemainingValue)
	}
	// Completion: (2+1)*100 / (2+3) = 300/5 = 60
	if result.CompletionPercent != 60 {
		t.Errorf("expected completion_percent 60, got %d", result.CompletionPercent)
	}
}
