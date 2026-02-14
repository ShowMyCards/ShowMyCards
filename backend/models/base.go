// Package models defines the domain models for ShowMyCards.
// These models represent the database schema and are the single source of truth for data structures.
package models

import (
	"time"
)

// BaseModel represents the base model with common fields
// tygo:export
type BaseModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
