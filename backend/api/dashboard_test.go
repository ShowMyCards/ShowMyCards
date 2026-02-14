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

func setupDashboardTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.StorageLocation{}, &models.List{}, &models.ListItem{}, &models.Inventory{}, &models.Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewDashboardHandler(db)
	app.Get("/dashboard", handler.GetStats)

	return app, db
}

// Test empty database

func TestDashboard_EmptyDatabase(t *testing.T) {
	app, _ := setupDashboardTestApp(t)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.TotalInventoryCards != 0 {
		t.Errorf("expected 0 inventory cards, got %d", stats.TotalInventoryCards)
	}

	if stats.TotalWishlistCards != 0 {
		t.Errorf("expected 0 wishlist cards, got %d", stats.TotalWishlistCards)
	}

	if stats.TotalCollectionValue != 0 {
		t.Errorf("expected 0 collection value, got %f", stats.TotalCollectionValue)
	}

	if stats.TotalCollectedFromLists != 0 {
		t.Errorf("expected 0 collected from lists value, got %f", stats.TotalCollectedFromLists)
	}

	if stats.TotalRemainingListsValue != 0 {
		t.Errorf("expected 0 remaining lists value, got %f", stats.TotalRemainingListsValue)
	}

	if stats.TotalStorageLocations != 0 {
		t.Errorf("expected 0 storage locations, got %d", stats.TotalStorageLocations)
	}

	if stats.TotalLists != 0 {
		t.Errorf("expected 0 lists, got %d", stats.TotalLists)
	}

	if stats.UnassignedCards != 0 {
		t.Errorf("expected 0 unassigned cards, got %d", stats.UnassignedCards)
	}
}

// Test with inventory only

func TestDashboard_WithInventoryOnly(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create storage location
	storage := &models.StorageLocation{
		Name:        "Box 1",
		StorageType: models.Box,
	}
	db.Create(storage)

	// Create card with price
	cardJSON := `{
		"id": "card-1",
		"oracle_id": "oracle-1",
		"name": "Test Card",
		"set": "tst",
		"prices": {
			"usd": "10.50"
		}
	}`
	card := &models.Card{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		RawJSON:    cardJSON,
	}
	db.Create(card)

	// Create inventory items
	inv1 := &models.Inventory{
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "nonfoil",
		Quantity:          4,
		StorageLocationID: &storage.ID,
	}
	db.Create(inv1)

	inv2 := &models.Inventory{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		Treatment:  "foil",
		Quantity:   2,
		// No storage location - unassigned
	}
	db.Create(inv2)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.TotalInventoryCards != 6 {
		t.Errorf("expected 6 inventory cards, got %d", stats.TotalInventoryCards)
	}

	if stats.TotalWishlistCards != 0 {
		t.Errorf("expected 0 wishlist cards, got %d", stats.TotalWishlistCards)
	}

	expectedValue := 10.50 * 6
	if stats.TotalCollectionValue != expectedValue {
		t.Errorf("expected collection value %f, got %f", expectedValue, stats.TotalCollectionValue)
	}

	if stats.TotalCollectedFromLists != 0 {
		t.Errorf("expected 0 collected from lists value, got %f", stats.TotalCollectedFromLists)
	}

	if stats.TotalRemainingListsValue != 0 {
		t.Errorf("expected 0 remaining lists value, got %f", stats.TotalRemainingListsValue)
	}

	if stats.TotalStorageLocations != 1 {
		t.Errorf("expected 1 storage location, got %d", stats.TotalStorageLocations)
	}

	if stats.TotalLists != 0 {
		t.Errorf("expected 0 lists, got %d", stats.TotalLists)
	}

	if stats.UnassignedCards != 2 {
		t.Errorf("expected 2 unassigned cards, got %d", stats.UnassignedCards)
	}
}

// Test with wishlists only

