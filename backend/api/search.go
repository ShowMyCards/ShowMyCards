package api

import (
	"backend/models"
	"backend/scryfall"
	"backend/services"
	"backend/utils"
	"log/slog"
	"strings"

	goscryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SearchHandler handles card search endpoints
type SearchHandler struct {
	client          *scryfall.Client
	db              *gorm.DB
	settingsService *services.SettingsService
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(client *scryfall.Client, db *gorm.DB, settingsService *services.SettingsService) *SearchHandler {
	return &SearchHandler{
		client:          client,
		db:              db,
		settingsService: settingsService,
	}
}

// SearchResponse wraps search results with pagination metadata
// tygo:export
type SearchResponse struct {
	Data       []EnhancedCardResult `json:"data"`
	Page       int                  `json:"page"`
	TotalCards int                  `json:"total_cards"`
	HasMore    bool                 `json:"has_more"`
}

// CardPrices represents card pricing information
// tygo:export
type CardPrices struct {
	USD       string `json:"usd,omitempty"`
	USDFoil   string `json:"usd_foil,omitempty"`
	USDEtched string `json:"usd_etched,omitempty"`
	EUR       string `json:"eur,omitempty"`
	EURFoil   string `json:"eur_foil,omitempty"`
	Tix       string `json:"tix,omitempty"`
}

// CardResult represents a card in search results
// tygo:export
type CardResult struct {
	ID              string     `json:"id"`
	OracleID        string     `json:"oracle_id"`
	Name            string     `json:"name"`
	SetCode         string     `json:"set_code,omitempty"`
	SetName         string     `json:"set_name,omitempty"`
	CollectorNumber string     `json:"collector_number,omitempty"`
	ImageURI        *string    `json:"image_uri,omitempty"`
	ColorIdentity   []string   `json:"color_identity"`
	Finishes        []string   `json:"finishes"`
	FrameEffects    []string   `json:"frame_effects,omitempty"`
	PromoTypes      []string   `json:"promo_types,omitempty"`
	EDHRECRank      *int       `json:"edhrec_rank,omitempty"`
	Prices          CardPrices `json:"prices"`
}

// CardInventoryData represents inventory information for a card
// tygo:export
type CardInventoryData struct {
	ThisPrinting   []models.Inventory `json:"this_printing"`
	OtherPrintings []models.Inventory `json:"other_printings"`
	TotalQuantity  int                `json:"total_quantity"`
}

// EnhancedCardResult represents a card with inventory information
// tygo:export
type EnhancedCardResult struct {
	CardResult
	Inventory CardInventoryData `json:"inventory"`
}

// Search searches for cards by query string
func (h *SearchHandler) Search(c fiber.Ctx) error {
	query := c.Query("q")

	if query == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "query parameter 'q' is required")
	}

	page := fiber.Query[int](c, "page", 1)
	if page < 1 {
		page = 1
	}

	// Get search settings
	defaultSearch, err := h.settingsService.Get(c.RequestCtx(), "scryfall_default_search")
	if err != nil {
		slog.Warn("failed to get scryfall_default_search setting", "component", "search", "error", err)
		defaultSearch = ""
	}

	uniqueModeStr, err := h.settingsService.Get(c.RequestCtx(), "scryfall_unique_mode")
	if err != nil {
		slog.Warn("failed to get scryfall_unique_mode setting", "component", "search", "error", err)
		uniqueModeStr = "cards"
	}

	// Append default search string to query
	if defaultSearch != "" {
		query = query + " " + defaultSearch
	}

	// Map unique mode string to scryfall.UniqueMode
	var uniqueMode goscryfall.UniqueMode
	switch strings.ToLower(uniqueModeStr) {
	case "cards":
		uniqueMode = goscryfall.UniqueModeCards
	case "art":
		uniqueMode = goscryfall.UniqueModeArt
	case "prints":
		uniqueMode = goscryfall.UniqueModePrints
	default:
		slog.Warn("unknown unique mode, defaulting to cards", "component", "search", "unique_mode", uniqueModeStr)
		uniqueMode = goscryfall.UniqueModeCards // default to cards
	}

	// Search with options
	result, err := h.client.SearchWithOptions(c.RequestCtx(), query, scryfall.SearchOptions{
		Page:       page,
		UniqueMode: uniqueMode,
	})
	if err != nil {
		return utils.HandleScryfallError(c, err, "failed to search cards")
	}

	// Collect all oracle_ids from search results
	oracleIDs := make([]string, len(result.Cards))
	for i, card := range result.Cards {
		oracleIDs[i] = card.OracleID
	}

	// Query all inventory items matching these oracle_ids
	var allInventory []models.Inventory
	if len(oracleIDs) > 0 {
		if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").
			Where("oracle_id IN ?", oracleIDs).
			Find(&allInventory).Error; err != nil {
			slog.Warn("inventory lookup failed", "component", "search", "error", err)
		}
	}

	// Build a map of oracle_id -> inventory items for quick lookup
	inventoryByOracle := make(map[string][]models.Inventory)
	for _, inv := range allInventory {
		inventoryByOracle[inv.OracleID] = append(inventoryByOracle[inv.OracleID], inv)
	}

	// Build enhanced results
	cards := make([]EnhancedCardResult, len(result.Cards))
	for i, card := range result.Cards {
		cardResult := BuildCardResult(card)

		// Split inventory into this printing vs other printings
		inventoryData := CardInventoryData{
			ThisPrinting:   []models.Inventory{},
			OtherPrintings: []models.Inventory{},
			TotalQuantity:  0,
		}

		if inventory, ok := inventoryByOracle[card.OracleID]; ok {
			for _, inv := range inventory {
				inventoryData.TotalQuantity += inv.Quantity
				if inv.ScryfallID == card.ID {
					inventoryData.ThisPrinting = append(inventoryData.ThisPrinting, inv)
				} else {
					inventoryData.OtherPrintings = append(inventoryData.OtherPrintings, inv)
				}
			}
		}

		cards[i] = EnhancedCardResult{
			CardResult: cardResult,
			Inventory:  inventoryData,
		}
	}

	response := SearchResponse{
		Data:       cards,
		Page:       result.Page,
		TotalCards: result.TotalCards,
		HasMore:    result.HasMore,
	}

	return c.JSON(response)
}

// GetCard retrieves a single card by Scryfall ID
func (h *SearchHandler) GetCard(c fiber.Ctx) error {
	cardID := c.Params("id")

	if cardID == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "card ID is required")
	}

	card, err := h.client.GetByID(c.RequestCtx(), cardID)
	if err != nil {
		return utils.HandleScryfallError(c, err, "failed to get card")
	}

	return c.JSON(card)
}

// AutocompleteResponse represents card name autocomplete suggestions
// tygo:export
type AutocompleteResponse struct {
	Suggestions []string `json:"suggestions"`
}

// Autocomplete returns card name autocomplete suggestions from Scryfall
func (h *SearchHandler) Autocomplete(c fiber.Ctx) error {
	query := c.Query("q")

	if query == "" || len(query) < 2 {
		return c.JSON(AutocompleteResponse{Suggestions: []string{}})
	}

	result, err := h.client.Autocomplete(c.RequestCtx(), query)
	if err != nil {
		slog.Warn("autocomplete failed", "component", "search", "error", err)
		return c.JSON(AutocompleteResponse{Suggestions: []string{}})
	}

	// Limit to 5 suggestions
	suggestions := result
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return c.JSON(AutocompleteResponse{Suggestions: suggestions})
}

// fiber:context-methods migrated
