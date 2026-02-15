package api

import (
	"backend/models"
	"backend/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ListHandler handles list endpoints
type ListHandler struct {
	db *gorm.DB
}

// NewListHandler creates a new list handler
func NewListHandler(db *gorm.DB) *ListHandler {
	return &ListHandler{db: db}
}

// ListSummary represents a list with summary statistics
// tygo:export
type ListSummary struct {
	ID                   uint   `json:"id"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	TotalItems           int    `json:"total_items"`
	TotalCardsWanted     int    `json:"total_cards_wanted"`
	TotalCardsCollected  int    `json:"total_cards_collected"`
	CompletionPercentage int    `json:"completion_percentage"`
}

// List returns all lists with summary statistics
func (h *ListHandler) List(c fiber.Ctx) error {
	// Get all lists with their items in a single query
	var lists []models.List
	if err := h.db.WithContext(c.RequestCtx()).Preload("Items").Order("created_at DESC").Find(&lists).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch lists", "database query failed", err)
	}

	// Build summary for each list using preloaded items
	summaries := make([]ListSummary, len(lists))
	for i, list := range lists {
		totalWanted := 0
		totalCollected := 0
		for _, item := range list.Items {
			totalWanted += item.DesiredQuantity
			totalCollected += item.CollectedQuantity
		}

		completionPercentage := 0
		if totalWanted > 0 {
			completionPercentage = (totalCollected * 100) / totalWanted
		}

		summaries[i] = ListSummary{
			ID:                   list.ID,
			CreatedAt:            list.CreatedAt.Format(time.RFC3339),
			UpdatedAt:            list.UpdatedAt.Format(time.RFC3339),
			Name:                 list.Name,
			Description:          list.Description,
			TotalItems:           len(list.Items),
			TotalCardsWanted:     totalWanted,
			TotalCardsCollected:  totalCollected,
			CompletionPercentage: completionPercentage,
		}
	}

	return c.JSON(summaries)
}

// Get returns a single list by ID
func (h *ListHandler) Get(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var list models.List
	if err := h.db.WithContext(c.RequestCtx()).First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list", "database query failed", err)
	}

	return c.JSON(list)
}

// CreateListRequest represents the request body for creating a list
// tygo:export
type CreateListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create creates a new list
func (h *ListHandler) Create(c fiber.Ctx) error {
	var req CreateListRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	var validationErrors []error
	validationErrors = append(validationErrors, utils.ValidateRequired(req.Name, "name"))
	validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Name, 255, "name"))
	validationErrors = append(validationErrors, utils.ValidateMaxLength(req.Description, 1000, "description"))

	if err := utils.CombineErrors(validationErrors); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	list := models.List{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.db.WithContext(c.RequestCtx()).Create(&list).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create list", "database insert failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(list)
}

// UpdateListRequest represents the request body for updating a list
// tygo:export
type UpdateListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Update updates an existing list
func (h *ListHandler) Update(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var list models.List
	if err := h.db.WithContext(c.RequestCtx()).First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list", "database query failed", err)
	}

	var req UpdateListRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name != "" {
		list.Name = req.Name
	}
	// Allow empty description to clear it
	list.Description = req.Description

	if err := h.db.WithContext(c.RequestCtx()).Save(&list).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update list", "database update failed", err)
	}

	return c.JSON(list)
}

// Delete deletes a list and all its items
func (h *ListHandler) Delete(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Use transaction to ensure atomicity
	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		// Verify list exists
		var list models.List
		if err := tx.First(&list, id).Error; err != nil {
			return err
		}

		// Delete list items first (explicit, not relying on CASCADE)
		if err := tx.Where("list_id = ?", id).Delete(&models.ListItem{}).Error; err != nil {
			return err
		}

		// Delete list
		if err := tx.Delete(&list).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete list", "database delete failed", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// EnrichedListItem represents a list item with card data from Scryfall
// tygo:export
type EnrichedListItem struct {
	ID                uint   `json:"id"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	ListID            uint   `json:"list_id"`
	ScryfallID        string `json:"scryfall_id"`
	OracleID          string `json:"oracle_id"`
	Treatment         string `json:"treatment"`
	DesiredQuantity   int    `json:"desired_quantity"`
	CollectedQuantity int    `json:"collected_quantity"`
	// Enriched fields (populated from Scryfall API)
	Name            string   `json:"name,omitempty"`
	SetName         string   `json:"set_name,omitempty"`
	SetCode         string   `json:"set_code,omitempty"`
	CollectorNumber string   `json:"collector_number,omitempty"`
	Rarity          string   `json:"rarity,omitempty"`
	CurrentPrice    float64  `json:"current_price,omitempty"`
	Finishes        []string `json:"finishes,omitempty"`
	FrameEffects    []string `json:"frame_effects,omitempty"`
	PromoTypes      []string `json:"promo_types,omitempty"`
}

// ListItemsResponse represents paginated list items with aggregate stats
// tygo:export
type ListItemsResponse struct {
	Data                []EnrichedListItem `json:"data"`
	Page                int                `json:"page"`
	PageSize            int                `json:"page_size"`
	TotalItems          int64              `json:"total_items"`
	TotalPages          int                `json:"total_pages"`
	TotalWanted         int                `json:"total_wanted"`
	TotalCollected      int                `json:"total_collected"`
	CompletionPercent   int                `json:"completion_percent"`
	TotalCollectedValue float64            `json:"total_collected_value"`
	TotalRemainingValue float64            `json:"total_remaining_value"`
}

// ListItems returns all items for a list with pagination and enriched card data.
//
// This endpoint performs several operations:
// 1. Validates the list exists
// 2. Calculates aggregate statistics (total wanted/collected across ALL items)
// 3. Calculates total value (collected and remaining) by fetching price data from Scryfall
// 4. Returns paginated list items enriched with card details (name, set, rarity, price)
//
// Performance notes:
// - Aggregate stats are calculated across all items (not just current page)
// - Value calculations require fetching ALL list items and their card data
// - Paginated items are fetched separately and enriched with card metadata
func (h *ListHandler) ListItems(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Verify list exists
	var list models.List
	if err := h.db.WithContext(c.RequestCtx()).First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list", "database query failed", err)
	}

	ctx := c.RequestCtx()
	params := utils.ParsePaginationParams(c, DefaultCardsPageSize, MaxCardsPageSize)
	listID := uint(id)

	// Count total items
	var total int64
	if err := h.db.WithContext(ctx).Model(&models.ListItem{}).Where("list_id = ?", listID).Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count list items", "count query failed", err)
	}

	// Calculate aggregate stats, value totals, and enriched items
	stats, completionPercent, err := h.calculateListStats(ctx, listID)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to calculate stats", "aggregate query failed", err)
	}

	collectedValue, remainingValue := h.calculateListValue(ctx, listID)

	enrichedItems, err := h.enrichListItems(ctx, listID, params.Page, params.PageSize)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list items", "database query failed", err)
	}

	return c.JSON(ListItemsResponse{
		Data:                enrichedItems,
		Page:                params.Page,
		PageSize:            params.PageSize,
		TotalItems:          total,
		TotalPages:          utils.CalculateTotalPages(total, params.PageSize),
		TotalWanted:         stats.TotalWanted,
		TotalCollected:      stats.TotalCollected,
		CompletionPercent:   completionPercent,
		TotalCollectedValue: collectedValue,
		TotalRemainingValue: remainingValue,
	})
}

