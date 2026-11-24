package models

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// StorageType represents the type of storage location
// tygo:export
type StorageType string

const (
	Box    StorageType = "Box"
	Binder StorageType = "Binder"
)

// IsValid checks if the storage type is valid
func (s StorageType) IsValid() bool {
	switch s {
	case Box, Binder:
		return true
	default:
		return false
	}
}

// tygo:export
type StorageLocation struct {
	BaseModel
	Name        string      `gorm:"type:varchar(255);not null" json:"name"`
	StorageType StorageType `gorm:"type:varchar(50);not null;check:storage_type IN ('Box', 'Binder')" json:"storage_type"`
	SortOrder   int         `gorm:"not null;default:0;index" json:"sort_order"`
}

func (s *StorageLocation) ValidateStorageLocation(tx *gorm.DB) error {
	if s.Name == "" {
		return errors.New("name cannot be empty")
	}
	if !s.StorageType.IsValid() {
		return errors.New("invalid storage type")
	}
	return nil
}

// BeforeCreate validates the storage location before creating a record
func (s *StorageLocation) BeforeCreate(tx *gorm.DB) error {
	return s.ValidateStorageLocation(tx)
}

// BeforeUpdate validates the storage location before updating a record
func (s *StorageLocation) BeforeUpdate(tx *gorm.DB) error {
	return s.ValidateStorageLocation(tx)
}

// OrderStorageLocations orders a storage_locations query by manual sort_order,
// tie-breaking by ID ascending.
func OrderStorageLocations(db *gorm.DB) *gorm.DB {
	return db.Order("sort_order ASC, id ASC")
}

// MaxStorageSortOrder returns the highest sort_order across storage_locations,
// or 0 when no storage locations exist.
func MaxStorageSortOrder(db *gorm.DB) (int, error) {
	var maxSortOrder int
	if err := db.Model(&StorageLocation{}).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxSortOrder).Error; err != nil {
		return 0, fmt.Errorf("querying max storage sort_order: %w", err)
	}
	return maxSortOrder, nil
}
