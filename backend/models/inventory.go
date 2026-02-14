package models

import (
	"errors"

	"gorm.io/gorm"
)

// Inventory represents a card in the collection
// tygo:export
type Inventory struct {
	BaseModel
	ScryfallID        string `gorm:"type:varchar(255);not null;index" json:"scryfall_id"`
	OracleID          string `gorm:"type:varchar(255);not null;index;index:idx_oracle_storage" json:"oracle_id"`
	Treatment         string `gorm:"type:varchar(100)" json:"treatment"`
	Quantity          int    `gorm:"not null;default:1" json:"quantity"`
	StorageLocationID *uint  `gorm:"index;index:idx_oracle_storage" json:"storage_location_id,omitempty"`

	// Relationship
	StorageLocation *StorageLocation `gorm:"foreignKey:StorageLocationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"storage_location,omitempty"`
}

func (i *Inventory) ValidateInventory(tx *gorm.DB) error {
	if i.ScryfallID == "" {
		return errors.New("scryfall_id cannot be empty")
	}
	if i.OracleID == "" {
		return errors.New("oracle_id cannot be empty")
	}
	if i.Quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return nil
}

// BeforeCreate validates the inventory before creating a record
func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	return i.ValidateInventory(tx)
}

// BeforeUpdate validates the inventory before updating a record
func (i *Inventory) BeforeUpdate(tx *gorm.DB) error {
	return i.ValidateInventory(tx)
}