// listAggregateStats holds aggregate quantity stats for a list.
type listAggregateStats struct {
	TotalWanted    int
	TotalCollected int
}

// calculateListStats computes aggregate wanted/collected stats across all items in a list.
func (h *ListHandler) calculateListStats(ctx context.Context, listID uint) (listAggregateStats, int, error) {
	var stats listAggregateStats
	if err := h.db.WithContext(ctx).Model(&models.ListItem{}).
		Where("list_id = ?", listID).
		Select("COALESCE(SUM(desired_quantity), 0) as total_wanted, COALESCE(SUM(collected_quantity), 0) as total_collected").
		Scan(&stats).Error; err != nil {
		return stats, 0, err
	}

	completionPercent := 0
	if stats.TotalWanted > 0 {
		completionPercent = (stats.TotalCollected * 100) / stats.TotalWanted
	}
	return stats, completionPercent, nil
}

// calculateListValue computes the total collected and remaining USD values for a list.
func (h *ListHandler) calculateListValue(ctx context.Context, listID uint) (collectedValue, remainingValue float64) {
	var allListItems []models.ListItem
	if err := h.db.WithContext(ctx).Where("list_id = ?", listID).Find(&allListItems).Error; err != nil {
		slog.Warn("failed to fetch list items for value calculation", "component", "lists", "list_id", listID, "error", err)
		return 0, 0
	}

	allScryfallIDs := make([]string, len(allListItems))
	for i, item := range allListItems {
		allScryfallIDs[i] = item.ScryfallID
	}

	if len(allScryfallIDs) == 0 {
		return 0, 0
	}

	var allCards []models.Card
	if err := h.db.WithContext(ctx).Where("scryfall_id IN ?", allScryfallIDs).Find(&allCards).Error; err != nil {
		slog.Warn("failed to fetch cards for value calculation", "component", "lists", "list_id", listID, "error", err)
		return 0, 0
	}

	allCardMap := make(map[string]models.Card, len(allCards))
	for _, card := range allCards {
		allCardMap[card.ScryfallID] = card
	}

	for _, item := range allListItems {
		card, ok := allCardMap[item.ScryfallID]
		if !ok {
			continue
		}
		scryfallCard, err := card.ToScryfallCard()
		if err != nil {
			continue
		}
		price := utils.ParsePriceFromScryfall(scryfallCard.Prices, item.Treatment)
		collectedValue += price * float64(item.CollectedQuantity)
		remaining := item.DesiredQuantity - item.CollectedQuantity
		if remaining > 0 {
			remainingValue += price * float64(remaining)
		}
	}
	return collectedValue, remainingValue
}

