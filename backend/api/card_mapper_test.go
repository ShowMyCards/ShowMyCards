package api

import (
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
)

func TestBuildCardPrices(t *testing.T) {
	prices := scryfall.Prices{
		USD:       "10.50",
		USDFoil:   "25.00",
		USDEtched: "30.00",
		EUR:       "9.00",
		EURFoil:   "22.00",
		Tix:       "5.00",
	}

	result := BuildCardPrices(prices)

	if result.USD != "10.50" {
		t.Errorf("USD: expected %q, got %q", "10.50", result.USD)
	}
	if result.USDFoil != "25.00" {
		t.Errorf("USDFoil: expected %q, got %q", "25.00", result.USDFoil)
	}
	if result.USDEtched != "30.00" {
		t.Errorf("USDEtched: expected %q, got %q", "30.00", result.USDEtched)
	}
	if result.EUR != "9.00" {
		t.Errorf("EUR: expected %q, got %q", "9.00", result.EUR)
	}
	if result.EURFoil != "22.00" {
		t.Errorf("EURFoil: expected %q, got %q", "22.00", result.EURFoil)
	}
	if result.Tix != "5.00" {
		t.Errorf("Tix: expected %q, got %q", "5.00", result.Tix)
	}
}

func TestBuildCardPrices_Empty(t *testing.T) {
	result := BuildCardPrices(scryfall.Prices{})

	if result.USD != "" || result.USDFoil != "" || result.USDEtched != "" {
		t.Errorf("expected empty prices, got %+v", result)
	}
}

func TestBuildCardResult(t *testing.T) {
	rank := 42
	card := scryfall.Card{
		ID:              "test-id",
		OracleID:        "oracle-id",
		Name:            "Lightning Bolt",
		Set:             "lea",
		SetName:         "Limited Edition Alpha",
		CollectorNumber: "161",
		ColorIdentity:   []scryfall.Color{"R"},
		Finishes:        []scryfall.Finish{"nonfoil"},
		EDHRECRank:      &rank,
		Prices: scryfall.Prices{
			USD: "500.00",
		},
	}

	result := BuildCardResult(card)

	if result.ID != "test-id" {
		t.Errorf("ID: expected %q, got %q", "test-id", result.ID)
	}
	if result.OracleID != "oracle-id" {
		t.Errorf("OracleID: expected %q, got %q", "oracle-id", result.OracleID)
	}
	if result.Name != "Lightning Bolt" {
		t.Errorf("Name: expected %q, got %q", "Lightning Bolt", result.Name)
	}
	if result.SetCode != "lea" {
		t.Errorf("SetCode: expected %q, got %q", "lea", result.SetCode)
	}
	if result.SetName != "Limited Edition Alpha" {
		t.Errorf("SetName: expected %q, got %q", "Limited Edition Alpha", result.SetName)
	}
	if result.CollectorNumber != "161" {
		t.Errorf("CollectorNumber: expected %q, got %q", "161", result.CollectorNumber)
	}
	if len(result.ColorIdentity) != 1 || result.ColorIdentity[0] != "R" {
		t.Errorf("ColorIdentity: expected [R], got %v", result.ColorIdentity)
	}
	if len(result.Finishes) != 1 || result.Finishes[0] != "nonfoil" {
		t.Errorf("Finishes: expected [nonfoil], got %v", result.Finishes)
	}
	if result.EDHRECRank == nil || *result.EDHRECRank != 42 {
		t.Errorf("EDHRECRank: expected 42, got %v", result.EDHRECRank)
	}
	if result.Prices.USD != "500.00" {
		t.Errorf("Prices.USD: expected %q, got %q", "500.00", result.Prices.USD)
	}
}

func TestBuildCardResult_EmptyCard(t *testing.T) {
	result := BuildCardResult(scryfall.Card{})

	if result.ID != "" {
		t.Errorf("expected empty ID, got %q", result.ID)
	}
	if result.Name != "" {
		t.Errorf("expected empty Name, got %q", result.Name)
	}
}