func TestDashboard_WithWishlistsOnly(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create card with price
	cardJSON := `{
		"id": "card-1",
		"oracle_id": "oracle-1",
		"name": "Test Card",
		"set": "tst",
		"prices": {
			"usd": "5.25"
		}
	}`
	card := &models.Card{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		RawJSON:    cardJSON,
	}
	db.Create(card)

	// Create list
	list := &models.List{
		Name:        "Wishlist",
		Description: "Cards I want",
	}
	db.Create(list)

	// Create list items
	item := &models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "nonfoil",
		DesiredQuantity:   4,
		CollectedQuantity: 3,
	}
	db.Create(item)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.TotalInventoryCards != 0 {
		t.Errorf("expected 0 inventory cards, got %d", stats.TotalInventoryCards)
	}

	if stats.TotalWishlistCards != 3 {
		t.Errorf("expected 3 wishlist cards, got %d", stats.TotalWishlistCards)
	}

	if stats.TotalCollectionValue != 0 {
		t.Errorf("expected 0 collection value, got %f", stats.TotalCollectionValue)
	}

	expectedCollectedValue := 5.25 * 3 // 3 collected
	if stats.TotalCollectedFromLists != expectedCollectedValue {
		t.Errorf("expected collected from lists value %f, got %f", expectedCollectedValue, stats.TotalCollectedFromLists)
	}

	expectedRemainingValue := 5.25 * 1 // 1 remaining (4 desired - 3 collected)
	if stats.TotalRemainingListsValue != expectedRemainingValue {
		t.Errorf("expected remaining lists value %f, got %f", expectedRemainingValue, stats.TotalRemainingListsValue)
	}

	if stats.TotalStorageLocations != 0 {
		t.Errorf("expected 0 storage locations, got %d", stats.TotalStorageLocations)
	}

	if stats.TotalLists != 1 {
		t.Errorf("expected 1 list, got %d", stats.TotalLists)
	}

	if stats.UnassignedCards != 0 {
		t.Errorf("expected 0 unassigned cards, got %d", stats.UnassignedCards)
	}
}

// Test with both inventory and wishlists - verify separate counting

func TestDashboard_WithBothInventoryAndWishlists(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create storage location
	storage := &models.StorageLocation{
		Name:        "Box 1",
		StorageType: models.Box,
	}
	db.Create(storage)

	// Create card with price
	cardJSON := `{
		"id": "card-1",
		"oracle_id": "oracle-1",
		"name": "Test Card",
		"set": "tst",
		"prices": {
			"usd": "8.00"
		}
	}`
	card := &models.Card{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		RawJSON:    cardJSON,
	}
	db.Create(card)

	// Create inventory
	inv := &models.Inventory{
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "nonfoil",
		Quantity:          5,
		StorageLocationID: &storage.ID,
	}
	db.Create(inv)

	// Create list with same card
	list := &models.List{
		Name:        "Wishlist",
		Description: "Cards I want",
	}
	db.Create(list)

	item := &models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "foil",
		DesiredQuantity:   4,
		CollectedQuantity: 2,
	}
	db.Create(item)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify separate counting - no addition
	if stats.TotalInventoryCards != 5 {
		t.Errorf("expected 5 inventory cards, got %d", stats.TotalInventoryCards)
	}

	if stats.TotalWishlistCards != 2 {
		t.Errorf("expected 2 wishlist cards, got %d", stats.TotalWishlistCards)
	}

	expectedInventoryValue := 8.00 * 5
	if stats.TotalCollectionValue != expectedInventoryValue {
		t.Errorf("expected collection value %f, got %f", expectedInventoryValue, stats.TotalCollectionValue)
	}

	expectedCollectedValue := 8.00 * 2 // 2 collected
	if stats.TotalCollectedFromLists != expectedCollectedValue {
		t.Errorf("expected collected from lists value %f, got %f", expectedCollectedValue, stats.TotalCollectedFromLists)
	}

	expectedRemainingValue := 8.00 * 2 // 2 remaining (4 desired - 2 collected)
	if stats.TotalRemainingListsValue != expectedRemainingValue {
		t.Errorf("expected remaining lists value %f, got %f", expectedRemainingValue, stats.TotalRemainingListsValue)
	}

	if stats.TotalStorageLocations != 1 {
		t.Errorf("expected 1 storage location, got %d", stats.TotalStorageLocations)
	}

	if stats.TotalLists != 1 {
		t.Errorf("expected 1 list, got %d", stats.TotalLists)
	}

	if stats.UnassignedCards != 0 {
		t.Errorf("expected 0 unassigned cards, got %d", stats.UnassignedCards)
	}
}

