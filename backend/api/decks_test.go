package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/models"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDeckTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
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
	handler := NewDeckHandler(db)

	app.Get("/api/decks", handler.List)
	app.Get("/api/decks/:id", handler.Get)
	app.Post("/api/decks", handler.Create)
	app.Put("/api/decks/:id", handler.Update)
	app.Delete("/api/decks/:id", handler.Delete)
	app.Get("/api/decks/:id/items", handler.ListItems)
	app.Post("/api/decks/:id/items/batch", handler.CreateItemsBatch)
	app.Put("/api/decks/:id/items/:item_id", handler.UpdateItem)
	app.Delete("/api/decks/:id/items/:item_id", handler.DeleteItem)

	return app, db
}

// createDeckTestCard inserts a minimal Card row so deck-item enrichment can
// resolve a name/set for the printing.
func createDeckTestCard(t *testing.T, db *gorm.DB, scryfallID, oracleID, name, set, rarity string) {
	t.Helper()
	rawJSON := fmt.Sprintf(`{
		"id": "%s", "oracle_id": "%s", "name": "%s", "set": "%s", "set_name": "%s Set",
		"collector_number": "1", "rarity": "%s", "finishes": ["nonfoil"],
		"prices": {"usd": "1.00"}, "lang": "en", "layout": "normal", "released_at": "2020-01-01"
	}`, scryfallID, oracleID, name, set, set, rarity)
	card := models.Card{ScryfallID: scryfallID, OracleID: oracleID, RawJSON: rawJSON}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}
}

// addDeckInventory inserts an inventory row for the allocation service.
func addDeckInventory(t *testing.T, db *gorm.DB, oracleID, scryfallID string, qty int) {
	t.Helper()
	inv := models.Inventory{OracleID: oracleID, ScryfallID: scryfallID, Treatment: "nonfoil", Quantity: qty}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("failed to create test inventory: %v", err)
	}
}

func createTestDeck(t *testing.T, db *gorm.DB, name string) models.Deck {
	t.Helper()
	deck := models.Deck{
		Name:        name,
		Description: "Test deck",
	}
	if err := db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create test deck: %v", err)
	}
	return deck
}

// List endpoint tests

func TestDeckList_Empty(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/decks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result []DeckSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 decks, got %d", len(result))
	}
}

func TestDeckList_DemandCountsExcludeMaybe(t *testing.T) {
	app, db := setupDeckTestApp(t)

	deck := createTestDeck(t, db, "Atraxa")
	// Two main-board cards count as demand.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 4,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-2", Zone: models.ZoneSide, DesiredQuantity: 2,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}
	// Maybeboard must NOT contribute to demand totals.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-3", Zone: models.ZoneMaybe, DesiredQuantity: 99,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/decks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result []DeckSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 deck, got %d", len(result))
	}
	got := result[0]
	if got.TotalItems != 3 {
		t.Errorf("expected total_items=3, got %d", got.TotalItems)
	}
	if got.TotalCardsDemand != 6 {
		t.Errorf("expected total_cards_demand=6 (4+2, excluding maybe), got %d", got.TotalCardsDemand)
	}
}

// The summary shortfall must use the same per-Oracle rollup as the items
// endpoint so the two never disagree for a card split across demand zones.
func TestDeckList_ShortfallRollsUpSplitZones(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Split")

	addDeckInventory(t, db, "oracle-bolt", "scry-bolt", 3) // own 3
	// 3 maindeck + 1 sideboard of the same Oracle -> combined demand 4.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-bolt", Zone: models.ZoneMain, DesiredQuantity: 3,
	}).Error; err != nil {
		t.Fatalf("create main item failed: %v", err)
	}
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-bolt", Zone: models.ZoneSide, DesiredQuantity: 1,
	}).Error; err != nil {
		t.Fatalf("create side item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/decks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result []DeckSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 deck, got %d", len(result))
	}
	// Combined demand 4 > owned 3 -> short exactly 1 (not 0 from per-row math).
	if result[0].AggregateShortfall != 1 {
		t.Errorf("expected aggregate_shortfall=1, got %d", result[0].AggregateShortfall)
	}
}

// Get endpoint tests

func TestDeckGet_Found(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Burn")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/decks/%d", deck.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var got models.Deck
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Name != "Burn" {
		t.Errorf("expected name %q, got %q", "Burn", got.Name)
	}
}