// enrichListItems fetches a page of list items and enriches them with card metadata.
func (h *ListHandler) enrichListItems(ctx context.Context, listID uint, page, pageSize int) ([]EnrichedListItem, error) {
	var items []models.ListItem
	offset := utils.CalculateOffset(page, pageSize)

	if err := h.db.WithContext(ctx).
		Where("list_id = ?", listID).
		Order("created_at ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	// Bulk fetch card data for this page's items
	scryfallIDs := make([]string, len(items))
	for i, item := range items {
		scryfallIDs[i] = item.ScryfallID
	}

	scryfallCardMap, err := models.GetScryfallCardsByIDs(h.db.WithContext(ctx), scryfallIDs)
	if err != nil {
		slog.Warn("failed to fetch card data for enrichment", "component", "lists", "error", err)
	}

	enrichedItems := make([]EnrichedListItem, len(items))
	for i, item := range items {
		enrichedItem := EnrichedListItem{
			ID:                item.ID,
			CreatedAt:         item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         item.UpdatedAt.Format(time.RFC3339),
			ListID:            item.ListID,
			ScryfallID:        item.ScryfallID,
			OracleID:          item.OracleID,
			Treatment:         item.Treatment,
			DesiredQuantity:   item.DesiredQuantity,
			CollectedQuantity: item.CollectedQuantity,
		}

		if scryfallCard, ok := scryfallCardMap[item.ScryfallID]; ok {
			enrichedItem.Name = scryfallCard.Name
			enrichedItem.SetName = scryfallCard.SetName
			enrichedItem.SetCode = scryfallCard.Set
			enrichedItem.CollectorNumber = scryfallCard.CollectorNumber
			enrichedItem.Rarity = string(scryfallCard.Rarity)
			enrichedItem.CurrentPrice = utils.ParsePriceFromScryfall(scryfallCard.Prices, item.Treatment)
			enrichedItem.Finishes = utils.ConvertEnumSliceToStrings(scryfallCard.Finishes)
			enrichedItem.FrameEffects = utils.ConvertEnumSliceToStrings(scryfallCard.FrameEffects)
			enrichedItem.PromoTypes = scryfallCard.PromoTypes
		}

		enrichedItems[i] = enrichedItem
	}
	return enrichedItems, nil
}

// CreateListItemRequest represents a single item to add to a list
// tygo:export
type CreateListItemRequest struct {
	ScryfallID      string `json:"scryfall_id"`
	OracleID        string `json:"oracle_id"`
	Treatment       string `json:"treatment"`
	DesiredQuantity int    `json:"desired_quantity"`
}

// CreateItemsBatchRequest represents the request body for batch adding items
// tygo:export
type CreateItemsBatchRequest struct {
	Items []CreateListItemRequest `json:"items"`
}

// CreateItemsBatch adds multiple items to a list
func (h *ListHandler) CreateItemsBatch(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Verify list exists
	var list models.List
	if err := h.db.WithContext(c.RequestCtx()).First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list", "database query failed", err)
	}

	var req CreateItemsBatchRequest
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

	// Create items in a transaction for atomicity
	items := make([]models.ListItem, len(req.Items))
	for i, itemReq := range req.Items {
		items[i] = models.ListItem{
			ListID:            uint(id),
			ScryfallID:        itemReq.ScryfallID,
			OracleID:          itemReq.OracleID,
			Treatment:         itemReq.Treatment,
			DesiredQuantity:   itemReq.DesiredQuantity,
			CollectedQuantity: 0,
		}
	}

	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&items).Error
	})
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create list items", "database insert failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(items)
}

