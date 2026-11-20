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
	"gorm.io/gorm"
)

// Tokens and other extras live in the "other" zone, are never counted as cards,
// and get their own ownership signal (token_shortfall / token_over_committed).

// createDeckTestToken inserts a token card (layout "token").
func createDeckTestToken(t *testing.T, db *gorm.DB, scryfallID, oracleID, name string) {
	t.Helper()
	rawJSON := fmt.Sprintf(`{
		"id": %q, "oracle_id": %q, "name": %q, "set": "tblb", "set_name": "Bloomburrow Tokens",
		"collector_number": "1", "rarity": "common", "finishes": ["nonfoil"], "lang": "en",
		"layout": "token", "type_line": "Token Creature — Rabbit", "released_at": "2024-08-02"
	}`, scryfallID, oracleID, name)
	if err := db.Create(&models.Card{ScryfallID: scryfallID, OracleID: oracleID, RawJSON: rawJSON}).Error; err != nil {
		t.Fatalf("failed to create test token: %v", err)
	}
}

func seedPinnedDeckItem(t *testing.T, db *gorm.DB, deckID uint, oracleID, scryfallID string, zone models.DeckZone, qty int) models.DeckItem {
	t.Helper()
	item := models.DeckItem{DeckID: deckID, OracleID: oracleID, ScryfallID: scryfallID, Zone: zone, DesiredQuantity: qty}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create deck item for %s: %v", oracleID, err)
	}
	return item
}

