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
	// ZoneOther holds tokens and other non-playable extras (emblems, helper
	// cards, art cards, planes, ...). It is reserved for them: extras always live
	// here and real cards never do (see IsDeckExtra and CheckExtraZone).
	ZoneOther DeckZone = "other"
)

// IsValid reports whether the zone is one of the recognised values.
func (z DeckZone) IsValid() bool {
	switch z {
	case ZoneMain, ZoneSide, ZoneCommand, ZoneMaybe, ZoneOther:
		return true
	}
	return false
}

// CountsAsDemand reports whether entries in this zone lock inventory for the
// allocation service. Maybe-board entries do not — they are aspirational and
// must not affect the over-commitment math. Extras in the other zone do: the
// token copies a deck uses are tracked across decks like cards, just reported
// separately.
func (z DeckZone) CountsAsDemand() bool {
	return z == ZoneMain || z == ZoneSide || z == ZoneCommand || z == ZoneOther
}

// CountsAsCards reports whether entries in this zone count toward a deck's
// cards (its card total and card shortfall). Extras in the other zone are
// items, not cards.
func (z DeckZone) CountsAsCards() bool {
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
		return errors.New("zone must be one of main, side, command, maybe, other")
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

// MoveDeckItemTx moves qty copies of source into zone within tx. If the zone
// already holds the same card (oracle, pinned printing and finish), the copies
// merge into that row rather than colliding on the unique index; otherwise a
// row is created, or source itself is moved when every copy goes. The source
// row is reduced, and deleted when emptied. Returns the resulting target row.
//
// The caller ensures source.Zone != zone and 1 <= qty <= source.DesiredQuantity.
func MoveDeckItemTx(tx *gorm.DB, source DeckItem, zone DeckZone, qty int) (DeckItem, error) {
	var target DeckItem
	err := tx.Where("deck_id = ? AND oracle_id = ? AND scryfall_id = ? AND treatment = ? AND zone = ?",
		source.DeckID, source.OracleID, source.ScryfallID, source.Treatment, zone).First(&target).Error
	switch {
	case err == nil:
		// Same card already in the target zone: merge into it.
		target.DesiredQuantity += qty
		if err := tx.Save(&target).Error; err != nil {
			return DeckItem{}, err
		}
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return DeckItem{}, err
	case qty == source.DesiredQuantity:
		// Moving every copy and nothing to merge with: move the row itself.
		source.Zone = zone
		if err := tx.Save(&source).Error; err != nil {
			return DeckItem{}, err
		}
		return source, nil
	default:
		target = DeckItem{
			DeckID:          source.DeckID,
			OracleID:        source.OracleID,
			ScryfallID:      source.ScryfallID,
			Treatment:       source.Treatment,
			Zone:            zone,
			DesiredQuantity: qty,
		}
		if err := tx.Create(&target).Error; err != nil {
			return DeckItem{}, err
		}
	}

	source.DesiredQuantity -= qty
	if source.DesiredQuantity == 0 {
		return target, tx.Delete(&source).Error
	}
	return target, tx.Save(&source).Error
}