func TestDeckGet_NotFound(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/decks/999", nil)
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

func TestDeckCreate_HappyPath(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	body, _ := json.Marshal(CreateDeckRequest{Name: "Modern Burn", Description: "Aggressive red"})
	req := httptest.NewRequest(http.MethodPost, "/api/decks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var got models.Deck
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero deck ID")
	}
	if got.Name != "Modern Burn" {
		t.Errorf("expected name %q, got %q", "Modern Burn", got.Name)
	}
}

func TestDeckCreate_ValidationErrors(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	tests := []struct {
		name string
		req  CreateDeckRequest
	}{
		{name: "empty name", req: CreateDeckRequest{Name: ""}},
		{name: "name too long", req: CreateDeckRequest{Name: strings.Repeat("a", 256)}},
		{name: "description too long", req: CreateDeckRequest{Name: "ok", Description: strings.Repeat("a", 1001)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/api/decks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
			}
		})
	}
}

// Update endpoint tests

func TestDeckUpdate_HappyPath(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Original")

	body, _ := json.Marshal(UpdateDeckRequest{Name: "Renamed", Description: "New description"})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d", deck.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var got models.Deck
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Name != "Renamed" {
		t.Errorf("expected name %q, got %q", "Renamed", got.Name)
	}
	if got.Description != "New description" {
		t.Errorf("expected description %q, got %q", "New description", got.Description)
	}
}

func TestDeckUpdate_ClearsDescription(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Original") // creates with Description "Test deck"

	body, _ := json.Marshal(UpdateDeckRequest{Description: ""})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d", deck.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var got models.Deck
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Description != "" {
		t.Errorf("expected description to be cleared, got %q", got.Description)
	}
}

func TestDeckUpdate_NotFound(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	body, _ := json.Marshal(UpdateDeckRequest{Name: "Renamed"})
	req := httptest.NewRequest(http.MethodPut, "/api/decks/999", bytes.NewReader(body))
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

func TestDeckDelete_CascadesItems(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Doomed")
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/decks/%d", deck.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	var deckCount int64
	if err := db.Model(&models.Deck{}).Where("id = ?", deck.ID).Count(&deckCount).Error; err != nil {
		t.Fatalf("count decks failed: %v", err)
	}
	if deckCount != 0 {
		t.Errorf("expected deck to be deleted, found %d", deckCount)
	}

	var itemCount int64
	if err := db.Model(&models.DeckItem{}).Where("deck_id = ?", deck.ID).Count(&itemCount).Error; err != nil {
		t.Fatalf("count items failed: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("expected items to be deleted, found %d", itemCount)
	}
}

func TestDeckDelete_NotFound(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/decks/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// ListItems tests (allocation-backed)

func TestDeckListItems_GroupsByZoneEnrichesAndComputesShortfall(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Allocated")

	// Card data + inventory for an any-printing main-board card.
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	addDeckInventory(t, db, "oracle-bolt", "scry-bolt", 2)

	// Main: own 2, want 4 -> under-owned by 2.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-bolt", Zone: models.ZoneMain, DesiredQuantity: 4,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}
	// Side: own 0, want 1 -> under-owned by 1.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-side", Zone: models.ZoneSide, DesiredQuantity: 1,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}
	// Maybe: must not contribute to shortfall even though under-owned.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-maybe", Zone: models.ZoneMaybe, DesiredQuantity: 10,
	}).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/decks/%d/items", deck.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var got DeckItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.DeckID != deck.ID {
		t.Errorf("expected deck_id=%d, got %d", deck.ID, got.DeckID)
	}
	if len(got.Main) != 1 || len(got.Side) != 1 || len(got.Maybe) != 1 || len(got.Command) != 0 {
		t.Fatalf("unexpected grouping main=%d side=%d maybe=%d command=%d",
			len(got.Main), len(got.Side), len(got.Maybe), len(got.Command))
	}

	main := got.Main[0]
	if main.Name != "Lightning Bolt" || main.SetCode != "lea" {
		t.Errorf("expected enriched name/set, got name=%q set=%q", main.Name, main.SetCode)
	}
	if main.Owned != 2 {
		t.Errorf("expected main owned=2, got %d", main.Owned)
	}
	if !main.UnderOwned {
		t.Error("expected main item to be under-owned (want 4, own 2)")
	}

	// Shortfall = (4-2) main + (1-0) side = 3; maybe excluded.
	if got.AggregateShortfall != 3 {
		t.Errorf("expected aggregate_shortfall=3, got %d", got.AggregateShortfall)
	}
}

