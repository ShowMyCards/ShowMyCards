package api

import (
	"backend/utils"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// BuildCardPrices extracts pricing information from a Scryfall card into our API type.
func BuildCardPrices(prices scryfall.Prices) CardPrices {
	return CardPrices{
		USD:       prices.USD,
		USDFoil:   prices.USDFoil,
		USDEtched: prices.USDEtched,
		EUR:       prices.EUR,
		EURFoil:   prices.EURFoil,
		Tix:       prices.Tix,
	}
}

// BuildCardResult creates a CardResult from a Scryfall card, extracting all
// display fields and converting enum types to strings.
func BuildCardResult(card scryfall.Card) CardResult {
	return CardResult{
		ID:              card.ID,
		OracleID:        card.OracleID,
		Name:            card.Name,
		SetCode:         card.Set,
		SetName:         card.SetName,
		CollectorNumber: card.CollectorNumber,
		ColorIdentity:   utils.ConvertEnumSliceToStrings(card.ColorIdentity),
		Finishes:        utils.ConvertEnumSliceToStrings(card.Finishes),
		FrameEffects:    utils.ConvertEnumSliceToStrings(card.FrameEffects),
		PromoTypes:      card.PromoTypes,
		EDHRECRank:      card.EDHRECRank,
		Prices:          BuildCardPrices(card.Prices),
		ImageURI:        utils.ExtractCardImageURI(card),
	}
}
