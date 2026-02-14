package models

import (
	"errors"

	"gorm.io/gorm"
)

// tygo:export
type Setting struct {
	BaseModel
	Key   string `gorm:"type:varchar(255);uniqueIndex;not null" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

func (s *Setting) ValidateSetting(tx *gorm.DB) error {
	if s.Key == "" {
		return errors.New("key cannot be empty")
	}
	return nil
}

// BeforeCreate validates the setting before creation
func (s *Setting) BeforeCreate(tx *gorm.DB) error {
	return s.ValidateSetting(tx)
}

// BeforeUpdate validates the setting before update
func (s *Setting) BeforeUpdate(tx *gorm.DB) error {
	return s.ValidateSetting(tx)
}
