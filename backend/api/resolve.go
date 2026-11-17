package api

import (
	"fmt"
	"strings"

	"backend/models"
	"backend/utils"

	scryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
)

// MaxResolveItems bounds a single resolve batch.
const MaxResolveItems = 1000

// ResolveItem is a single decklist line to resolve against the locally-ingested
// Scryfall bulk data. Provide Set + CollectorNumber to pin a specific printing,
// or Name to resolve any printing of a card.
//
// tygo:export
type ResolveItem struct {
	Set             string `json:"set,omitempty"`
	CollectorNumber string `json:"collector_number,omitempty"`
	Name            string `json:"name,omitempty"`
}

// ResolveRequest is a batch of lines to resolve locally.
//
// tygo:export
type ResolveRequest struct {
	Items    []ResolveItem `json:"items"`
	Language string        `json:"language,omitempty"`
}

// ResolveResult is the outcome for one requested item, aligned by index with the
// request. Found is false when the card is not in the local database, in which
// case the caller should fall back to the live Scryfall search for that line.
//
// tygo:export
type ResolveResult struct {
	Found bool        `json:"found"`
	Card  *CardResult `json:"card,omitempty"`
}

// ResolveResponse holds per-item results in request order.
//
// tygo:export
type ResolveResponse struct {
	Results []ResolveResult `json:"results"`
}

// Resolve resolves decklist lines against the locally-ingested Scryfall bulk data
// instead of the rate-limited live API. Pinned lines match on
// (set_code, collector_number) literally — so promo/stamp collector suffixes
// (e.g. "33p", "115s") resolve correctly where Scryfall's cn: search does not —
// and name-only lines match on an exact name. Unmatched lines return Found=false
// so the caller can fall back to Scryfall for just those, keeping bulk imports
// fast without any outbound rate-limit pressure.
func (h *SearchHandler) Resolve(c fiber.Ctx) error {
	var req ResolveRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.Items) == 0 {
		return c.JSON(ResolveResponse{Results: []ResolveResult{}})
	}
	if len(req.Items) > MaxResolveItems {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("too many items (max %d)", MaxResolveItems))
	}

	lang := req.Language
	if lang == "" {
		lang = "en"
	}

	// Collect the print keys and names into two batch queries. Scryfall stores set
	// codes lower-cased, so normalise here.
	printKeys := make([]models.PrintKey, 0, len(req.Items))
	names := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Set != "" && item.CollectorNumber != "" {
			printKeys = append(printKeys, models.PrintKey{
				SetCode:         strings.ToLower(item.Set),
				CollectorNumber: item.CollectorNumber,
			})
		} else if item.Name != "" {
			names = append(names, item.Name)
		}
	}

	db := h.db.WithContext(c.RequestCtx())

	printMap, err := models.GetScryfallCardsByPrints(db, printKeys, lang)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to resolve cards", "print resolution failed", err)
	}
	nameMap, err := models.GetScryfallCardsByNames(db, names, lang)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to resolve cards", "name resolution failed", err)
	}

	results := make([]ResolveResult, len(req.Items))
	found := make([]*CardResult, 0, len(req.Items))
	for i, item := range req.Items {
		var sc *scryfall.Card
		if item.Set != "" && item.CollectorNumber != "" {
			key := models.PrintKey{
				SetCode:         strings.ToLower(item.Set),
				CollectorNumber: item.CollectorNumber,
			}
			if card, ok := printMap[key.String()]; ok {
				sc = &card
			}
		} else if item.Name != "" {
			if card, ok := nameMap[strings.ToLower(item.Name)]; ok {
				sc = &card
			}
		}

		if sc == nil {
			results[i] = ResolveResult{Found: false}
			continue
		}

		cr := BuildCardResult(*sc)
		results[i] = ResolveResult{Found: true, Card: &cr}
		found = append(found, results[i].Card)
	}

	// Non-English printings carry empty prices; back-fill from the English print
	// (mirrors the search handler).
	BackfillEnglishPrices(db, found)

	return c.JSON(ResolveResponse{Results: results})
}
