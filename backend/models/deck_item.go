package models

import (
	"errors"

	"gorm.io/gorm"
)

// DeckZone is where a card sits within a deck.
//
// tygo:export
type DeckZone string

const (
	ZoneMain    DeckZone = "main"
	ZoneSide    DeckZone = "side"
	ZoneCommand DeckZone = "command"
	ZoneMaybe   DeckZone = "maybe"
)

// IsValid reports whether the zone is one of the recognised values.
func (z DeckZone) IsValid() bool {
	switch z {
	case ZoneMain, ZoneSide, ZoneCommand, ZoneMaybe:
		return true
	}
	return false
}

// CountsAsDemand reports whether cards in this zone lock inventory for the
// allocation service. Maybe-board entries do not — they are aspirational and
// must not affect the over-commitment math.
func (z DeckZone) CountsAsDemand() bool {
	return z == ZoneMain || z == ZoneSide || z == ZoneCommand
}

// DeckItem represents a single card entry in a deck.
//
// ScryfallID is empty when the user does not pin a specific printing (any
// printing of the Oracle satisfies the entry); Treatment is empty when the
// user does not pin a specific finish. Both are stored as empty strings rather
// than NULL: SQLite treats NULLs as distinct in a unique index, which would
// allow duplicate "any printing" rows past the composite uniqueness check.
//
// Treatment is carried through but is NOT factored into the global
// over-commitment math in Milestone 1 (see FR98/IMPLEMENTATION_PLAN.md §2).
//
// tygo:export
type DeckItem struct {
	BaseModel
	DeckID          uint     `gorm:"not null;index;uniqueIndex:idx_deck_card" json:"deck_id"`
	OracleID        string   `gorm:"type:varchar(255);not null;index;uniqueIndex:idx_deck_card" json:"oracle_id"`
	ScryfallID      string   `gorm:"type:varchar(255);uniqueIndex:idx_deck_card" json:"scryfall_id"`
	Treatment       string   `gorm:"type:varchar(100);uniqueIndex:idx_deck_card" json:"treatment"`
	Zone            DeckZone `gorm:"type:varchar(50);not null;uniqueIndex:idx_deck_card" json:"zone"`
	DesiredQuantity int      `gorm:"not null;default:1" json:"desired_quantity"`

	// Relationship
	Deck *Deck `gorm:"foreignKey:DeckID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"deck,omitempty"`
}

func (di *DeckItem) ValidateDeckItem(tx *gorm.DB) error {
	if di.OracleID == "" {
		return errors.New("oracle_id cannot be empty")
	}
	if di.DesiredQuantity < 1 {
		return errors.New("desired_quantity must be at least 1")
	}
	if !di.Zone.IsValid() {
		return errors.New("zone must be one of main, side, command, maybe")
	}
	return nil
}

// BeforeCreate validates the deck item before creating a record
func (di *DeckItem) BeforeCreate(tx *gorm.DB) error {
	return di.ValidateDeckItem(tx)
}

// BeforeUpdate validates the deck item before updating a record
func (di *DeckItem) BeforeUpdate(tx *gorm.DB) error {
	return di.ValidateDeckItem(tx)
}
