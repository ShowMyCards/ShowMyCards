package models

import (
	"errors"

	"gorm.io/gorm"
)

// ListItem represents a single card entry in a list
// tygo:export
type ListItem struct {
	BaseModel
	ListID            uint   `gorm:"not null;index;uniqueIndex:idx_list_card_treatment" json:"list_id"`
	ScryfallID        string `gorm:"type:varchar(255);not null;uniqueIndex:idx_list_card_treatment" json:"scryfall_id"`
	OracleID          string `gorm:"type:varchar(255);not null;index" json:"oracle_id"`
	Treatment         string `gorm:"type:varchar(100);uniqueIndex:idx_list_card_treatment" json:"treatment"`
	DesiredQuantity   int    `gorm:"not null;default:1" json:"desired_quantity"`
	CollectedQuantity int    `gorm:"not null;default:0" json:"collected_quantity"`

	// Relationship
	List *List `gorm:"foreignKey:ListID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"list,omitempty"`
}

func (li *ListItem) ValidateListItem(tx *gorm.DB) error {
	if li.ScryfallID == "" {
		return errors.New("scryfall_id cannot be empty")
	}
	if li.OracleID == "" {
		return errors.New("oracle_id cannot be empty")
	}
	if li.DesiredQuantity < 1 {
		return errors.New("desired_quantity must be at least 1")
	}
	if li.CollectedQuantity < 0 {
		return errors.New("collected_quantity cannot be negative")
	}
	if li.CollectedQuantity > li.DesiredQuantity {
		return errors.New("collected_quantity cannot exceed desired_quantity")
	}
	return nil
}

// BeforeCreate validates the list item before creating a record
func (li *ListItem) BeforeCreate(tx *gorm.DB) error {
	return li.ValidateListItem(tx)
}

// BeforeUpdate validates the list item before updating a record
func (li *ListItem) BeforeUpdate(tx *gorm.DB) error {
	return li.ValidateListItem(tx)
}
