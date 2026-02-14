package utils

import (
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
)

func TestParsePriceFromScryfall(t *testing.T) {
	tests := []struct {
		name      string
		prices    scryfall.Prices
		treatment string
		expected  float64
	}{
		{
			name:      "nonfoil uses USD",
			prices:    scryfall.Prices{USD: "12.50"},
			treatment: "nonfoil",
			expected:  12.5,
		},
		{
			name:      "foil uses USDFoil",
			prices:    scryfall.Prices{USDFoil: "25.00"},
			treatment: "foil",
			expected:  25.0,
		},
		{
			name:      "etched uses USDEtched",
			prices:    scryfall.Prices{USDEtched: "8.00"},
			treatment: "etched",
			expected:  8.0,
		},
		{
			name:      "unknown treatment tries USDFoil first",
			prices:    scryfall.Prices{USDFoil: "15.00", USD: "10.00"},
			treatment: "glossy",
			expected:  15.0,
		},
		{
			name:      "foil fallback to USD when USDFoil empty",
			prices:    scryfall.Prices{USD: "10.00", USDFoil: ""},
			treatment: "foil",
			expected:  10.0,
		},
		{
			name:      "etched fallback to USD when USDEtched empty",
			prices:    scryfall.Prices{USD: "5.00", USDEtched: ""},
			treatment: "etched",
			expected:  5.0,
		},
		{
			name:      "nonfoil no fallback returns zero",
			prices:    scryfall.Prices{USD: ""},
			treatment: "nonfoil",
			expected:  0.0,
		},
		{
			name:      "all prices empty nonfoil",
			prices:    scryfall.Prices{},
			treatment: "nonfoil",
			expected:  0.0,
		},
		{
			name:      "all prices empty foil",
			prices:    scryfall.Prices{},
			treatment: "foil",
			expected:  0.0,
		},
		{
			name:      "malformed price string",
			prices:    scryfall.Prices{USD: "not-a-number"},
			treatment: "nonfoil",
			expected:  0.0,
		},
		{
			name:      "unknown treatment falls back to USD when USDFoil empty",
			prices:    scryfall.Prices{USD: "7.00", USDFoil: ""},
			treatment: "glossy",
			expected:  7.0,
		},
		{
			name:      "unknown treatment no prices at all",
			prices:    scryfall.Prices{},
			treatment: "glossy",
			expected:  0.0,
		},
		{
			name:      "foil with both prices uses foil",
			prices:    scryfall.Prices{USD: "2.00", USDFoil: "8.00"},
			treatment: "foil",
			expected:  8.0,
		},
		{
			name:      "malformed foil falls back to valid USD",
			prices:    scryfall.Prices{USD: "3.00", USDFoil: "N/A"},
			treatment: "foil",
			expected:  3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePriceFromScryfall(tt.prices, tt.treatment)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
