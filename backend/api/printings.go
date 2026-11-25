package api

import (
	"backend/models"
	"backend/scryfall"
	"backend/utils"
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	goscryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Printings returns all printings of a card (by oracle_id), combining every
// owned printing across all languages with the language-filtered Scryfall
// printings. Owned printings appear first, sorted newest-release first.
func (h *SearchHandler) Printings(c fiber.Ctx) error {
	oracleID := strings.TrimSpace(c.Params("oracle_id"))
	if oracleID == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "oracle_id is required")
	}

	ctx := c.RequestCtx()
	explicitLang := strings.TrimSpace(c.Query("lang"))

	defaultSearch, err := h.settingsService.Get(ctx, "scryfall_default_search")
	if err != nil {
		slog.Warn("failed to get scryfall_default_search setting", "component", "search", "error", err)
		defaultSearch = ""
	}

	defaultLang, err := h.settingsService.Get(ctx, "scryfall_default_language")
	if err != nil {
		slog.Warn("failed to get scryfall_default_language setting", "component", "search", "error", err)
		defaultLang = ""
	}

	query := "oracleid:" + oracleID
	if defaultSearch != "" {
		query = query + " " + defaultSearch
	}
	query = applyLanguage(query, explicitLang, defaultLang)

	searchRes, err := h.client.SearchWithOptions(ctx, query, scryfall.SearchOptions{
		Page:       1,
		UniqueMode: goscryfall.UniqueModePrints,
	})
	var catalogCards []goscryfall.Card
	scryfallFailed := false
	if err != nil {
		var scryfallErr *goscryfall.Error
		if !errors.As(err, &scryfallErr) || scryfallErr.Status != fiber.StatusNotFound {
			scryfallFailed = true
			slog.Warn("scryfall printings search failed; falling back to local inventory",
				"component", "search", "oracle_id", oracleID, "query", query, "error", err)
		}
	} else {
		catalogCards = searchRes.Cards
	}

	db := h.db.WithContext(ctx)
	var allInventory []models.Inventory
	if err := db.Preload("StorageLocation").Where("oracle_id = ?", oracleID).Find(&allInventory).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory items", "inventory lookup failed", err)
	}

	merged, err := h.mergeOwnedPrintings(ctx, db, oracleID, catalogCards, allInventory, !scryfallFailed)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to load owned printings", "owned printings lookup failed", err)
	}

	cards := make([]EnhancedCardResult, len(merged))
	for i, card := range merged {
		if card.OracleID == "" {
			card.OracleID = oracleID
		}
		cards[i] = EnhancedCardResult{
			CardResult: BuildCardResult(card),
			Inventory:  buildOracleInventoryData(card.ID, allInventory),
		}
	}

	ptrs := make([]*CardResult, len(cards))
	for i := range cards {
		ptrs[i] = &cards[i].CardResult
	}
	BackfillEnglishPrices(db, ptrs)

	availability, err := h.allocation.ComputeAvailability(ctx)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to compute deck availability", "allocation service failed", err)
	}

	oa := availability.OracleAvailabilityFor(oracleID)
	deck := DeckAvailability{
		Owned:  oa.Owned,
		Decked: oa.Owned - oa.Free,
		Free:   oa.Free,
	}

	results := make([]InventoryCard, len(cards))
	for i := range cards {
		results[i] = InventoryCard{
			EnhancedCardResult: cards[i],
			Deck:               deck,
		}
	}

	return c.JSON(results)
}

// mergeOwnedPrintings places every owned printing (quantity > 0) ahead of
// catalogCards and deduplicates by Scryfall ID.
func (h *SearchHandler) mergeOwnedPrintings(
	ctx context.Context,
	db *gorm.DB,
	oracleID string,
	catalogCards []goscryfall.Card,
	allInventory []models.Inventory,
	allowScryfallFallback bool,
) ([]goscryfall.Card, error) {
	catalogByID := make(map[string]goscryfall.Card, len(catalogCards))
	for _, card := range catalogCards {
		catalogByID[card.ID] = card
	}

	seenOwned := make(map[string]bool)
	var ownedIDs []string
	var missingIDs []string
	for _, inv := range allInventory {
		if inv.Quantity <= 0 || seenOwned[inv.ScryfallID] {
			continue
		}
		seenOwned[inv.ScryfallID] = true
		ownedIDs = append(ownedIDs, inv.ScryfallID)
		if _, ok := catalogByID[inv.ScryfallID]; !ok {
			missingIDs = append(missingIDs, inv.ScryfallID)
		}
	}

	var localCards map[string]goscryfall.Card
	if len(missingIDs) > 0 {
		var err error
		localCards, err = models.GetScryfallCardsByIDs(db, missingIDs)
		if err != nil {
			return nil, fmt.Errorf("fetching local cards: %w", err)
		}
	}

	ownedCards := make([]goscryfall.Card, 0, len(ownedIDs))
	for _, id := range ownedIDs {
		if card, ok := catalogByID[id]; ok {
			ownedCards = append(ownedCards, card)
			continue
		}
		if card, ok := localCards[id]; ok {
			ownedCards = append(ownedCards, card)
			continue
		}
		if !allowScryfallFallback {
			continue
		}
		card, err := h.client.GetByID(ctx, id)
		if err != nil {
			slog.Warn("failed to fetch owned printing", "component", "search",
				"oracle_id", oracleID, "scryfall_id", id, "error", err)
			continue
		}
		ownedCards = append(ownedCards, card)
	}

	slices.SortFunc(ownedCards, func(a, b goscryfall.Card) int {
		return cmp.Or(
			cmp.Compare(b.ReleasedAt.String(), a.ReleasedAt.String()),
			cmp.Compare(a.Set, b.Set),
			cmp.Compare(a.CollectorNumber, b.CollectorNumber),
			cmp.Compare(string(a.Lang), string(b.Lang)),
			cmp.Compare(a.ID, b.ID),
		)
	})

	emitted := make(map[string]bool, len(ownedCards)+len(catalogCards))
	merged := make([]goscryfall.Card, 0, len(ownedCards)+len(catalogCards))
	for _, card := range ownedCards {
		if emitted[card.ID] {
			continue
		}
		emitted[card.ID] = true
		merged = append(merged, card)
	}
	for _, card := range catalogCards {
		if emitted[card.ID] {
			continue
		}
		emitted[card.ID] = true
		merged = append(merged, card)
	}
	return merged, nil
}
