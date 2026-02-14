package models

import (
	"encoding/json"
	"errors"
	"strings"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
)

// cleanRawJSON normalizes date fields in card JSON for the go-scryfall Date type.
// Replaces zero-time values with null and truncates timestamp date fields to date-only format.
func cleanRawJSON(rawJSON string) string {
	cleaned := strings.ReplaceAll(rawJSON, `"0001-01-01T00:00:00Z"`, `null`)

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &jsonData); err != nil {
		return cleaned
	}

	changed := false
	if releasedAt, ok := jsonData["released_at"].(string); ok && len(releasedAt) > 10 {
		jsonData["released_at"] = releasedAt[:10]
		changed = true
	}
	if preview, ok := jsonData["preview"].(map[string]interface{}); ok {
		if previewedAt, ok := preview["previewed_at"].(string); ok && len(previewedAt) > 10 {
			preview["previewed_at"] = previewedAt[:10]
			changed = true
		}
	}

	if !changed && !strings.Contains(rawJSON, `"0001-01-01T00:00:00Z"`) {
		return rawJSON
	}

	cleanedBytes, err := json.Marshal(jsonData)
	if err != nil {
		return cleaned
	}
	return string(cleanedBytes)
}

// Card represents a Magic card from Scryfall's bulk data
// Stores the complete card data as JSON to avoid duplication
// tygo:export
type Card struct {
	ScryfallID string `gorm:"primaryKey;type:varchar(255);not null" json:"scryfall_id"`
	OracleID   string `gorm:"index;type:varchar(255)" json:"oracle_id"` // Can be empty for tokens/emblems
	RawJSON    string `gorm:"type:text;not null" json:"-"`              // Don't expose in API

	// Generated columns (created via migration, not by GORM)
	// These are read-only and populated by SQLite from RawJSON
	// Use "-" tag to exclude from AutoMigrate entirely
	Name    string `gorm:"-" json:"name"`
	SetCode string `gorm:"-" json:"set_code"`
}

// TableName specifies the table name for the Card model
func (Card) TableName() string {
	return "cards"
}

// Validate a card is a valid record:
func (c *Card) ValidateCard(tx *gorm.DB) error {
	if c.ScryfallID == "" {
		return errors.New("scryfall_id cannot be empty")
	}
	// OracleID can be empty for some card types (tokens, emblems, etc.)
	if c.RawJSON == "" {
		return errors.New("raw_json cannot be empty")
	}
	return nil
}

// BeforeCreate validates the card before creation
func (c *Card) BeforeCreate(tx *gorm.DB) error {
	return c.ValidateCard(tx)
}

// BeforeUpdate validates the card before update
func (c *Card) BeforeUpdate(tx *gorm.DB) error {
	return c.ValidateCard(tx)
}

// ToScryfallCard unmarshals the RawJSON into a scryfall.Card struct.
// Applies cleanRawJSON to handle cards imported before date normalization was added.
func (c *Card) ToScryfallCard() (scryfall.Card, error) {
	var card scryfall.Card
	if err := json.Unmarshal([]byte(cleanRawJSON(c.RawJSON)), &card); err != nil {
		return scryfall.Card{}, err
	}
	return card, nil
}

// FromScryfallCard creates a Card from a scryfall.Card.
// Date fields are cleaned at import time so ToScryfallCard can do a single unmarshal.
func FromScryfallCard(scryfallCard scryfall.Card) (*Card, error) {
	rawJSON, err := json.Marshal(scryfallCard)
	if err != nil {
		return nil, err
	}

	return &Card{
		ScryfallID: scryfallCard.ID,
		OracleID:   scryfallCard.OracleID,
		RawJSON:    cleanRawJSON(string(rawJSON)),
	}, nil
}

// GetCardsByIDs fetches multiple cards by their Scryfall IDs and returns them as a map
func GetCardsByIDs(db *gorm.DB, scryfallIDs []string) (map[string]Card, error) {
	if len(scryfallIDs) == 0 {
		return make(map[string]Card), nil
	}

	var cards []Card
	if err := db.Where("scryfall_id IN ?", scryfallIDs).Find(&cards).Error; err != nil {
		return nil, err
	}

	cardMap := make(map[string]Card, len(cards))
	for _, card := range cards {
		cardMap[card.ScryfallID] = card
	}
	return cardMap, nil
}
