package models

import "strings"

// Symbol represents a Magic card symbol (mana, tap, etc.) sourced from
// Scryfall's symbology endpoint. The cached SVG content is stored directly
// in SQLite so it can be served without re-fetching from Scryfall.
//
// tygo:export
type Symbol struct {
	// Code is the normalized symbol key with surrounding braces stripped and
	// uppercased (e.g. "T", "W", "2"). It is used for API lookups.
	Code string `gorm:"primaryKey;type:varchar(20);not null" json:"code"`
	// Symbol is the original Scryfall representation including braces (e.g. "{T}").
	Symbol string `gorm:"type:varchar(20);not null" json:"symbol"`
	// English is the human-readable description of the symbol.
	English string `gorm:"type:varchar(255)" json:"english"`
	// SVG is the raw SVG content for the symbol.
	SVG string `gorm:"type:text" json:"-"`
}

func (Symbol) TableName() string {
	return "symbols"
}

// NormalizeSymbolCode strips surrounding braces and whitespace from a symbol
// string and uppercases it so that lookups and storage use a consistent key.
// For example "{T}", "t", and " T " all normalize to "T".
func NormalizeSymbolCode(symbol string) string {
	code := strings.TrimSpace(symbol)
	code = strings.TrimPrefix(code, "{")
	code = strings.TrimSuffix(code, "}")
	code = strings.TrimSpace(code)
	return strings.ToUpper(code)
}
