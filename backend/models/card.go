package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
		return nil, fmt.Errorf("fetching cards by IDs: %w", err)
	}

	cardMap := make(map[string]Card, len(cards))
	for _, card := range cards {
		cardMap[card.ScryfallID] = card
	}
	return cardMap, nil
}

// PrintKey identifies a printing by its set code and collector number, which a
// card shares across all of its language variants.
type PrintKey struct {
	SetCode         string
	CollectorNumber string
}

// String returns the map key form "set|collector_number".
func (k PrintKey) String() string {
	return k.SetCode + "|" + k.CollectorNumber
}

// GetEnglishPricesByPrint returns English-printing prices keyed by
// "set|collector_number" for the requested prints. Scryfall populates prices
// only on the English printing of each set + collector number; non-English
// printings carry empty prices, so callers use this to back-fill them.
//
// Prices are read with json_extract (COALESCE'd to "") to avoid unmarshaling
// full card JSON. The query is backed by idx_cards_print_lookup.
func GetEnglishPricesByPrint(db *gorm.DB, keys []PrintKey) (map[string]scryfall.Prices, error) {
	result := make(map[string]scryfall.Prices)
	if len(keys) == 0 {
		return result, nil
	}

	// Dedupe input pairs for the IN clause.
	seen := make(map[string]bool, len(keys))
	pairs := make([][]any, 0, len(keys))
	for _, k := range keys {
		if seen[k.String()] {
			continue
		}
		seen[k.String()] = true
		pairs = append(pairs, []any{k.SetCode, k.CollectorNumber})
	}

	type priceRow struct {
		SetCode         string
		CollectorNumber string
		USD             string
		USDFoil         string
		USDEtched       string
		EUR             string
		EURFoil         string
		Tix             string
	}

	var rows []priceRow
	if err := db.Table("cards").
		Select(`set_code, collector_number,
			COALESCE(json_extract(raw_json, '$.prices.usd'), '') AS usd,
			COALESCE(json_extract(raw_json, '$.prices.usd_foil'), '') AS usd_foil,
			COALESCE(json_extract(raw_json, '$.prices.usd_etched'), '') AS usd_etched,
			COALESCE(json_extract(raw_json, '$.prices.eur'), '') AS eur,
			COALESCE(json_extract(raw_json, '$.prices.eur_foil'), '') AS eur_foil,
			COALESCE(json_extract(raw_json, '$.prices.tix'), '') AS tix`).
		Where("lang = ? AND (set_code, collector_number) IN ?", "en", pairs).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("fetching english prices by print: %w", err)
	}

	for _, r := range rows {
		key := PrintKey{SetCode: r.SetCode, CollectorNumber: r.CollectorNumber}
		result[key.String()] = scryfall.Prices{
			USD:       r.USD,
			USDFoil:   r.USDFoil,
			USDEtched: r.USDEtched,
			EUR:       r.EUR,
			EURFoil:   r.EURFoil,
			Tix:       r.Tix,
		}
	}
	return result, nil
}

// GetScryfallCardsByIDs fetches cards by their Scryfall IDs, unmarshals them,
// and returns a map of Scryfall ID to parsed scryfall.Card.
// Cards that fail to unmarshal are logged and skipped.
func GetScryfallCardsByIDs(db *gorm.DB, scryfallIDs []string) (map[string]scryfall.Card, error) {
	cardMap, err := GetCardsByIDs(db, scryfallIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[string]scryfall.Card, len(cardMap))
	for id, card := range cardMap {
		scryfallCard, err := card.ToScryfallCard()
		if err != nil {
			slog.Warn("failed to unmarshal card", "scryfall_id", id, "error", err)
			continue
		}
		result[id] = scryfallCard
	}
	return result, nil
}
