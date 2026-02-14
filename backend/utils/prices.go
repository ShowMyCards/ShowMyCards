package utils

import (
	"strconv"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// ParsePriceFromScryfall extracts the USD price for a specific treatment from scryfall.Prices.
// It maps card treatments to Scryfall price fields and falls back to nonfoil price if unavailable.
func ParsePriceFromScryfall(prices scryfall.Prices, treatment string) float64 {
	// Map treatment to Scryfall price field
	var priceStr string
	switch treatment {
	case "foil":
		priceStr = prices.USDFoil
	case "etched":
		priceStr = prices.USDEtched
	case "nonfoil":
		priceStr = prices.USD
	default:
		// For other treatments (glossy, etc.), try foil first
		priceStr = prices.USDFoil
	}

	// Parse the price string to float64
	if priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			return price
		}
	}

	// Fallback to nonfoil price if treatment-specific price not available
	if treatment != "nonfoil" && prices.USD != "" {
		if price, err := strconv.ParseFloat(prices.USD, 64); err == nil {
			return price
		}
	}

	return 0.0
}
