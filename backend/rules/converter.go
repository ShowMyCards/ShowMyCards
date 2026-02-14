package rules

import (
	"encoding/json"
	"fmt"
	"strconv"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// ScryfallCardToRuleData converts a scryfall.Card to a map suitable for rule evaluation
// Includes the treatment field for inventory-specific rules
func ScryfallCardToRuleData(card scryfall.Card, treatment string) (map[string]interface{}, error) {
	cardData := make(map[string]interface{})

	// Basic card fields
	cardData["name"] = card.Name
	cardData["set"] = card.Set
	cardData["set_name"] = card.SetName
	cardData["rarity"] = string(card.Rarity)
	cardData["type_line"] = card.TypeLine
	cardData["oracle_text"] = card.OracleText
	cardData["mana_cost"] = card.ManaCost
	cardData["cmc"] = card.CMC
	cardData["layout"] = string(card.Layout)
	cardData["promo"] = card.Promo
	cardData["reprint"] = card.Reprint
	cardData["digital"] = card.Digital
	cardData["reserved"] = card.Reserved
	cardData["foil"] = card.Foil
	cardData["nonfoil"] = card.NonFoil
	cardData["oversized"] = card.Oversized
	cardData["full_art"] = card.FullArt
	cardData["booster"] = card.Booster
	cardData["frame"] = string(card.Frame)
	cardData["border_color"] = string(card.BorderColor)
	cardData["collector_number"] = card.CollectorNumber
	cardData["artist"] = getString(card.Artist)

	// Power/toughness (nullable strings)
	cardData["power"] = getString(card.Power)
	cardData["toughness"] = getString(card.Toughness)

	// Loyalty (nullable string)
	if card.Loyalty != nil {
		cardData["loyalty"] = *card.Loyalty
	} else {
		cardData["loyalty"] = ""
	}

	// Arrays
	cardData["colors"] = card.Colors
	cardData["color_identity"] = card.ColorIdentity
	cardData["keywords"] = card.Keywords
	cardData["finishes"] = card.Finishes
	cardData["promo_types"] = card.PromoTypes

	// EDHREC rank (nullable int)
	if card.EDHRECRank != nil {
		cardData["edhrec_rank"] = *card.EDHRECRank
	} else {
		cardData["edhrec_rank"] = 0
	}

	// Prices - convert to floats for rule evaluation
	prices := make(map[string]interface{})
	prices["usd"] = parsePriceString(card.Prices.USD)
	prices["usd_foil"] = parsePriceString(card.Prices.USDFoil)
	prices["usd_etched"] = parsePriceString(card.Prices.USDEtched)
	prices["eur"] = parsePriceString(card.Prices.EUR)
	prices["eur_foil"] = parsePriceString(card.Prices.EURFoil)
	prices["tix"] = parsePriceString(card.Prices.Tix)
	cardData["prices"] = prices

	// Inventory-specific field
	cardData["treatment"] = treatment

	return cardData, nil
}

// getString safely extracts a string from a pointer, returning empty string if nil
func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// parsePriceString converts a price string to float64 for expression evaluation
func parsePriceString(price string) interface{} {
	if price == "" {
		return nil
	}

	if floatValue, err := strconv.ParseFloat(price, 64); err == nil {
		return floatValue
	}
	return nil
}

// RawJSONToRuleData converts raw Scryfall JSON to a map suitable for rule evaluation
// This avoids time parsing issues by working directly with the JSON
func RawJSONToRuleData(rawJSON string, treatment string) (map[string]interface{}, error) {
	// First unmarshal to a generic map to extract values
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	cardData := make(map[string]interface{})

	// Extract fields, using type assertions with defaults
	cardData["name"] = getStringFromJSON(jsonData, "name")
	cardData["set"] = getStringFromJSON(jsonData, "set")
	cardData["set_name"] = getStringFromJSON(jsonData, "set_name")
	cardData["rarity"] = getStringFromJSON(jsonData, "rarity")
	cardData["type_line"] = getStringFromJSON(jsonData, "type_line")
	cardData["oracle_text"] = getStringFromJSON(jsonData, "oracle_text")
	cardData["mana_cost"] = getStringFromJSON(jsonData, "mana_cost")
	cardData["cmc"] = getFloatFromJSON(jsonData, "cmc")
	cardData["layout"] = getStringFromJSON(jsonData, "layout")
	cardData["promo"] = getBoolFromJSON(jsonData, "promo")
	cardData["reprint"] = getBoolFromJSON(jsonData, "reprint")
	cardData["digital"] = getBoolFromJSON(jsonData, "digital")
	cardData["reserved"] = getBoolFromJSON(jsonData, "reserved")
	cardData["foil"] = getBoolFromJSON(jsonData, "foil")
	cardData["nonfoil"] = getBoolFromJSON(jsonData, "nonfoil")
	cardData["oversized"] = getBoolFromJSON(jsonData, "oversized")
	cardData["full_art"] = getBoolFromJSON(jsonData, "full_art")
	cardData["booster"] = getBoolFromJSON(jsonData, "booster")
	cardData["frame"] = getStringFromJSON(jsonData, "frame")
	cardData["border_color"] = getStringFromJSON(jsonData, "border_color")
	cardData["collector_number"] = getStringFromJSON(jsonData, "collector_number")
	cardData["artist"] = getStringFromJSON(jsonData, "artist")
	cardData["power"] = getStringFromJSON(jsonData, "power")
	cardData["toughness"] = getStringFromJSON(jsonData, "toughness")
	cardData["loyalty"] = getStringFromJSON(jsonData, "loyalty")

	// Arrays
	cardData["colors"] = getArrayFromJSON(jsonData, "colors")
	cardData["color_identity"] = getArrayFromJSON(jsonData, "color_identity")
	cardData["keywords"] = getArrayFromJSON(jsonData, "keywords")
	cardData["finishes"] = getArrayFromJSON(jsonData, "finishes")
	cardData["promo_types"] = getArrayFromJSON(jsonData, "promo_types")

	// EDHREC rank
	cardData["edhrec_rank"] = getIntFromJSON(jsonData, "edhrec_rank")

	// Prices
	prices := make(map[string]interface{})
	if pricesData, ok := jsonData["prices"].(map[string]interface{}); ok {
		prices["usd"] = parsePriceString(getStringFromMap(pricesData, "usd"))
		prices["usd_foil"] = parsePriceString(getStringFromMap(pricesData, "usd_foil"))
		prices["usd_etched"] = parsePriceString(getStringFromMap(pricesData, "usd_etched"))
		prices["eur"] = parsePriceString(getStringFromMap(pricesData, "eur"))
		prices["eur_foil"] = parsePriceString(getStringFromMap(pricesData, "eur_foil"))
		prices["tix"] = parsePriceString(getStringFromMap(pricesData, "tix"))
	}
	cardData["prices"] = prices

	// Inventory-specific field
	cardData["treatment"] = treatment

	return cardData, nil
}

// Helper functions for type-safe JSON extraction
func getStringFromJSON(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getStringFromMap(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getBoolFromJSON(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}

func getFloatFromJSON(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	return 0
}

func getIntFromJSON(data map[string]interface{}, key string) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getArrayFromJSON(data map[string]interface{}, key string) []interface{} {
	if val, ok := data[key].([]interface{}); ok {
		return val
	}
	return []interface{}{}
}
