package api

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DeckHandler handles deck endpoints.
type DeckHandler struct {
	db         *gorm.DB
	allocation *services.AllocationService
}

// NewDeckHandler creates a new deck handler.
func NewDeckHandler(db *gorm.DB) *DeckHandler {
	return &DeckHandler{
		db:         db,
		allocation: services.NewAllocationService(db),
	}
}

// DeckSummary represents a deck with summary statistics.
//
// AggregateShortfall is the under-owned shortfall — the number of cards the deck
// wants that the user does not physically own (Σ max(0, desired − owned_O) over
// the deck's demand-zone items). It is independent of other decks; the
// cross-deck over-commitment signal lives on the deck-detail items endpoint.
// See FR98/IMPLEMENTATION_PLAN.md §2.
//
// tygo:export
type DeckSummary struct {
	ID                 uint   `json:"id"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	TotalItems         int    `json:"total_items"`
	TotalCardsDemand   int    `json:"total_cards_demand"`
	AggregateShortfall int    `json:"aggregate_shortfall"`
}

// List returns all decks with summary statistics.
func (h *DeckHandler) List(c fiber.Ctx) error {
	var decks []models.Deck
	if err := h.db.WithContext(c.RequestCtx()).Preload("Items").Order("created_at DESC").Find(&decks).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch decks", "database query failed", err)
	}

	// Compute the availability map once and reuse it across every deck rather
	// than running the allocation service per deck in the loop.
	availability, err := h.allocation.ComputeAvailability(c.RequestCtx())
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to compute deck availability", "allocation service failed", err)
	}

	summaries := make([]DeckSummary, len(decks))
	for i, deck := range decks {
		// Roll demand-zone rows up per Oracle before measuring shortfall so a
		// card split across zones (e.g. 3 main + 1 side) is counted once against
		// the owned pool, matching the deck-detail items endpoint. See ListItems.
		totalDemand := 0
		demandByOracle := make(map[string]int)
		for _, item := range deck.Items {
			if !item.Zone.CountsAsDemand() {
				continue
			}
			totalDemand += item.DesiredQuantity
			demandByOracle[item.OracleID] += item.DesiredQuantity
		}

		shortfall := 0
		for oracleID, demand := range demandByOracle {
			if owned := availability.OracleAvailabilityFor(oracleID).Owned; demand > owned {
				shortfall += demand - owned
			}
		}

		summaries[i] = DeckSummary{
			ID:                 deck.ID,
			CreatedAt:          deck.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          deck.UpdatedAt.Format(time.RFC3339),
			Name:               deck.Name,
			Description:        deck.Description,
			TotalItems:         len(deck.Items),
			TotalCardsDemand:   totalDemand,
			AggregateShortfall: shortfall,
		}
	}

	return c.JSON(summaries)
}

// Get returns a single deck by ID.
func (h *DeckHandler) Get(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var deck models.Deck
	if err := h.db.WithContext(c.RequestCtx()).First(&deck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck", "database query failed", err)
	}

	return c.JSON(deck)
}

// CreateDeckRequest represents the request body for creating a deck.
//
// tygo:export
type CreateDeckRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create creates a new deck.
func (h *DeckHandler) Create(c fiber.Ctx) error {
	var req CreateDeckRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	var validationErrors []error
	validationErrors = append(validationErrors, utils.ValidateRequired(req.Name, "name"))
	validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Name, 255, "name"))
	validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Description, 1000, "description"))

	if err := utils.CombineErrors(validationErrors); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	deck := models.Deck{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.db.WithContext(c.RequestCtx()).Create(&deck).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create deck", "database insert failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(deck)
}

// UpdateDeckRequest represents the request body for updating a deck.
//
// tygo:export
type UpdateDeckRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Update updates an existing deck.
func (h *DeckHandler) Update(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var deck models.Deck
	if err := h.db.WithContext(c.RequestCtx()).First(&deck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck", "database query failed", err)
	}

	var req UpdateDeckRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	var validationErrors []error
	if req.Name != "" {
		validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Name, 255, "name"))
	}
	validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Description, 1000, "description"))
	if err := utils.CombineErrors(validationErrors); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	if req.Name != "" {
		deck.Name = req.Name
	}
	// Allow empty description to clear it.
	deck.Description = req.Description

	if err := h.db.WithContext(c.RequestCtx()).Save(&deck).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update deck", "database update failed", err)
	}

	return c.JSON(deck)
}

// Delete deletes a deck and all its items.
func (h *DeckHandler) Delete(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		var deck models.Deck
		if err := tx.First(&deck, id).Error; err != nil {
			return err
		}

		// Delete items explicitly inside the transaction rather than relying
		// on the GORM CASCADE constraint, mirroring the lists pattern.
		if err := tx.Where("deck_id = ?", id).Delete(&models.DeckItem{}).Error; err != nil {
			return err
		}

		return tx.Delete(&deck).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete deck", "database delete failed", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// EnrichedDeckItem represents a deck item enriched with card data and
// availability information.
//
// Owned is the Oracle-level owned count. UnderOwned is true when this item's
// desired quantity exceeds what the user physically owns of the Oracle.
// OverCommitted is the Oracle's cross-deck over-commitment flag (you own enough
// for this deck alone, but not for every deck at once, or a pinned printing is
// over-subscribed). These are two distinct signals — see
// FR98/IMPLEMENTATION_PLAN.md §2 "Two distinct deck-detail signals".
//
// tygo:export
type EnrichedDeckItem struct {
	ID              uint            `json:"id"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	DeckID          uint            `json:"deck_id"`
	OracleID        string          `json:"oracle_id"`
	ScryfallID      string          `json:"scryfall_id"`
	Treatment       string          `json:"treatment"`
	Zone            models.DeckZone `json:"zone"`
	DesiredQuantity int             `json:"desired_quantity"`
	// Enriched fields (populated from Scryfall card data in 1b).
	Name            string   `json:"name,omitempty"`
	SetName         string   `json:"set_name,omitempty"`
	SetCode         string   `json:"set_code,omitempty"`
	CollectorNumber string   `json:"collector_number,omitempty"`
	Rarity          string   `json:"rarity,omitempty"`
	Finishes        []string `json:"finishes,omitempty"`
	// Availability (populated by the allocation service in 1b).
	Owned         int  `json:"owned"`
	UnderOwned    bool `json:"under_owned"`
	OverCommitted bool `json:"over_committed"`
}

