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

	if err := db.AutoMigrate(&models.Deck{}, &models.DeckItem{}); err != nil {
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

	return app, db
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

// Items stub tests (real implementation lands in 1b)

func TestDeckListItems_StubReturnsEmptyGroupedResponse(t *testing.T) {
	app, db := setupDeckTestApp(t)
	deck := createTestDeck(t, db, "Stubbed")

	// Stash an item so we can prove the stub does NOT yet return items.
	if err := db.Create(&models.DeckItem{
		DeckID: deck.ID, OracleID: "oracle-1", Zone: models.ZoneMain, DesiredQuantity: 1,
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
	if len(got.Main) != 0 || len(got.Side) != 0 || len(got.Command) != 0 || len(got.Maybe) != 0 {
		t.Errorf("expected all zone arrays empty in 1a stub, got main=%d side=%d command=%d maybe=%d",
			len(got.Main), len(got.Side), len(got.Command), len(got.Maybe))
	}
	if got.AggregateShortfall != 0 {
		t.Errorf("expected aggregate_shortfall=0 in stub, got %d", got.AggregateShortfall)
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