// UpdateListItemRequest represents the request body for updating a list item
// tygo:export
type UpdateListItemRequest struct {
	DesiredQuantity   *int `json:"desired_quantity,omitempty"`
	CollectedQuantity *int `json:"collected_quantity,omitempty"`
}

// UpdateItem updates a list item (primarily for updating collected quantity)
func (h *ListHandler) UpdateItem(c fiber.Ctx) error {
	listID := fiber.Params[int](c, "id")
	if listID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID := fiber.Params[int](c, "item_id")
	if itemID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid item id")
	}

	var item models.ListItem
	if err := h.db.WithContext(c.RequestCtx()).Where("id = ? AND list_id = ?", itemID, listID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "list item not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list item", "database query failed", err)
	}

	var req UpdateListItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.DesiredQuantity == nil && req.CollectedQuantity == nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "at least one field must be provided for update")
	}

	if req.DesiredQuantity != nil {
		item.DesiredQuantity = *req.DesiredQuantity
	}
	if req.CollectedQuantity != nil {
		item.CollectedQuantity = *req.CollectedQuantity
	}

	if err := h.db.WithContext(c.RequestCtx()).Save(&item).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update list item", "database update failed", err)
	}

	return c.JSON(item)
}

// DeleteItem removes an item from a list
func (h *ListHandler) DeleteItem(c fiber.Ctx) error {
	listID := fiber.Params[int](c, "id")
	if listID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID := fiber.Params[int](c, "item_id")
	if itemID == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid item id")
	}

	result := h.db.WithContext(c.RequestCtx()).Where("id = ? AND list_id = ?", itemID, listID).Delete(&models.ListItem{})
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete list item", "database delete failed", result.Error)
	}

	if result.RowsAffected == 0 {
		return utils.ReturnError(c, fiber.StatusNotFound, "list item not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// fiber:context-methods migrated
