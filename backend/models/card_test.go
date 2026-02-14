package models

import (
	"encoding/json"
	"strings"
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestCard_ValidateCard(t *testing.T) {
	db := setupCardTestDB(t)

	tests := []struct {
		name        string
		card        *Card
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Card",
			card: &Card{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				RawJSON:    `{"name": "Test Card"}`,
			},
			expectError: false,
		},
		{
			name: "Valid Card - Empty OracleID",
			card: &Card{
				ScryfallID: "test-id",
				OracleID:   "",
				RawJSON:    `{"name": "Test Token"}`,
			},
			expectError: false,
		},
		{
			name: "Invalid - Empty ScryfallID",
			card: &Card{
				ScryfallID: "",
				OracleID:   "oracle-id",
				RawJSON:    `{"name": "Test"}`,
			},
			expectError: true,
			errorMsg:    "scryfall_id cannot be empty",
		},
		{
			name: "Invalid - Empty RawJSON",
			card: &Card{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				RawJSON:    "",
			},
			expectError: true,
			errorMsg:    "raw_json cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.card.ValidateCard(db)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCard_BeforeCreate(t *testing.T) {
	db := setupCardTestDB(t)

	tests := []struct {
		name        string
		card        *Card
		expectError bool
	}{
		{
			name: "Valid Create",
			card: &Card{
				ScryfallID: "test-id-1",
				OracleID:   "oracle-id",
				RawJSON:    `{"name": "Test Card"}`,
			},
			expectError: false,
		},
		{
			name: "Invalid Create - Empty ScryfallID",
			card: &Card{
				ScryfallID: "",
				OracleID:   "oracle-id",
				RawJSON:    `{"name": "Test"}`,
			},
			expectError: true,
		},
		{
			name: "Invalid Create - Empty RawJSON",
			card: &Card{
				ScryfallID: "test-id-2",
				OracleID:   "oracle-id",
				RawJSON:    "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.card).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCard_BeforeUpdate(t *testing.T) {
	db := setupCardTestDB(t)

	// Create initial card
	card := &Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    `{"name": "Original"}`,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	tests := []struct {
		name        string
		updateFunc  func(*Card)
		expectError bool
	}{
		{
			name: "Valid Update - RawJSON",
			updateFunc: func(c *Card) {
				c.RawJSON = `{"name": "Updated"}`
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Empty RawJSON",
			updateFunc: func(c *Card) {
				c.RawJSON = ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reload card
			var testCard Card
			if err := db.First(&testCard, "scryfall_id = ?", card.ScryfallID).Error; err != nil {
				t.Fatalf("failed to load card: %v", err)
			}

			// Apply update
			tt.updateFunc(&testCard)

			// Try to save
			err := db.Save(&testCard).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCard_ToScryfallCard(t *testing.T) {
	tests := []struct {
		name        string
		rawJSON     string
		expectError bool
		validateFn  func(*testing.T, scryfall.Card)
	}{
		{
			name:        "Valid JSON",
			rawJSON:     `{"id": "test-id", "name": "Lightning Bolt", "set": "lea"}`,
			expectError: false,
			validateFn: func(t *testing.T, card scryfall.Card) {
				if card.ID != "test-id" {
					t.Errorf("expected ID 'test-id', got '%s'", card.ID)
				}
				if card.Name != "Lightning Bolt" {
					t.Errorf("expected name 'Lightning Bolt', got '%s'", card.Name)
				}
				if card.Set != "lea" {
					t.Errorf("expected set 'lea', got '%s'", card.Set)
				}
			},
		},
		{
			name:        "JSON with Zero Time - Should Clean via cleanRawJSON",
			rawJSON:     cleanRawJSON(`{"id": "test-id", "name": "Test", "released_at": "0001-01-01T00:00:00Z"}`),
			expectError: false,
			validateFn: func(t *testing.T, card scryfall.Card) {
				if card.Name != "Test" {
					t.Errorf("expected name 'Test', got '%s'", card.Name)
				}
			},
		},
		{
			name:        "Invalid JSON",
			rawJSON:     `{invalid json}`,
			expectError: true,
			validateFn:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &Card{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				RawJSON:    tt.rawJSON,
			}

			scryfallCard, err := card.ToScryfallCard()
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if tt.validateFn != nil {
					tt.validateFn(t, scryfallCard)
				}
			}
		})
	}
}

func TestCard_ToScryfallCard_ZeroTimeReplacement(t *testing.T) {
	// Date cleanup now happens at import time via cleanRawJSON/FromScryfallCard
	card := &Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    cleanRawJSON(`{"name": "Test", "released_at": "0001-01-01T00:00:00Z"}`),
	}

	scryfallCard, err := card.ToScryfallCard()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if scryfallCard.Name != "Test" {
		t.Errorf("expected name 'Test', got '%s'", scryfallCard.Name)
	}
}

func TestCard_FromScryfallCard(t *testing.T) {
	scryfallCard := scryfall.Card{
		ID:       "test-id",
		OracleID: "oracle-id",
		Name:     "Lightning Bolt",
		Set:      "lea",
	}

	card, err := FromScryfallCard(scryfallCard)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify fields
	if card.ScryfallID != "test-id" {
		t.Errorf("expected ScryfallID 'test-id', got '%s'", card.ScryfallID)
	}
	if card.OracleID != "oracle-id" {
		t.Errorf("expected OracleID 'oracle-id', got '%s'", card.OracleID)
	}

	// Verify RawJSON is valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(card.RawJSON), &jsonData); err != nil {
		t.Errorf("expected valid JSON, got error: %v", err)
	}

	// Verify JSON contains expected fields
	if name, ok := jsonData["name"].(string); !ok || name != "Lightning Bolt" {
		t.Errorf("expected name 'Lightning Bolt' in JSON, got %v", jsonData["name"])
	}
	if set, ok := jsonData["set"].(string); !ok || set != "lea" {
		t.Errorf("expected set 'lea' in JSON, got %v", jsonData["set"])
	}
}

func TestCard_FromScryfallCard_RoundTrip(t *testing.T) {
	// Create a Scryfall card
	original := scryfall.Card{
		ID:       "round-trip-id",
		OracleID: "round-trip-oracle",
		Name:     "Test Card",
		Set:      "tst",
	}

	// Convert to Card
	card, err := FromScryfallCard(original)
	if err != nil {
		t.Fatalf("FromScryfallCard failed: %v", err)
	}

	// Convert back to Scryfall card
	result, err := card.ToScryfallCard()
	if err != nil {
		t.Fatalf("ToScryfallCard failed: %v", err)
	}

	// Verify round-trip preserved data
	if result.ID != original.ID {
		t.Errorf("expected ID '%s', got '%s'", original.ID, result.ID)
	}
	if result.OracleID != original.OracleID {
		t.Errorf("expected OracleID '%s', got '%s'", original.OracleID, result.OracleID)
	}
	if result.Name != original.Name {
		t.Errorf("expected Name '%s', got '%s'", original.Name, result.Name)
	}
	if result.Set != original.Set {
		t.Errorf("expected Set '%s', got '%s'", original.Set, result.Set)
	}
}

func TestCard_GetCardsByIDs(t *testing.T) {
	db := setupCardTestDB(t)

	// Create test cards
	cards := []*Card{
		{ScryfallID: "id-1", OracleID: "oracle-1", RawJSON: `{"name": "Card 1"}`},
		{ScryfallID: "id-2", OracleID: "oracle-2", RawJSON: `{"name": "Card 2"}`},
		{ScryfallID: "id-3", OracleID: "oracle-3", RawJSON: `{"name": "Card 3"}`},
	}

	for _, card := range cards {
		if err := db.Create(card).Error; err != nil {
			t.Fatalf("failed to create card: %v", err)
		}
	}

	tests := []struct {
		name         string
		ids          []string
		expectedLen  int
		expectError  bool
		validateFunc func(*testing.T, map[string]Card)
	}{
		{
			name:        "Get All Cards",
			ids:         []string{"id-1", "id-2", "id-3"},
			expectedLen: 3,
			expectError: false,
			validateFunc: func(t *testing.T, cardMap map[string]Card) {
				if _, ok := cardMap["id-1"]; !ok {
					t.Error("expected id-1 in map")
				}
				if _, ok := cardMap["id-2"]; !ok {
					t.Error("expected id-2 in map")
				}
				if _, ok := cardMap["id-3"]; !ok {
					t.Error("expected id-3 in map")
				}
			},
		},
		{
			name:        "Get Subset",
			ids:         []string{"id-1", "id-3"},
			expectedLen: 2,
			expectError: false,
			validateFunc: func(t *testing.T, cardMap map[string]Card) {
				if _, ok := cardMap["id-1"]; !ok {
					t.Error("expected id-1 in map")
				}
				if _, ok := cardMap["id-3"]; !ok {
					t.Error("expected id-3 in map")
				}
				if _, ok := cardMap["id-2"]; ok {
					t.Error("did not expect id-2 in map")
				}
			},
		},
		{
			name:        "Non-existent ID",
			ids:         []string{"non-existent"},
			expectedLen: 0,
			expectError: false,
			validateFunc: func(t *testing.T, cardMap map[string]Card) {
				if len(cardMap) != 0 {
					t.Errorf("expected empty map, got %d cards", len(cardMap))
				}
			},
		},
		{
			name:        "Mixed Existent and Non-existent",
			ids:         []string{"id-1", "non-existent", "id-2"},
			expectedLen: 2,
			expectError: false,
			validateFunc: func(t *testing.T, cardMap map[string]Card) {
				if len(cardMap) != 2 {
					t.Errorf("expected 2 cards, got %d", len(cardMap))
				}
			},
		},
		{
			name:        "Empty ID List",
			ids:         []string{},
			expectedLen: 0,
			expectError: false,
			validateFunc: func(t *testing.T, cardMap map[string]Card) {
				if len(cardMap) != 0 {
					t.Errorf("expected empty map, got %d cards", len(cardMap))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cardMap, err := GetCardsByIDs(db, tt.ids)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if len(cardMap) != tt.expectedLen {
					t.Errorf("expected %d cards, got %d", tt.expectedLen, len(cardMap))
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, cardMap)
				}
			}
		})
	}
}

// LOW-VALUE: Tests a one-liner that returns a constant string. Verifies Go works, not business logic.
func TestCard_TableName(t *testing.T) {
	card := Card{}
	tableName := card.TableName()
	if tableName != "cards" {
		t.Errorf("expected table name 'cards', got '%s'", tableName)
	}
}

func TestCard_ZeroTimeHandling(t *testing.T) {
	// Date cleanup now happens at import time via cleanRawJSON
	rawJSON := `{
		"id": "test-id",
		"name": "Test Card",
		"released_at": "0001-01-01T00:00:00Z",
		"other_date": "2020-01-01T00:00:00Z"
	}`

	cleaned := cleanRawJSON(rawJSON)
	card := &Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    cleaned,
	}

	scryfallCard, err := card.ToScryfallCard()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if scryfallCard.Name != "Test Card" {
		t.Errorf("expected name 'Test Card', got '%s'", scryfallCard.Name)
	}

	// Verify zero time was replaced in cleaned JSON
	if strings.Contains(cleaned, "0001-01-01T00:00:00Z") {
		t.Error("cleaned JSON should not contain zero time")
	}
}

func TestCard_MalformedDateHandling(t *testing.T) {
	// Date cleanup now happens at import time via cleanRawJSON
	rawJSON := `{
		"id": "be010c2f-06db-47e3-80bd-df3f2a21ca34",
		"oracle_id": "73864fcc-1bde-4bc0-831e-2b93e546e417",
		"name": "Godless Shrine",
		"released_at": "2006-02-03T00:00:00-08:00",
		"set": "gpt",
		"set_name": "Guildpact",
		"collector_number": "157"
	}`

	card := &Card{
		ScryfallID: "be010c2f-06db-47e3-80bd-df3f2a21ca34",
		OracleID:   "73864fcc-1bde-4bc0-831e-2b93e546e417",
		RawJSON:    cleanRawJSON(rawJSON),
	}

	scryfallCard, err := card.ToScryfallCard()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if scryfallCard.Name != "Godless Shrine" {
		t.Errorf("expected name 'Godless Shrine', got '%s'", scryfallCard.Name)
	}

	if scryfallCard.ID != "be010c2f-06db-47e3-80bd-df3f2a21ca34" {
		t.Errorf("expected ID 'be010c2f-06db-47e3-80bd-df3f2a21ca34', got '%s'", scryfallCard.ID)
	}

	if scryfallCard.Set != "gpt" {
		t.Errorf("expected set 'gpt', got '%s'", scryfallCard.Set)
	}
}

func TestCard_MalformedPreviewedAtHandling(t *testing.T) {
	// Date cleanup now happens at import time via cleanRawJSON
	rawJSON := `{
		"id": "test-id",
		"name": "Test Card",
		"previewed_at": "2020-01-15T00:00:00-08:00",
		"released_at": "2020-02-01",
		"set": "test"
	}`

	card := &Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    cleanRawJSON(rawJSON),
	}

	scryfallCard, err := card.ToScryfallCard()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if scryfallCard.Name != "Test Card" {
		t.Errorf("expected name 'Test Card', got '%s'", scryfallCard.Name)
	}
}
