package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/models"
	"backend/services"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInventoryCardsTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	db.Exec("PRAGMA foreign_keys = ON")

	if err := db.AutoMigrate(
		&models.Deck{},
		&models.DeckItem{},
		&models.Inventory{},
		&models.Card{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewInventoryHandler(db, services.NewAutoSortService(db))
	app.Get("/api/inventory/cards", handler.ListAsCards)
	return app, db
}

func addDeckItemDemand(t *testing.T, db *gorm.DB, deckID uint, oracleID string, qty int) {
	t.Helper()
	item := models.DeckItem{
		DeckID:          deckID,
		OracleID:        oracleID,
		DesiredQuantity: qty,
		Zone:            models.ZoneMain,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create deck item: %v", err)
	}
}

type invCardResp struct {
	OracleID string `json:"oracle_id"`
	Name     string `json:"name"`
	Deck     struct {
		Owned  int `json:"owned"`
		Decked int `json:"decked"`
		Free   int `json:"free"`
	} `json:"deck"`
}

type invCardsResp struct {
	Data []invCardResp `json:"data"`
}

// seedAvailabilityFixture creates three cards: untouched (owned, no demand),
// fully decked (owned == demand), and partly decked (owned > demand).
func seedAvailabilityFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	createDeckTestCard(t, db, "sol-1", "oracle-sol", "Sol Ring", "cmm", "rare")
	addDeckInventory(t, db, "oracle-sol", "sol-1", 4)

	createDeckTestCard(t, db, "bolt-1", "oracle-bolt", "Lightning Bolt", "2xm", "uncommon")
	addDeckInventory(t, db, "oracle-bolt", "bolt-1", 4)

	createDeckTestCard(t, db, "path-1", "oracle-path", "Path to Exile", "2xm", "uncommon")
	addDeckInventory(t, db, "oracle-path", "path-1", 8)

	deck := createTestDeck(t, db, "Test Deck")
	addDeckItemDemand(t, db, deck.ID, "oracle-bolt", 4) // fully decked
	addDeckItemDemand(t, db, deck.ID, "oracle-path", 4) // partly decked
}

func fetchInventoryCards(t *testing.T, app *fiber.App, url string) invCardsResp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out invCardsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return out
}

func TestListAsCards_DeckAvailabilityBlock(t *testing.T) {
	app, db := setupInventoryCardsTestApp(t)
	seedAvailabilityFixture(t, db)

	out := fetchInventoryCards(t, app, "/api/inventory/cards")

	byOracle := make(map[string]invCardResp)
	for _, c := range out.Data {
		byOracle[c.OracleID] = c
	}
	if len(byOracle) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(byOracle))
	}

	cases := []struct {
		oracle              string
		owned, decked, free int
	}{
		{"oracle-sol", 4, 0, 4},  // untouched
		{"oracle-bolt", 4, 4, 0}, // fully decked
		{"oracle-path", 8, 4, 4}, // partly decked
	}
	for _, tc := range cases {
		got := byOracle[tc.oracle].Deck
		if got.Owned != tc.owned || got.Decked != tc.decked || got.Free != tc.free {
			t.Errorf("%s: expected owned/decked/free %d/%d/%d, got %d/%d/%d",
				tc.oracle, tc.owned, tc.decked, tc.free, got.Owned, got.Decked, got.Free)
		}
	}
}

func TestListAsCards_DeckAvailableFilter(t *testing.T) {
	app, db := setupInventoryCardsTestApp(t)
	seedAvailabilityFixture(t, db)

	out := fetchInventoryCards(t, app, "/api/inventory/cards?deck_available=true")

	got := make(map[string]bool)
	for _, c := range out.Data {
		got[c.OracleID] = true
	}

	// Untouched and partly-decked cards have a free copy; the fully-decked card does not.
	if !got["oracle-sol"] || !got["oracle-path"] {
		t.Errorf("expected sol + path to be available, got %v", got)
	}
	if got["oracle-bolt"] {
		t.Errorf("fully-decked card should be filtered out, got %v", got)
	}
	if len(out.Data) != 2 {
		t.Errorf("expected 2 available cards, got %d", len(out.Data))
	}
}