// DeckItemsResponse represents the items in a deck, grouped by zone, with
// aggregate availability information.
//
// AggregateShortfall is the deck's under-owned shortfall (Σ max(0, desired −
// owned_O) over demand-zone items). Maybe-board items are returned but never
// contribute to the shortfall.
//
// tygo:export
type DeckItemsResponse struct {
	DeckID             uint               `json:"deck_id"`
	Command            []EnrichedDeckItem `json:"command"`
	Main               []EnrichedDeckItem `json:"main"`
	Side               []EnrichedDeckItem `json:"side"`
	Maybe              []EnrichedDeckItem `json:"maybe"`
	AggregateShortfall int                `json:"aggregate_shortfall"`
}

// ListItems returns the items in a deck, grouped by zone, enriched with card
// data and availability from the allocation service.
func (h *DeckHandler) ListItems(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	ctx := c.RequestCtx()

	var deck models.Deck
	if err := h.db.WithContext(ctx).First(&deck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck", "database query failed", err)
	}

	var items []models.DeckItem
	if err := h.db.WithContext(ctx).
		Where("deck_id = ?", deck.ID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck items", "database query failed", err)
	}

	availability, err := h.allocation.ComputeAvailability(ctx)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to compute deck availability", "allocation service failed", err)
	}

	// Roll demand-zone rows up per Oracle. A card may legitimately appear in
	// more than one demand zone of a single deck (e.g. 3 maindeck + 1 in the
	// sideboard); those copies are all committed, so under-ownership and the
	// aggregate shortfall must measure the deck's combined demand for an Oracle
	// against what is owned, not each row independently. (The zone is part of
	// the deck_items unique index precisely to allow a card in several zones.)
	deckDemandByOracle := make(map[string]int)
	for _, item := range items {
		if item.Zone.CountsAsDemand() {
			deckDemandByOracle[item.OracleID] += item.DesiredQuantity
		}
	}

	enriched := h.enrichDeckItems(ctx, items, availability, deckDemandByOracle)

	resp := DeckItemsResponse{
		DeckID:  deck.ID,
		Command: []EnrichedDeckItem{},
		Main:    []EnrichedDeckItem{},
		Side:    []EnrichedDeckItem{},
		Maybe:   []EnrichedDeckItem{},
	}

	for _, item := range enriched {
		switch item.Zone {
		case models.ZoneCommand:
			resp.Command = append(resp.Command, item)
		case models.ZoneMain:
			resp.Main = append(resp.Main, item)
		case models.ZoneSide:
			resp.Side = append(resp.Side, item)
		case models.ZoneMaybe:
			resp.Maybe = append(resp.Maybe, item)
		}
	}

	// Aggregate shortfall = Σ over distinct demand-zone Oracles of
	// max(0, deckDemand − owned). Computed per Oracle (not per row) so a card
	// split across zones is counted once against the owned pool.
	for oracleID, demand := range deckDemandByOracle {
		if owned := availability.OracleAvailabilityFor(oracleID).Owned; demand > owned {
			resp.AggregateShortfall += demand - owned
		}
	}

	return c.JSON(resp)
}

