package rules

import (
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
)

func TestParsePriceString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"valid price", "12.50", 12.5},
		{"zero price", "0.00", 0.0},
		{"integer price", "5", 5.0},
		{"empty string", "", nil},
		{"invalid string", "N/A", nil},
		{"whitespace", " ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePriceString(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				floatResult, ok := result.(float64)
				if !ok {
					t.Fatalf("expected float64, got %T", result)
				}
				if floatResult != tt.expected.(float64) {
					t.Errorf("expected %v, got %v", tt.expected, floatResult)
				}
			}
		})
	}
}

func TestRawJSONToRuleData_FullCard(t *testing.T) {
	rawJSON := `{
		"name": "Lightning Bolt",
		"set": "lea",
		"set_name": "Limited Edition Alpha",
		"rarity": "common",
		"type_line": "Instant",
		"oracle_text": "Lightning Bolt deals 3 damage to any target.",
		"mana_cost": "{R}",
		"cmc": 1.0,
		"layout": "normal",
		"promo": false,
		"reprint": true,
		"digital": false,
		"reserved": true,
		"foil": false,
		"nonfoil": true,
		"oversized": false,
		"full_art": false,
		"booster": true,
		"frame": "1993",
		"border_color": "black",
		"collector_number": "161",
		"artist": "Christopher Rush",
		"power": "3",
		"toughness": "2",
		"loyalty": "4",
		"colors": ["R"],
		"color_identity": ["R"],
		"keywords": ["instant"],
		"finishes": ["nonfoil"],
		"promo_types": [],
		"edhrec_rank": 42,
		"prices": {
			"usd": "0.25",
			"usd_foil": "1.50",
			"usd_etched": "",
			"eur": "0.20",
			"eur_foil": "1.30",
			"tix": "0.01"
		}
	}`

	cardData, err := RawJSONToRuleData(rawJSON, "nonfoil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// String fields
	if cardData["name"] != "Lightning Bolt" {
		t.Errorf("expected name 'Lightning Bolt', got '%v'", cardData["name"])
	}
	if cardData["set"] != "lea" {
		t.Errorf("expected set 'lea', got '%v'", cardData["set"])
	}
	if cardData["rarity"] != "common" {
		t.Errorf("expected rarity 'common', got '%v'", cardData["rarity"])
	}
	if cardData["artist"] != "Christopher Rush" {
		t.Errorf("expected artist 'Christopher Rush', got '%v'", cardData["artist"])
	}

	// Float field
	if cardData["cmc"] != 1.0 {
		t.Errorf("expected cmc 1.0, got %v", cardData["cmc"])
	}

	// Bool fields
	if cardData["promo"] != false {
		t.Errorf("expected promo false, got %v", cardData["promo"])
	}
	if cardData["reserved"] != true {
		t.Errorf("expected reserved true, got %v", cardData["reserved"])
	}
	if cardData["reprint"] != true {
		t.Errorf("expected reprint true, got %v", cardData["reprint"])
	}

	// Int field
	if cardData["edhrec_rank"] != 42 {
		t.Errorf("expected edhrec_rank 42, got %v", cardData["edhrec_rank"])
	}

	// Treatment injection
	if cardData["treatment"] != "nonfoil" {
		t.Errorf("expected treatment 'nonfoil', got '%v'", cardData["treatment"])
	}

	// Array field
	colors, ok := cardData["colors"].([]interface{})
	if !ok {
		t.Fatalf("expected colors to be []interface{}, got %T", cardData["colors"])
	}
	if len(colors) != 1 || colors[0] != "R" {
		t.Errorf("expected colors [R], got %v", colors)
	}

	// Prices sub-map
	prices, ok := cardData["prices"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected prices to be map[string]interface{}, got %T", cardData["prices"])
	}
	if prices["usd"] != 0.25 {
		t.Errorf("expected prices.usd 0.25, got %v", prices["usd"])
	}
	if prices["usd_foil"] != 1.5 {
		t.Errorf("expected prices.usd_foil 1.5, got %v", prices["usd_foil"])
	}
	if prices["usd_etched"] != nil {
		t.Errorf("expected prices.usd_etched nil (empty string), got %v", prices["usd_etched"])
	}
	if prices["eur"] != 0.2 {
		t.Errorf("expected prices.eur 0.2, got %v", prices["eur"])
	}
}

func TestRawJSONToRuleData_MissingFields(t *testing.T) {
	rawJSON := `{"name": "Minimal Card"}`

	cardData, err := RawJSONToRuleData(rawJSON, "foil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// String defaults
	if cardData["set"] != "" {
		t.Errorf("expected set '', got '%v'", cardData["set"])
	}
	if cardData["rarity"] != "" {
		t.Errorf("expected rarity '', got '%v'", cardData["rarity"])
	}

	// Float default
	if cardData["cmc"] != 0.0 {
		t.Errorf("expected cmc 0.0, got %v", cardData["cmc"])
	}

	// Bool default
	if cardData["promo"] != false {
		t.Errorf("expected promo false, got %v", cardData["promo"])
	}

	// Int default
	if cardData["edhrec_rank"] != 0 {
		t.Errorf("expected edhrec_rank 0, got %v", cardData["edhrec_rank"])
	}

	// Array default (empty slice, not nil)
	colors, ok := cardData["colors"].([]interface{})
	if !ok {
		t.Fatalf("expected colors to be []interface{}, got %T", cardData["colors"])
	}
	if len(colors) != 0 {
		t.Errorf("expected empty colors, got %v", colors)
	}

	// Treatment injection still works
	if cardData["treatment"] != "foil" {
		t.Errorf("expected treatment 'foil', got '%v'", cardData["treatment"])
	}

	// Prices map exists but with nil values (no prices object in JSON)
	prices, ok := cardData["prices"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected prices to be map[string]interface{}, got %T", cardData["prices"])
	}
	// When prices key is missing from JSON, the prices map is empty (no keys extracted)
	if len(prices) != 0 {
		t.Errorf("expected empty prices map when no prices in JSON, got %v", prices)
	}
}

func TestRawJSONToRuleData_InvalidJSON(t *testing.T) {
	_, err := RawJSONToRuleData("{invalid", "nonfoil")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestRawJSONToRuleData_PricesParsing(t *testing.T) {
	rawJSON := `{
		"name": "Test Card",
		"prices": {
			"usd": "12.50",
			"usd_foil": "",
			"usd_etched": "N/A",
			"eur": "3.00"
		}
	}`

	cardData, err := RawJSONToRuleData(rawJSON, "nonfoil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices, ok := cardData["prices"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected prices to be map[string]interface{}, got %T", cardData["prices"])
	}

	if prices["usd"] != 12.5 {
		t.Errorf("expected usd 12.5, got %v", prices["usd"])
	}
	if prices["usd_foil"] != nil {
		t.Errorf("expected usd_foil nil (empty string), got %v", prices["usd_foil"])
	}
	if prices["usd_etched"] != nil {
		t.Errorf("expected usd_etched nil (malformed), got %v", prices["usd_etched"])
	}
	if prices["eur"] != 3.0 {
		t.Errorf("expected eur 3.0, got %v", prices["eur"])
	}
	// Missing keys from JSON should still produce nil via parsePriceString("")
	if prices["eur_foil"] != nil {
		t.Errorf("expected eur_foil nil (missing from JSON), got %v", prices["eur_foil"])
	}
	if prices["tix"] != nil {
		t.Errorf("expected tix nil (missing from JSON), got %v", prices["tix"])
	}
}

func TestRawJSONToRuleData_ArrayFields(t *testing.T) {
	rawJSON := `{
		"name": "Omnath, Locus of Creation",
		"colors": ["R", "G", "W", "U"],
		"color_identity": ["R", "G", "W", "U"],
		"keywords": ["landfall"],
		"finishes": ["nonfoil", "foil"],
		"promo_types": ["boosterfun"]
	}`

	cardData, err := RawJSONToRuleData(rawJSON, "nonfoil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	colors := cardData["colors"].([]interface{})
	if len(colors) != 4 {
		t.Errorf("expected 4 colors, got %d", len(colors))
	}

	keywords := cardData["keywords"].([]interface{})
	if len(keywords) != 1 || keywords[0] != "landfall" {
		t.Errorf("expected keywords [landfall], got %v", keywords)
	}

	finishes := cardData["finishes"].([]interface{})
	if len(finishes) != 2 {
		t.Errorf("expected 2 finishes, got %d", len(finishes))
	}

	promoTypes := cardData["promo_types"].([]interface{})
	if len(promoTypes) != 1 || promoTypes[0] != "boosterfun" {
		t.Errorf("expected promo_types [boosterfun], got %v", promoTypes)
	}
}

func TestScryfallCardToRuleData_FullCard(t *testing.T) {
	artist := "Christopher Rush"
	power := "3"
	toughness := "2"
	loyalty := "4"
	edhrecRank := 42

	card := scryfall.Card{
		Name:            "Lightning Bolt",
		Set:             "lea",
		SetName:         "Limited Edition Alpha",
		Rarity:          "common",
		TypeLine:        "Instant",
		OracleText:      "Lightning Bolt deals 3 damage to any target.",
		ManaCost:        "{R}",
		CMC:             1.0,
		Layout:          scryfall.LayoutNormal,
		Promo:           false,
		Reprint:         true,
		Digital:         false,
		Reserved:        true,
		Foil:            false,
		NonFoil:         true,
		Oversized:       false,
		FullArt:         false,
		Booster:         true,
		Frame:           "1993",
		BorderColor:     "black",
		CollectorNumber: "161",
		Artist:          &artist,
		Power:           &power,
		Toughness:       &toughness,
		Loyalty:         &loyalty,
		EDHRECRank:      &edhrecRank,
		Colors:          []scryfall.Color{"R"},
		ColorIdentity:   []scryfall.Color{"R"},
		Keywords:        []string{"instant"},
		Finishes:        []scryfall.Finish{"nonfoil"},
		PromoTypes:      []string{},
		Prices: scryfall.Prices{
			USD:     "0.25",
			USDFoil: "1.50",
		},
	}

	cardData, err := ScryfallCardToRuleData(card, "nonfoil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify basic fields
	if cardData["name"] != "Lightning Bolt" {
		t.Errorf("expected name 'Lightning Bolt', got '%v'", cardData["name"])
	}
	if cardData["rarity"] != "common" {
		t.Errorf("expected rarity 'common', got '%v'", cardData["rarity"])
	}
	if cardData["cmc"] != 1.0 {
		t.Errorf("expected cmc 1.0, got %v", cardData["cmc"])
	}
	if cardData["layout"] != "normal" {
		t.Errorf("expected layout 'normal', got '%v'", cardData["layout"])
	}
	if cardData["reserved"] != true {
		t.Errorf("expected reserved true, got %v", cardData["reserved"])
	}

	// Pointer fields dereferenced
	if cardData["artist"] != "Christopher Rush" {
		t.Errorf("expected artist 'Christopher Rush', got '%v'", cardData["artist"])
	}
	if cardData["power"] != "3" {
		t.Errorf("expected power '3', got '%v'", cardData["power"])
	}
	if cardData["loyalty"] != "4" {
		t.Errorf("expected loyalty '4', got '%v'", cardData["loyalty"])
	}
	if cardData["edhrec_rank"] != 42 {
		t.Errorf("expected edhrec_rank 42, got %v", cardData["edhrec_rank"])
	}

	// Treatment injection
	if cardData["treatment"] != "nonfoil" {
		t.Errorf("expected treatment 'nonfoil', got '%v'", cardData["treatment"])
	}

	// Prices
	prices := cardData["prices"].(map[string]interface{})
	if prices["usd"] != 0.25 {
		t.Errorf("expected prices.usd 0.25, got %v", prices["usd"])
	}
	if prices["usd_foil"] != 1.5 {
		t.Errorf("expected prices.usd_foil 1.5, got %v", prices["usd_foil"])
	}
}

func TestScryfallCardToRuleData_NilPointerFields(t *testing.T) {
	card := scryfall.Card{
		Name: "Force of Will",
		// All pointer fields left nil: Artist, Power, Toughness, Loyalty, EDHRECRank
	}

	cardData, err := ScryfallCardToRuleData(card, "foil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cardData["artist"] != "" {
		t.Errorf("expected artist '' for nil pointer, got '%v'", cardData["artist"])
	}
	if cardData["power"] != "" {
		t.Errorf("expected power '' for nil pointer, got '%v'", cardData["power"])
	}
	if cardData["toughness"] != "" {
		t.Errorf("expected toughness '' for nil pointer, got '%v'", cardData["toughness"])
	}
	if cardData["loyalty"] != "" {
		t.Errorf("expected loyalty '' for nil pointer, got '%v'", cardData["loyalty"])
	}
	if cardData["edhrec_rank"] != 0 {
		t.Errorf("expected edhrec_rank 0 for nil pointer, got %v", cardData["edhrec_rank"])
	}
	if cardData["treatment"] != "foil" {
		t.Errorf("expected treatment 'foil', got '%v'", cardData["treatment"])
	}
}

func TestScryfallCardToRuleData_PriceParsing(t *testing.T) {
	card := scryfall.Card{
		Name: "Test Card",
		Prices: scryfall.Prices{
			USD:       "12.50",
			USDFoil:   "",
			USDEtched: "N/A",
			EUR:       "3.00",
		},
	}

	cardData, err := ScryfallCardToRuleData(card, "nonfoil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices := cardData["prices"].(map[string]interface{})
	if prices["usd"] != 12.5 {
		t.Errorf("expected usd 12.5, got %v", prices["usd"])
	}
	if prices["usd_foil"] != nil {
		t.Errorf("expected usd_foil nil (empty), got %v", prices["usd_foil"])
	}
	if prices["usd_etched"] != nil {
		t.Errorf("expected usd_etched nil (malformed), got %v", prices["usd_etched"])
	}
	if prices["eur"] != 3.0 {
		t.Errorf("expected eur 3.0, got %v", prices["eur"])
	}
}
