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

	app.Get("/api/lists", handler.List)
	app.Get("/api/lists/:id", handler.Get)
	app.Post("/api/lists", handler.Create)
	app.Put("/api/lists/:id", handler.Update)
	app.Delete("/api/lists/:id", handler.Delete)

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

	req := httptest.NewRequest(http.MethodGet, "/api/lists", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/lists", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/lists/1", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/lists/999", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewReader(body))
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
	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewReader(body))
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
	req := httptest.NewRequest(http.MethodPut, "/api/lists/1", bytes.NewReader(body))
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

	req := httptest.NewRequest(http.MethodDelete, "/api/lists/1", nil)
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

	req := httptest.NewRequest(http.MethodDelete, "/api/lists/999", nil)
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

	app.Get("/api/lists", handler.List)
	app.Get("/api/lists/:id", handler.Get)
	app.Post("/api/lists", handler.Create)
	app.Put("/api/lists/:id", handler.Update)
	app.Delete("/api/lists/:id", handler.Delete)
	app.Get("/api/lists/:id/items", handler.ListItems)

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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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

// setupListTestAppWithPrintLookup builds a list test app whose cards table has
// the generated columns (set_code, collector_number, lang) and index that the
// English-price fallback (models.GetEnglishPricesByPrint) relies on, mirroring
// the production migration.
func setupListTestAppWithPrintLookup(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	if err := db.AutoMigrate(&models.List{}, &models.ListItem{}, &models.Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE cards ADD COLUMN set_code TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.set')) VIRTUAL`,
		`ALTER TABLE cards ADD COLUMN collector_number TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.collector_number')) VIRTUAL`,
		`ALTER TABLE cards ADD COLUMN lang TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.lang')) VIRTUAL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to add generated column: %v", err)
		}
	}

	app := fiber.New()
	handler := NewListHandler(db)
	app.Get("/api/lists/:id/items", handler.ListItems)

	return app, db
}

// seedCardWithPrint inserts a card with explicit set, collector number, language
// and prices so the English-price fallback can be exercised.
func seedCardWithPrint(t *testing.T, db *gorm.DB, scryfallID, set, collectorNumber, lang, usd, usdFoil string) {
	t.Helper()
	rawJSON := fmt.Sprintf(`{
		"id": "%s", "name": "Sol Ring", "set": "%s", "set_name": "Commander 2021",
		"rarity": "uncommon", "collector_number": "%s", "lang": "%s",
		"released_at": "2021-04-23", "type_line": "Artifact", "mana_cost": "{1}",
		"cmc": 1.0, "layout": "normal",
		"prices": {"usd": "%s", "usd_foil": "%s", "usd_etched": ""},
		"colors": [], "color_identity": [], "keywords": [],
		"finishes": ["nonfoil", "foil"], "promo_types": []
	}`, scryfallID, set, collectorNumber, lang, usd, usdFoil)
	card := models.Card{
		ScryfallID: scryfallID,
		OracleID:   "oracle-sol-ring",
		RawJSON:    rawJSON,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}
}

// TestListItems_EnglishPriceFallback verifies that a non-English list item with
// no price of its own is back-filled from the English printing of the same set +
// collector number (both for the displayed price and for value totals), while an
// already-priced English item is left untouched.
func TestListItems_EnglishPriceFallback(t *testing.T) {
	app, db := setupListTestAppWithPrintLookup(t)

	// English printing carries the price; the German printing of the same
	// set + collector number has empty prices (as Scryfall ships them).
	seedCardWithPrint(t, db, "en-sol", "c21", "263", "en", "1.73", "5.00")
	seedCardWithPrint(t, db, "de-sol", "c21", "263", "de", "", "")

	list := createTestList(t, db, "Mixed Language Deck")
	// Non-English item: should be back-filled to the English price (1.73).
	createTestListItem(t, db, list.ID, "de-sol", "oracle-sol-ring", "nonfoil", 4, 2)
	// English item: already priced, must stay at its own price (1.73).
	createTestListItem(t, db, list.ID, "en-sol", "oracle-sol-ring", "nonfoil", 2, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Find the enriched items by Scryfall ID.
	prices := make(map[string]float64, len(result.Data))
	for _, item := range result.Data {
		prices[item.ScryfallID] = item.CurrentPrice
	}

	if prices["de-sol"] != 1.73 {
		t.Errorf("non-english item: expected back-filled current_price 1.73, got %.2f", prices["de-sol"])
	}
	if prices["en-sol"] != 1.73 {
		t.Errorf("english item: expected untouched current_price 1.73, got %.2f", prices["en-sol"])
	}

	// Value totals must include the backed-off price for the German item.
	// Collected: de (1.73 * 2) + en (1.73 * 1) = 5.19
	if !floatsClose(result.TotalCollectedValue, 5.19) {
		t.Errorf("expected total_collected_value 5.19, got %.4f", result.TotalCollectedValue)
	}
	// Remaining: de has 2 remaining (1.73 * 2 = 3.46); en has 1 remaining (1.73). Total 5.19.
	if !floatsClose(result.TotalRemainingValue, 5.19) {
		t.Errorf("expected total_remaining_value 5.19, got %.4f", result.TotalRemainingValue)
	}
}

// floatsClose reports whether two float values are equal within a small epsilon,
// to avoid spurious failures from floating-point accumulation in value totals.
func floatsClose(a, b float64) bool {
	const epsilon = 1e-9
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

// TestListItems_NonEnglishWithOwnPrice verifies that a non-English item that
// does carry its own price keeps it and is not overwritten by the fallback.
func TestListItems_NonEnglishWithOwnPrice(t *testing.T) {
	app, db := setupListTestAppWithPrintLookup(t)

	// English printing priced at 1.73, but the German printing has its own 9.99.
	seedCardWithPrint(t, db, "en-sol", "c21", "263", "en", "1.73", "5.00")
	seedCardWithPrint(t, db, "de-sol", "c21", "263", "de", "9.99", "")

	list := createTestList(t, db, "German Deck")
	createTestListItem(t, db, list.ID, "de-sol", "oracle-sol-ring", "nonfoil", 1, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ListItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Data))
	}
	if result.Data[0].CurrentPrice != 9.99 {
		t.Errorf("non-english item with own price: expected 9.99 untouched, got %.2f", result.Data[0].CurrentPrice)
	}
	if result.TotalCollectedValue != 9.99 {
		t.Errorf("expected total_collected_value 9.99, got %.2f", result.TotalCollectedValue)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/items", list.ID), nil)
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