// enrichDeckItems maps deck items to EnrichedDeckItem, adding Scryfall card
// metadata and per-item availability. Pinned items (non-empty scryfall_id) are
// enriched from their exact printing; any-printing items are enriched from a
// representative printing of the Oracle. Missing card data is left empty rather
// than failing the request.
func (h *DeckHandler) enrichDeckItems(ctx context.Context, items []models.DeckItem, availability services.AvailabilityMap, deckDemandByOracle map[string]int) []EnrichedDeckItem {
	// Bulk-fetch pinned printings by Scryfall ID.
	pinnedIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.ScryfallID != "" {
			pinnedIDs = append(pinnedIDs, item.ScryfallID)
		}
	}
	scryfallCardMap, err := models.GetScryfallCardsByIDs(h.db.WithContext(ctx), pinnedIDs)
	if err != nil {
		slog.Warn("failed to fetch card data for deck enrichment", "component", "decks", "error", err)
	}

	// For any-printing items, fetch a representative printing per Oracle ID.
	oracleCardMap := h.representativeCardsByOracle(ctx, items)

	enriched := make([]EnrichedDeckItem, len(items))
	for i, item := range items {
		e := EnrichedDeckItem{
			ID:              item.ID,
			CreatedAt:       item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       item.UpdatedAt.Format(time.RFC3339),
			DeckID:          item.DeckID,
			OracleID:        item.OracleID,
			ScryfallID:      item.ScryfallID,
			Treatment:       item.Treatment,
			Zone:            item.Zone,
			DesiredQuantity: item.DesiredQuantity,
		}

		if item.ScryfallID != "" {
			if card, ok := scryfallCardMap[item.ScryfallID]; ok {
				e.Name = card.Name
				e.SetName = card.SetName
				e.SetCode = card.Set
				e.CollectorNumber = card.CollectorNumber
				e.Rarity = string(card.Rarity)
				e.Finishes = utils.ConvertEnumSliceToStrings(card.Finishes)
			}
		} else if card, ok := oracleCardMap[item.OracleID]; ok {
			e.Name = card.Name
			e.SetName = card.SetName
			e.SetCode = card.Set
			e.CollectorNumber = card.CollectorNumber
			e.Rarity = string(card.Rarity)
			e.Finishes = utils.ConvertEnumSliceToStrings(card.Finishes)
		}

		// Availability is computed at the Oracle level. Treatment is not part of
		// the M1 over-commitment math (see FR98/IMPLEMENTATION_PLAN.md §2).
		a := availability.OracleAvailabilityFor(item.OracleID)
		e.Owned = a.Owned

		// under_owned reflects the deck's total committed demand for this Oracle
		// (summed across demand zones), not this single row, so a card split
		// across main + side is short only when the combined demand exceeds what
		// is owned. Maybe-board rows are not committed demand, so they fall back
		// to their own desired quantity.
		relevantDesired := item.DesiredQuantity
		if item.Zone.CountsAsDemand() {
			relevantDesired = deckDemandByOracle[item.OracleID]
		}
		e.UnderOwned = relevantDesired > a.Owned
		// over_committed and under_owned are distinct signals: over_committed means
		// "you own enough for this deck alone, but not across all decks". Suppress
		// it when the item is already under-owned so the two are never conflated
		// (FR98/IMPLEMENTATION_PLAN.md §2 "Two distinct deck-detail signals").
		e.OverCommitted = a.OverCommitted && !e.UnderOwned

		enriched[i] = e
	}
	return enriched
}

// representativeCardsByOracle returns one Scryfall card per Oracle ID for the
// any-printing items in the slice, so they can be enriched with a name and set
// even though no specific printing was pinned.
func (h *DeckHandler) representativeCardsByOracle(ctx context.Context, items []models.DeckItem) map[string]scryfall.Card {
	oracleIDs := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		if item.ScryfallID == "" && !seen[item.OracleID] {
			seen[item.OracleID] = true
			oracleIDs = append(oracleIDs, item.OracleID)
		}
	}

	out := make(map[string]scryfall.Card, len(oracleIDs))
	if len(oracleIDs) == 0 {
		return out
	}

	// Fetch the first card row for each Oracle ID. cards.oracle_id is indexed.
	var cards []models.Card
	if err := h.db.WithContext(ctx).
		Where("oracle_id IN ?", oracleIDs).
		Find(&cards).Error; err != nil {
		slog.Warn("failed to fetch representative cards for deck enrichment", "component", "decks", "error", err)
		return out
	}

	for _, card := range cards {
		if _, ok := out[card.OracleID]; ok {
			continue // keep the first representative printing per Oracle
		}
		sc, err := card.ToScryfallCard()
		if err != nil {
			slog.Warn("failed to unmarshal representative card", "component", "decks", "scryfall_id", card.ScryfallID, "error", err)
			continue
		}
		out[card.OracleID] = sc
	}
	return out
}