// Test value calculations with nil/missing prices

func TestDashboard_WithNilPrices(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Card with no prices object
	card1JSON := `{
		"id": "card-1",
		"oracle_id": "oracle-1",
		"name": "Test Card 1",
		"set": "tst"
	}`
	card1 := &models.Card{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		RawJSON:    card1JSON,
	}
	db.Create(card1)

	// Card with prices but null usd
	card2JSON := `{
		"id": "card-2",
		"oracle_id": "oracle-2",
		"name": "Test Card 2",
		"set": "tst",
		"prices": {
			"usd": null,
			"eur": "5.00"
		}
	}`
	card2 := &models.Card{
		ScryfallID: "card-2",
		OracleID:   "oracle-2",
		RawJSON:    card2JSON,
	}
	db.Create(card2)

	// Card with empty string price
	card3JSON := `{
		"id": "card-3",
		"oracle_id": "oracle-3",
		"name": "Test Card 3",
		"set": "tst",
		"prices": {
			"usd": ""
		}
	}`
	card3 := &models.Card{
		ScryfallID: "card-3",
		OracleID:   "oracle-3",
		RawJSON:    card3JSON,
	}
	db.Create(card3)

	// Create inventory for all cards
	inv1 := &models.Inventory{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		Treatment:  "nonfoil",
		Quantity:   2,
	}
	db.Create(inv1)

	inv2 := &models.Inventory{
		ScryfallID: "card-2",
		OracleID:   "oracle-2",
		Treatment:  "nonfoil",
		Quantity:   3,
	}
	db.Create(inv2)

	inv3 := &models.Inventory{
		ScryfallID: "card-3",
		OracleID:   "oracle-3",
		Treatment:  "nonfoil",
		Quantity:   1,
	}
	db.Create(inv3)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Cards should be counted
	if stats.TotalInventoryCards != 6 {
		t.Errorf("expected 6 inventory cards, got %d", stats.TotalInventoryCards)
	}

	// Value should be 0 (no valid USD prices)
	if stats.TotalCollectionValue != 0 {
		t.Errorf("expected 0 collection value with nil/empty prices, got %f", stats.TotalCollectionValue)
	}
}

// Test mixed data - comprehensive integration test

