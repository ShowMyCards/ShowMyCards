package api

import (
	"backend/models"
	"backend/utils"
	"log/slog"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
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

// IsEmpty reports whether no price field is set.
func (p CardPrices) IsEmpty() bool {
	return p.USD == "" && p.USDFoil == "" && p.USDEtched == "" &&
		p.EUR == "" && p.EURFoil == "" && p.Tix == ""
}

// BackfillEnglishPrices fills empty prices on non-English results from the
// English printing of the same set + collector number, using locally imported
// card data. Scryfall mostly populates prices only on the English printing, so
// non-English cards otherwise have no price to display.
//
// Best-effort: on a query error or when no English printing is found, the
// affected results are left unchanged (so they simply show no price, as before).
func BackfillEnglishPrices(db *gorm.DB, results []*CardResult) {
	// Collect prints that need a fallback: non-English, no price of their own,
	// and identifiable by set + collector number.
	var keys []models.PrintKey
	for _, r := range results {
		if r.Language == "en" || !r.Prices.IsEmpty() {
			continue
		}
		if r.SetCode == "" || r.CollectorNumber == "" {
			continue
		}
		keys = append(keys, models.PrintKey{SetCode: r.SetCode, CollectorNumber: r.CollectorNumber})
	}
	if len(keys) == 0 {
		return
	}

	pricesByPrint, err := models.GetEnglishPricesByPrint(db, keys)
	if err != nil {
		slog.Warn("english price fallback lookup failed", "component", "search", "error", err)
		return
	}

	for _, r := range results {
		if r.Language == "en" || !r.Prices.IsEmpty() {
			continue
		}
		key := models.PrintKey{SetCode: r.SetCode, CollectorNumber: r.CollectorNumber}
		if prices, ok := pricesByPrint[key.String()]; ok {
			r.Prices = BuildCardPrices(prices)
		}
	}
}

// BuildCardResult creates a CardResult from a Scryfall card, extracting all
// display fields and converting enum types to strings.
func BuildCardResult(card scryfall.Card) CardResult {
	return CardResult{
		ID:              card.ID,
		OracleID:        card.OracleID,
		Name:            card.Name,
		PrintedName:     card.PrintedName,
		SetCode:         card.Set,
		SetName:         card.SetName,
		CollectorNumber: card.CollectorNumber,
		Language:        string(card.Lang),
		ColorIdentity:   utils.ConvertEnumSliceToStrings(card.ColorIdentity),
		Finishes:        utils.ConvertEnumSliceToStrings(card.Finishes),
		FrameEffects:    utils.ConvertEnumSliceToStrings(card.FrameEffects),
		PromoTypes:      card.PromoTypes,
		EDHRECRank:      card.EDHRECRank,
		Prices:          BuildCardPrices(card.Prices),
		ImageURI:        utils.ExtractCardImageURI(card),
	}
}
