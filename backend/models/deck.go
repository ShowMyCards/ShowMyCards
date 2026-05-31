package models

import (
	"errors"

	"gorm.io/gorm"
)

// Deck represents a user-defined deck of cards.
//
// A deck is a declarative want-list: it does not hold cards directly. Owned
// copies live in Inventory; "decked" status is derived by the allocation
// service from the deck's items.
//
// tygo:export
type Deck struct {
	BaseModel
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// Relationship - items in this deck
	Items []DeckItem `gorm:"foreignKey:DeckID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"items,omitempty"`
}

func (d *Deck) ValidateDeck(tx *gorm.DB) error {
	if d.Name == "" {
		return errors.New("name cannot be empty")
	}
	return nil
}

// BeforeCreate validates the deck before creating a record
func (d *Deck) BeforeCreate(tx *gorm.DB) error {
	return d.ValidateDeck(tx)
}

// BeforeUpdate validates the deck before updating a record
func (d *Deck) BeforeUpdate(tx *gorm.DB) error {
	return d.ValidateDeck(tx)
}
