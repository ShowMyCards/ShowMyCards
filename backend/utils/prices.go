package utils

import (
	"strconv"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// ParsePriceFromScryfall extracts the USD price for a specific treatment from scryfall.Prices.
// It maps card treatments to Scryfall price fields and falls back to nonfoil price if unavailable.
func ParsePriceFromScryfall(prices scryfall.Prices, treatment string) float64 {
	return parseTreatmentPrice(prices.USD, prices.USDFoil, prices.USDEtched, treatment)
}

// ParseEURPriceFromScryfall extracts a treatment-aware EUR price, falling back
// only within EUR. Etched cards use EUR foil prices, matching the frontend.
func ParseEURPriceFromScryfall(prices scryfall.Prices, treatment string) float64 {
	return parseTreatmentPrice(prices.EUR, prices.EURFoil, prices.EURFoil, treatment)
}

func parseTreatmentPrice(nonfoil, foil, etched, treatment string) float64 {
	// Map treatment to Scryfall price field
	var priceStr string
	switch treatment {
	case "foil":
		priceStr = foil
	case "etched":
		priceStr = etched
	case "nonfoil":
		priceStr = nonfoil
	default:
		// For other treatments (glossy, etc.), try foil first
		priceStr = foil
	}

	// Parse the price string to float64
	if priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			return price
		}
	}

	// Fallback to nonfoil price if treatment-specific price not available
	if treatment != "nonfoil" && nonfoil != "" {
		if price, err := strconv.ParseFloat(nonfoil, 64); err == nil {
			return price
		}
	}

	return 0.0
}