// CreateDeckItemRequest represents a single item to add to a deck.
//
// tygo:export
type CreateDeckItemRequest struct {
	OracleID        string          `json:"oracle_id"`
	ScryfallID      string          `json:"scryfall_id"`
	Treatment       string          `json:"treatment"`
	DesiredQuantity int             `json:"desired_quantity"`
	Zone            models.DeckZone `json:"zone"`
}

// CreateDeckItemsBatchRequest represents the request body for batch adding deck
// items.
//
// tygo:export
type CreateDeckItemsBatchRequest struct {
	Items []CreateDeckItemRequest `json:"items"`
}

// CreateItemsBatch adds multiple items to a deck.
func (h *DeckHandler) CreateItemsBatch(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Verify deck exists.
	var deck models.Deck
	if err := h.db.WithContext(c.RequestCtx()).First(&deck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck", "database query failed", err)
	}

	var req CreateDeckItemsBatchRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.Items) == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "no items provided")
	}

	if len(req.Items) > MaxBatchItems {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("too many items (max %d)", MaxBatchItems))
	}

	// Validate each item up front (oracle non-empty, desired >= 1, zone valid)
	// so bad input is a 400. The model's BeforeCreate hook enforces the same
	// rules, but a hook error surfaces only as a generic insert failure (500).
	items := make([]models.DeckItem, len(req.Items))
	for i, itemReq := range req.Items {
		items[i] = models.DeckItem{
			DeckID:          uint(id),
			OracleID:        itemReq.OracleID,
			ScryfallID:      itemReq.ScryfallID,
			Treatment:       itemReq.Treatment,
			DesiredQuantity: itemReq.DesiredQuantity,
			Zone:            itemReq.Zone,
		}
		if err := items[i].ValidateDeckItem(nil); err != nil {
			return utils.ReturnError(c, fiber.StatusBadRequest,
				fmt.Sprintf("item %d: %s", i, err.Error()))
		}
	}

	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&items).Error
	})
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create deck items", "database insert failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(items)
}

// UpdateDeckItemRequest represents the request body for updating a deck item.
// All fields are optional; only provided fields are changed. ScryfallID and
// Treatment use pointers so callers can explicitly clear a pin (set to "").
//
// tygo:export
type UpdateDeckItemRequest struct {
	DesiredQuantity *int             `json:"desired_quantity,omitempty"`
	Zone            *models.DeckZone `json:"zone,omitempty"`
	ScryfallID      *string          `json:"scryfall_id,omitempty"`
	Treatment       *string          `json:"treatment,omitempty"`
}

// UpdateItem updates a deck item's quantity, zone, or printing/treatment pin.
func (h *DeckHandler) UpdateItem(c fiber.Ctx) error {
	deckID := fiber.Params[int](c, "id")
	if deckID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid deck id")
	}

	itemID := fiber.Params[int](c, "item_id")
	if itemID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid item id")
	}

	var item models.DeckItem
	if err := h.db.WithContext(c.RequestCtx()).Where("id = ? AND deck_id = ?", itemID, deckID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "deck item not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch deck item", "database query failed", err)
	}

	var req UpdateDeckItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.DesiredQuantity == nil && req.Zone == nil && req.ScryfallID == nil && req.Treatment == nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "at least one field must be provided for update")
	}

	if req.DesiredQuantity != nil {
		item.DesiredQuantity = *req.DesiredQuantity
	}
	if req.Zone != nil {
		item.Zone = *req.Zone
	}
	if req.ScryfallID != nil {
		item.ScryfallID = *req.ScryfallID
	}
	if req.Treatment != nil {
		item.Treatment = *req.Treatment
	}

	// Validate up front so bad input is a 400 rather than a 500 from the
	// BeforeUpdate hook surfacing as a generic update failure.
	if err := item.ValidateDeckItem(nil); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.db.WithContext(c.RequestCtx()).Save(&item).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update deck item", "database update failed", err)
	}

	return c.JSON(item)
}

// DeleteItem removes an item from a deck.
func (h *DeckHandler) DeleteItem(c fiber.Ctx) error {
	deckID := fiber.Params[int](c, "id")
	if deckID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid deck id")
	}

	itemID := fiber.Params[int](c, "item_id")
	if itemID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid item id")
	}

	result := h.db.WithContext(c.RequestCtx()).Where("id = ? AND deck_id = ?", itemID, deckID).Delete(&models.DeckItem{})
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete deck item", "database delete failed", result.Error)
	}

	if result.RowsAffected == 0 {
		return utils.ReturnError(c, fiber.StatusNotFound, "deck item not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