func putDeckItem(t *testing.T, app *fiber.App, deckID, itemID uint, req UpdateDeckItemRequest) *http.Response {
	t.Helper()
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/decks/%d/items/%d", deckID, itemID), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestDeckCreateItemsBatch_TokenGoesToOther(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")

	resp := postDeckBatch(t, app, deck.ID, []CreateDeckItemRequest{
		{OracleID: "oracle-rabbit", ScryfallID: "scry-rabbit", Zone: models.ZoneMain, DesiredQuantity: 2},
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var got []models.DeckItem
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 1 || got[0].Zone != models.ZoneOther {
		t.Errorf("response items = %+v, want one item in zone other", got)
	}
	if zones := deckZoneQuantities(t, db, deck.ID); zones[models.ZoneOther] != 2 || len(zones) != 1 {
		t.Errorf("persisted zones = %v, want only other: 2", zones)
	}
}

func TestDeckCreateItemsBatch_AnyPrintingTokenGoesToOther(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")

	resp := postDeckBatch(t, app, deck.ID, []CreateDeckItemRequest{
		{OracleID: "oracle-rabbit", Zone: models.ZoneSide, DesiredQuantity: 1},
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	if zones := deckZoneQuantities(t, db, deck.ID); zones[models.ZoneOther] != 1 || len(zones) != 1 {
		t.Errorf("persisted zones = %v, want only other: 1", zones)
	}
}

// The same token listed under two sections of an import collapses into one row.
func TestDeckCreateItemsBatch_TokensFromSeveralZonesMerge(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")

	resp := postDeckBatch(t, app, deck.ID, []CreateDeckItemRequest{
		{OracleID: "oracle-rabbit", ScryfallID: "scry-rabbit", Zone: models.ZoneMain, DesiredQuantity: 2},
		{OracleID: "oracle-rabbit", ScryfallID: "scry-rabbit", Zone: models.ZoneSide, DesiredQuantity: 1},
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	if zones := deckZoneQuantities(t, db, deck.ID); zones[models.ZoneOther] != 3 || len(zones) != 1 {
		t.Errorf("persisted zones = %v, want only other: 3", zones)
	}
}

func TestDeckCreateItemsBatch_CardInOtherRejected(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Bolts")
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")

	resp := postDeckBatch(t, app, deck.ID, []CreateDeckItemRequest{
		{OracleID: "oracle-bolt", ScryfallID: "scry-bolt", Zone: models.ZoneOther, DesiredQuantity: 1},
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestDeckUpdateItem_ExtraZoneRules(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	token := seedPinnedDeckItem(t, db, deck.ID, "oracle-rabbit", "scry-rabbit", models.ZoneOther, 1)
	card := seedPinnedDeckItem(t, db, deck.ID, "oracle-bolt", "scry-bolt", models.ZoneMain, 1)

	main, other := models.ZoneMain, models.ZoneOther
	qty := 3
	cases := []struct {
		name string
		item models.DeckItem
		req  UpdateDeckItemRequest
		want int
	}{
		{"token cannot leave other", token, UpdateDeckItemRequest{Zone: &main}, http.StatusBadRequest},
		{"card cannot enter other", card, UpdateDeckItemRequest{Zone: &other}, http.StatusBadRequest},
		{"token quantity can change", token, UpdateDeckItemRequest{DesiredQuantity: &qty}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := putDeckItem(t, app, deck.ID, tc.item.ID, tc.req)
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != tc.want {
				t.Errorf("expected status %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}
}

func TestDeckMoveItem_ExtraZoneRules(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	token := seedPinnedDeckItem(t, db, deck.ID, "oracle-rabbit", "scry-rabbit", models.ZoneOther, 2)
	card := seedPinnedDeckItem(t, db, deck.ID, "oracle-bolt", "scry-bolt", models.ZoneMain, 2)

	for name, tc := range map[string]struct {
		item models.DeckItem
		zone models.DeckZone
	}{
		"token cannot leave other": {token, models.ZoneSide},
		"card cannot enter other":  {card, models.ZoneOther},
	} {
		t.Run(name, func(t *testing.T) {
			resp := postDeckMove(t, app, deck.ID, tc.item.ID, tc.zone, 1)
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
			}
		})
	}
}

// Tokens are items, not cards: they never add to total_cards_demand or the card
// shortfall, and get their own token signals on the deck page.
func TestDeckTokens_CountedSeparatelyFromCards(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deckA := createTestDeck(t, db, "Rabbits A")
	deckB := createTestDeck(t, db, "Rabbits B")

	// Cards: deck A wants 4 bolts and owns them all.
	createDeckTestCard(t, db, "scry-bolt", "oracle-bolt", "Lightning Bolt", "lea", "common")
	addDeckInventory(t, db, "oracle-bolt", "scry-bolt", 4)
	seedPinnedDeckItem(t, db, deckA.ID, "oracle-bolt", "scry-bolt", models.ZoneMain, 4)

	// Rabbit tokens: deck A wants 3 but only 1 is owned (2 missing).
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")
	addDeckInventory(t, db, "oracle-rabbit", "scry-rabbit", 1)
	seedPinnedDeckItem(t, db, deckA.ID, "oracle-rabbit", "scry-rabbit", models.ZoneOther, 3)

	// Food token: 1 owned, wanted once by each deck (over-committed across decks).
	createDeckTestToken(t, db, "scry-food", "oracle-food", "Food")
	addDeckInventory(t, db, "oracle-food", "scry-food", 1)
	seedPinnedDeckItem(t, db, deckA.ID, "oracle-food", "scry-food", models.ZoneOther, 1)
	seedPinnedDeckItem(t, db, deckB.ID, "oracle-food", "scry-food", models.ZoneOther, 1)

	summary := fetchDeckSummaries(t, app)[deckA.ID]
	if summary.TotalItems != 3 {
		t.Errorf("total_items = %d, want 3 (tokens count as items)", summary.TotalItems)
	}
	if summary.TotalCardsDemand != 4 {
		t.Errorf("total_cards_demand = %d, want 4 (tokens are not cards)", summary.TotalCardsDemand)
	}
	if summary.AggregateShortfall != 0 || summary.AggregateOverCommitted != 0 {
		t.Errorf("card signals = shortfall %d, over-committed %d; want 0, 0 (tokens excluded)",
			summary.AggregateShortfall, summary.AggregateOverCommitted)
	}

	items := fetchDeckItems(t, app, deckA.ID)
	if len(items.Main) != 1 || len(items.Other) != 2 {
		t.Fatalf("main = %d rows, other = %d rows; want 1 and 2", len(items.Main), len(items.Other))
	}
	if items.AggregateShortfall != 0 || items.AggregateOverCommitted != 0 {
		t.Errorf("card signals = shortfall %d, over-committed %d; want 0, 0",
			items.AggregateShortfall, items.AggregateOverCommitted)
	}
	if items.TokenShortfall != 2 {
		t.Errorf("token_shortfall = %d, want 2", items.TokenShortfall)
	}
	if items.TokenOverCommitted != 1 {
		t.Errorf("token_over_committed = %d, want 1", items.TokenOverCommitted)
	}
	for _, it := range items.Other {
		switch it.OracleID {
		case "oracle-rabbit":
			if !it.UnderOwned {
				t.Error("Rabbit row should be under_owned")
			}
		case "oracle-food":
			if !it.OverCommitted {
				t.Error("Food row should be over_committed")
			}
		}
	}
}

// Re-adding a token that is already in the deck merges into its other-zone row:
// the client asked for main and can't know the server moves it to other.
func TestDeckCreateItemsBatch_ReaddedTokenMergesIntoOther(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Rabbits")
	createDeckTestToken(t, db, "scry-rabbit", "oracle-rabbit", "Rabbit")
	seedPinnedDeckItem(t, db, deck.ID, "oracle-rabbit", "scry-rabbit", models.ZoneOther, 2)

	resp := postDeckBatch(t, app, deck.ID, []CreateDeckItemRequest{
		{OracleID: "oracle-rabbit", ScryfallID: "scry-rabbit", Zone: models.ZoneMain, DesiredQuantity: 1},
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var got []models.DeckItem
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 1 || got[0].Zone != models.ZoneOther || got[0].DesiredQuantity != 3 {
		t.Errorf("response items = %+v, want one other-zone item with quantity 3", got)
	}
	if zones := deckZoneQuantities(t, db, deck.ID); zones[models.ZoneOther] != 3 || len(zones) != 1 {
		t.Errorf("persisted zones = %v, want only other: 3", zones)
	}
}
