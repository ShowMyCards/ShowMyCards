package api

import (
	"backend/models"
	"backend/utils"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DeckHandler handles deck endpoints.
type DeckHandler struct {
	db *gorm.DB
}

// NewDeckHandler creates a new deck handler.
func NewDeckHandler(db *gorm.DB) *DeckHandler {
	return &DeckHandler{db: db}
}

// DeckSummary represents a deck with summary statistics.
//
// Milestone 1a ships counts only. The aggregate shortfall (cards short across
// the deck, computed against the user's inventory) is added in 1b once the
// allocation service lands — see FR98/IMPLEMENTATION_PLAN.md §2.
//
// tygo:export
type DeckSummary struct {
	ID               uint   `json:"id"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	TotalItems       int    `json:"total_items"`
	TotalCardsDemand int    `json:"total_cards_demand"`
}

// List returns all decks with summary statistics.
func (h *DeckHandler) List(c fiber.Ctx) error {
	var decks []models.Deck
	if err := h.db.WithContext(c.RequestCtx()).Preload("Items").Order("created_at DESC").Find(&decks).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch decks", "database query failed", err)
	}

	summaries := make([]DeckSummary, len(decks))
	for i, deck := range decks {
		totalDemand := 0
		for _, item := range deck.Items {
			if item.Zone.CountsAsDemand() {
				totalDemand += item.DesiredQuantity
			}
		}

		summaries[i] = DeckSummary{
			ID:               deck.ID,
			CreatedAt:        deck.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        deck.UpdatedAt.Format(time.RFC3339),
			Name:             deck.Name,
			Description:      deck.Description,
			TotalItems:       len(deck.Items),
			TotalCardsDemand: totalDemand,
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
// The availability fields are populated by the allocation service in
// Milestone 1b. In 1a the items endpoint is a stub that returns an empty
// response, so these fields are documented here for the type contract only.
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
// In 1a all zone slices are empty and AggregateShortfall is zero — the
// endpoint is a contract stub. 1b will populate it from the allocation
// service.
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

// ListItems returns the items in a deck, grouped by zone, with availability.
//
// TODO(FR98 Issue 1b): this endpoint is currently a stub. It verifies that
// the deck exists and returns an empty grouped response so the frontend can
// integrate against the real URL. The allocation-service-backed
// implementation lands in Milestone 1b.
func (h *DeckHandler) ListItems(c fiber.Ctx) error {
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

	return c.JSON(DeckItemsResponse{
		DeckID:             deck.ID,
		Command:            []EnrichedDeckItem{},
		Main:               []EnrichedDeckItem{},
		Side:               []EnrichedDeckItem{},
		Maybe:              []EnrichedDeckItem{},
		AggregateShortfall: 0,
	})
}