func TestDashboard_MixedData(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create storage locations
	storage := &models.StorageLocation{
		Name:        "Box 1",
		StorageType: models.Box,
	}
	db.Create(storage)

	// Create cards with different prices
	card1JSON := `{
		"id": "card-1",
		"oracle_id": "oracle-1",
		"name": "Expensive Card",
		"set": "tst",
		"prices": {
			"usd": "50.00"
		}
	}`
	card1 := &models.Card{
		ScryfallID: "card-1",
		OracleID:   "oracle-1",
		RawJSON:    card1JSON,
	}
	db.Create(card1)

	card2JSON := `{
		"id": "card-2",
		"oracle_id": "oracle-2",
		"name": "Cheap Card",
		"set": "tst",
		"prices": {
			"usd": "0.25"
		}
	}`
	card2 := &models.Card{
		ScryfallID: "card-2",
		OracleID:   "oracle-2",
		RawJSON:    card2JSON,
	}
	db.Create(card2)

	// Create inventory with different quantities
	inv1 := &models.Inventory{
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "nonfoil",
		Quantity:          2,
		StorageLocationID: &storage.ID,
	}
	db.Create(inv1)

	inv2 := &models.Inventory{
		ScryfallID: "card-2",
		OracleID:   "oracle-2",
		Treatment:  "foil",
		Quantity:   8,
		// Unassigned
	}
	db.Create(inv2)

	// Create list with items
	list := &models.List{Name: "Wishlist"}
	db.Create(list)

	item := &models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "foil",
		DesiredQuantity:   4,
		CollectedQuantity: 3,
	}
	db.Create(item)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify counts
	if stats.TotalInventoryCards != 10 {
		t.Errorf("expected 10 inventory cards (2+8), got %d", stats.TotalInventoryCards)
	}

	if stats.TotalWishlistCards != 3 {
		t.Errorf("expected 3 wishlist cards, got %d", stats.TotalWishlistCards)
	}

	// Verify values
	expectedInventoryValue := 50.00*2 + 0.25*8 // 100 + 2 = 102
	if stats.TotalCollectionValue != expectedInventoryValue {
		t.Errorf("expected collection value %f, got %f", expectedInventoryValue, stats.TotalCollectionValue)
	}

	expectedCollectedValue := 50.00 * 3 // 3 collected = 150
	if stats.TotalCollectedFromLists != expectedCollectedValue {
		t.Errorf("expected collected from lists value %f, got %f", expectedCollectedValue, stats.TotalCollectedFromLists)
	}

	expectedRemainingValue := 50.00 * 1 // 1 remaining (4 desired - 3 collected) = 50
	if stats.TotalRemainingListsValue != expectedRemainingValue {
		t.Errorf("expected remaining lists value %f, got %f", expectedRemainingValue, stats.TotalRemainingListsValue)
	}

	// Verify storage and lists
	if stats.TotalStorageLocations != 1 {
		t.Errorf("expected 1 storage location, got %d", stats.TotalStorageLocations)
	}

	if stats.TotalLists != 1 {
		t.Errorf("expected 1 list, got %d", stats.TotalLists)
	}

	// Verify unassigned
	if stats.UnassignedCards != 8 {
		t.Errorf("expected 8 unassigned cards, got %d", stats.UnassignedCards)
	}
}

// Test multiple storage locations

func TestDashboard_MultipleStorageLocations(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create multiple storage locations
	storage1 := &models.StorageLocation{Name: "Box 1", StorageType: models.Box}
	storage2 := &models.StorageLocation{Name: "Box 2", StorageType: models.Box}
	storage3 := &models.StorageLocation{Name: "Binder 1", StorageType: models.Binder}
	db.Create(storage1)
	db.Create(storage2)
	db.Create(storage3)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.TotalStorageLocations != 3 {
		t.Errorf("expected 3 storage locations, got %d", stats.TotalStorageLocations)
	}
}

// Test multiple lists

func TestDashboard_MultipleLists(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create multiple lists
	list1 := &models.List{Name: "Wishlist"}
	list2 := &models.List{Name: "Deck 1"}
	list3 := &models.List{Name: "Deck 2"}
	db.Create(list1)
	db.Create(list2)
	db.Create(list3)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.TotalLists != 3 {
		t.Errorf("expected 3 lists, got %d", stats.TotalLists)
	}
}

// Test unassigned cards count

func TestDashboard_UnassignedCardsCount(t *testing.T) {
	app, db := setupDashboardTestApp(t)

	// Create storage location
	storage := &models.StorageLocation{
		Name:        "Box 1",
		StorageType: models.Box,
	}
	db.Create(storage)

	// Create some assigned inventory
	inv1 := &models.Inventory{
		ScryfallID:        "card-1",
		OracleID:          "oracle-1",
		Treatment:         "nonfoil",
		Quantity:          3,
		StorageLocationID: &storage.ID,
	}
	db.Create(inv1)

	// Create unassigned inventory
	inv2 := &models.Inventory{
		ScryfallID: "card-2",
		OracleID:   "oracle-2",
		Treatment:  "nonfoil",
		Quantity:   5,
		// No storage location
	}
	db.Create(inv2)

	inv3 := &models.Inventory{
		ScryfallID: "card-3",
		OracleID:   "oracle-3",
		Treatment:  "foil",
		Quantity:   2,
		// No storage location
	}
	db.Create(inv3)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var stats DashboardStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if stats.UnassignedCards != 7 {
		t.Errorf("expected 7 unassigned cards (5+2), got %d", stats.UnassignedCards)
	}

	if stats.TotalInventoryCards != 10 {
		t.Errorf("expected 10 total inventory cards (3+5+2), got %d", stats.TotalInventoryCards)
	}
}
