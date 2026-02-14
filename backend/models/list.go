package models

import (
	"errors"

	"gorm.io/gorm"
)

// List represents a collection of cards the user wants to acquire
// tygo:export
type List struct {
	BaseModel
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// Relationship - items in this list
	Items []ListItem `gorm:"foreignKey:ListID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"items,omitempty"`
}

func (l *List) ValidateList(tx *gorm.DB) error {
	if l.Name == "" {
		return errors.New("name cannot be empty")
	}
	return nil
}

// BeforeCreate validates the list before creating a record
func (l *List) BeforeCreate(tx *gorm.DB) error {
	return l.ValidateList(tx)
}

// BeforeUpdate validates the list before updating a record
func (l *List) BeforeUpdate(tx *gorm.DB) error {
	return l.ValidateList(tx)
}
