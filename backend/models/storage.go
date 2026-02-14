package models

import (
	"errors"

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
