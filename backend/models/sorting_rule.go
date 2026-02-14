package models

import (
	"errors"

	"gorm.io/gorm"
)

// SortingRule represents a rule for automatically sorting cards into storage locations
// tygo:export
type SortingRule struct {
	BaseModel
	Name              string `gorm:"type:varchar(255);not null" json:"name"`
	Priority          int    `gorm:"not null;index" json:"priority"`
	Expression        string `gorm:"type:text;not null" json:"expression"`
	StorageLocationID uint   `gorm:"not null;index" json:"storage_location_id"`
	Enabled           bool   `gorm:"default:true;not null" json:"enabled"`

	// Relationship
	StorageLocation StorageLocation `gorm:"foreignKey:StorageLocationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"storage_location,omitempty"`
}

func (r *SortingRule) ValidateSortingRule(tx *gorm.DB) error {
	if r.Name == "" {
		return errors.New("rule name cannot be empty")
	}
	if r.Expression == "" {
		return errors.New("rule expression cannot be empty")
	}
	if r.StorageLocationID == 0 {
		return errors.New("storage location ID must be set")
	}
	return nil
}

// BeforeCreate validates the rule before creating a record
func (r *SortingRule) BeforeCreate(tx *gorm.DB) error {
	return r.ValidateSortingRule(tx)
}

// BeforeUpdate validates the rule before updating a record
func (r *SortingRule) BeforeUpdate(tx *gorm.DB) error {
	return r.ValidateSortingRule(tx)
}