// A card split across two demand zones of one deck (e.g. 3 maindeck + 1
// sideboard) is legitimate; its copies are all committed. Shortfall and
// under-ownership must measure the deck's combined demand for the Oracle
// against what is owned, not each row independently.
func TestDeckListItems_SameOracleSplitAcrossZones(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Split")

	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	addDeckInventory(t, db, "oracle-bolt", "scry-bolt", 3) // own 3

	// 3 maindeck + 1 sideboard of the same card -> combined demand 4, own 3.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-bolt", Zone: models.ZoneMain, DesiredQuantity: 3,
	}).Error; err != nil {
		t.Fatalf("create main item failed: %v", err)
	}
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-bolt", Zone: models.ZoneSide, DesiredQuantity: 1,
	}).Error; err != nil {
		t.Fatalf("create side item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/decks/%d/items", deck.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var got DeckItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(got.Main) != 1 || len(got.Side) != 1 {
		t.Fatalf("unexpected grouping main=%d side=%d", len(got.Main), len(got.Side))
	}

	// Combined demand 4 > owned 3, so both rows are under-owned and the deck is
	// short exactly 1 — not 0 (the per-row bug) and not 2 (double-counting).
	if !got.Main[0].UnderOwned {
		t.Error("expected main row to be under-owned (combined demand 4 > own 3)")
	}
	if !got.Side[0].UnderOwned {
		t.Error("expected side row to be under-owned (combined demand 4 > own 3)")
	}
	if got.AggregateShortfall != 1 {
		t.Errorf("expected aggregate_shortfall=1, got %d", got.AggregateShortfall)
	}
}

// over_committed and under_owned are distinct deck-detail signals and must never
// be conflated (FR98/IMPLEMENTATION_PLAN.md §2 "Two distinct deck-detail
// signals"). over_committed means "you own enough for this deck alone, but not
// across all decks"; it must not fire for a card that is simply under-owned.
func TestDeckListItems_OverCommittedDistinctFromUnderOwned(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deckA := createTestDeck(t, db, "A")
	deckB := createTestDeck(t, db, "B")

	// oracle-bolt: own 4. Deck A wants 3 (fits this deck alone), Deck B also
	// wants 3 -> global demand 6 > owned 4. From Deck A's view the card is NOT
	// under-owned (3 <= 4) but IS over-committed across decks.
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	addDeckInventory(t, db, "oracle-bolt", "scry-bolt", 4)
	if err := db.Create(&models.DeckItem{
		DeckID: deckA.ID, OracleID: "oracle-bolt", Zone: models.ZoneMain, DesiredQuantity: 3,
	}).Error; err != nil {
		t.Fatalf("create deckA bolt item failed: %v", err)
	}
	if err := db.Create(&models.DeckItem{
		DeckID: deckB.ID, OracleID: "oracle-bolt", Zone: models.ZoneMain, DesiredQuantity: 3,
	}).Error; err != nil {
		t.Fatalf("create deckB bolt item failed: %v", err)
	}

	// oracle-orb: Deck A wants 2 but none are owned -> purely under-owned, which
	// must NOT also be reported as over-committed (the regression guard).
	if err := db.Create(&models.DeckItem{
		DeckID: deckA.ID, OracleID: "oracle-orb", Zone: models.ZoneMain, DesiredQuantity: 2,
	}).Error; err != nil {
		t.Fatalf("create deckA orb item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/decks/%d/items", deckA.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var got DeckItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	byOracle := make(map[string]EnrichedDeckItem, len(got.Main))
	for _, item := range got.Main {
		byOracle[item.OracleID] = item
	}

	bolt, ok := byOracle["oracle-bolt"]
	if !ok {
		t.Fatal("expected oracle-bolt in main zone")
	}
	if bolt.UnderOwned {
		t.Error("oracle-bolt: own 4, deck wants 3 -> must not be under-owned")
	}
	if !bolt.OverCommitted {
		t.Error("oracle-bolt: global demand 6 > owned 4 -> must be over-committed")
	}

	orb, ok := byOracle["oracle-orb"]
	if !ok {
		t.Fatal("expected oracle-orb in main zone")
	}
	if !orb.UnderOwned {
		t.Error("oracle-orb: own 0, deck wants 2 -> must be under-owned")
	}
	if orb.OverCommitted {
		t.Error("oracle-orb: under-owned cards must not also be flagged over-committed")
	}
}

func TestDeckListItems_NotFound(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/decks/999/items", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// Batch add item tests

func TestDeckCreateItemsBatch_HappyPath(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Batch")

	body, _ := json.Marshal(CreateDeckItemsBatchRequest{Items: []CreateDeckItemRequest{
		{OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 4},
		{OracleID: "oracle-2", ScryfallID: "scry-2", Treatment: "foil", Zone: models.ZoneSide, DesiredQuantity: 1},
	}})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/items/batch", deck.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var got []models.DeckItem
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}

	var count int64
	db.Model(&models.DeckItem{}).Where("deck_id = ?", deck.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 persisted items, got %d", count)
	}
}

func TestDeckCreateItemsBatch_EmptyItemsRejected(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Empty")

	body, _ := json.Marshal(CreateDeckItemsBatchRequest{Items: []CreateDeckItemRequest{}})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/items/batch", deck.ID), bytes.NewReader(body))
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

func TestDeckCreateItemsBatch_ExceedsMaxRejected(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "TooMany")

	items := make([]CreateDeckItemRequest, MaxBatchItems+1)
	for i := range items {
		items[i] = CreateDeckItemRequest{OracleID: fmt.Sprintf("oracle-%d", i), Zone: models.ZoneMain, DesiredQuantity: 1}
	}
	body, _ := json.Marshal(CreateDeckItemsBatchRequest{Items: items})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/items/batch", deck.ID), bytes.NewReader(body))
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

func TestDeckCreateItemsBatch_InvalidItemRejected(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Invalid")

	// Invalid client input must be a 400, not a 500 from the model hook.
	cases := []struct {
		name string
		item CreateDeckItemRequest
	}{
		{"empty oracle_id", CreateDeckItemRequest{OracleID: "", Zone: models.ZoneMain, DesiredQuantity: 1}},
		{"zero desired_quantity", CreateDeckItemRequest{OracleID: "oracle-x", Zone: models.ZoneMain, DesiredQuantity: 0}},
		{"invalid zone", CreateDeckItemRequest{OracleID: "oracle-x", Zone: models.DeckZone("graveyard"), DesiredQuantity: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(CreateDeckItemsBatchRequest{Items: []CreateDeckItemRequest{tc.item}})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/items/batch", deck.ID), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
			}
		})
	}
}

func TestDeckCreateItemsBatch_DeckNotFound(t *testing.T) {
	app, _ := setupDeckTestApp(t)

	body, _ := json.Marshal(CreateDeckItemsBatchRequest{Items: []CreateDeckItemRequest{
		{OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1},
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/decks/999/items/batch", bytes.NewReader(body))
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

// Update item tests

func TestDeckUpdateItem_QuantityZoneAndPinning(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "UpdateMe")
	item := models.DeckItem{DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	qty := 3
	zone := models.ZoneSide
	scry := "scry-pinned"
	treat := "foil"
	body, _ := json.Marshal(UpdateDeckItemRequest{
		DesiredQuantity: &qty, Zone: &zone, ScryfallID: &scry, Treatment: &treat,
	})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d/items/%d", deck.ID, item.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var got models.DeckItem
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.DesiredQuantity != 3 || got.Zone != models.ZoneSide || got.ScryfallID != "scry-pinned" || got.Treatment != "foil" {
		t.Errorf("unexpected updated item: %+v", got)
	}
}

func TestDeckUpdateItem_NoFieldsRejected(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "UpdateNone")
	item := models.DeckItem{DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	body, _ := json.Marshal(UpdateDeckItemRequest{})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d/items/%d", deck.ID, item.ID), bytes.NewReader(body))
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

func TestDeckUpdateItem_NotFound(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "UpdateMissing")

	qty := 2
	body, _ := json.Marshal(UpdateDeckItemRequest{DesiredQuantity: &qty})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d/items/999", deck.ID), bytes.NewReader(body))
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

func TestDeckUpdateItem_WrongDeckScoped(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deckA := createTestDeck(t, db, "DeckA")
	deckB := createTestDeck(t, db, "DeckB")
	item := models.DeckItem{DeckID: deckA.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	qty := 5
	body, _ := json.Marshal(UpdateDeckItemRequest{DesiredQuantity: &qty})
	// Item belongs to deckA but we address it via deckB -> must 404.
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d/items/%d", deckB.ID, item.ID), bytes.NewReader(body))
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

// Delete item tests

func TestDeckDeleteItem_HappyPath(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "DeleteMe")
	item := models.DeckItem{DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/decks/%d/items/%d", deck.ID, item.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
	var count int64
	db.Model(&models.DeckItem{}).Where("id = ?", item.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected item deleted, found %d", count)
	}
}

func TestDeckDeleteItem_NotFound(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "DeleteMissing")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/decks/%d/items/999", deck.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
