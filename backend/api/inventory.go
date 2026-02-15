package api

import (
	"backend/models"
	"backend/rules"
	"backend/services"
	"backend/utils"
	"errors"
	"fmt"
	"log/slog"

	scryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// InventoryHandler handles inventory endpoints
type InventoryHandler struct {
	db          *gorm.DB
	autoSortSvc *services.AutoSortService
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(db *gorm.DB, autoSortSvc *services.AutoSortService) *InventoryHandler {
	return &InventoryHandler{
		db:          db,
		autoSortSvc: autoSortSvc,
	}
}

// List returns inventory items with pagination
func (h *InventoryHandler) List(c fiber.Ctx) error {
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, utils.MaxPageSize)

	// Optional filters
	scryfallID := c.Query("scryfall_id")
	storageLocationID := c.Query("storage_location_id")

	query := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{})

	if scryfallID != "" {
		query = query.Where("scryfall_id = ?", scryfallID)
	}

	if storageLocationID != "" {
		if storageLocationID == "null" {
			query = query.Where("storage_location_id IS NULL")
		} else {
			if err := utils.ValidateNumericParam(storageLocationID, "storage_location_id"); err != nil {
				return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
			}
			query = query.Where("storage_location_id = ?", storageLocationID)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count inventory items", "count query failed", err)
	}

	var items []models.Inventory
	offset := utils.CalculateOffset(params.Page, params.PageSize)
	if err := query.Preload("StorageLocation").
		Offset(offset).
		Limit(params.PageSize).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory items", "database query failed", err)
	}

	response := utils.NewPaginatedResponse(items, params.Page, params.PageSize, total)
	return c.JSON(response)
}

// Get returns a single inventory item by ID
func (h *InventoryHandler) Get(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var item models.Inventory
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "inventory item not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory item", "database query failed", err)
	}
	return c.JSON(item)
}

// CreateInventoryRequest represents the request body for creating an inventory item
type CreateInventoryRequest struct {
	ScryfallID        string `json:"scryfall_id"`
	OracleID          string `json:"oracle_id"`
	Treatment         string `json:"treatment,omitempty"`
	Quantity          int    `json:"quantity"`
	StorageLocationID *uint  `json:"storage_location_id,omitempty"`
}

// Create creates a new inventory item
func (h *InventoryHandler) Create(c fiber.Ctx) error {
	var req CreateInventoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	var validationErrors []error
	validationErrors = append(validationErrors, utils.ValidateRequired(req.ScryfallID, "scryfall_id"))
	validationErrors = append(validationErrors, utils.ValidateRequired(req.OracleID, "oracle_id"))
	validationErrors = append(validationErrors, utils.ValidateNonNegative(req.Quantity, "quantity"))

	if err := utils.CombineErrors(validationErrors); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	// Set default quantity if not provided
	if req.Quantity == 0 {
		req.Quantity = 1
	}

	// Validate storage location exists if provided
	if req.StorageLocationID != nil {
		var location models.StorageLocation
		if err := h.db.WithContext(c.RequestCtx()).First(&location, *req.StorageLocationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ReturnError(c, fiber.StatusBadRequest, "storage location not found")
			}
			return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
				"Failed to validate storage location", "storage location lookup failed", err)
		}
	} else {
		// If no storage location provided, automatically evaluate sorting rules
		slog.Info("evaluating sorting rules", "component", "inventory", "scryfall_id", req.ScryfallID)

		locationID, err := h.autoSortSvc.DetermineStorageLocation(c.RequestCtx(), req.ScryfallID, req.Treatment)
		if err != nil {
			slog.Debug("auto-sort did not assign location", "component", "inventory", "scryfall_id", req.ScryfallID, "error", err)
		} else {
			req.StorageLocationID = locationID
		}
	}

	item := models.Inventory{
		ScryfallID:        req.ScryfallID,
		OracleID:          req.OracleID,
		Treatment:         req.Treatment,
		Quantity:          req.Quantity,
		StorageLocationID: req.StorageLocationID,
	}

	if err := h.db.WithContext(c.RequestCtx()).Create(&item).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create inventory item", "database insert failed", err)
	}

	// Reload with storage location
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&item, item.ID).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to reload inventory item", "database query failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateInventoryRequest represents the request body for updating an inventory item
type UpdateInventoryRequest struct {
	ScryfallID        *string `json:"scryfall_id,omitempty"`
	OracleID          *string `json:"oracle_id,omitempty"`
	Treatment         *string `json:"treatment,omitempty"`
	Quantity          *int    `json:"quantity,omitempty"`
	StorageLocationID *uint   `json:"storage_location_id,omitempty"`
	ClearStorage      bool    `json:"clear_storage,omitempty"`
}

// Update updates an existing inventory item
func (h *InventoryHandler) Update(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var item models.Inventory
	if err := h.db.WithContext(c.RequestCtx()).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "inventory item not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory item", "database query failed", err)
	}

	var req UpdateInventoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.ScryfallID == nil && req.OracleID == nil && req.Treatment == nil &&
		req.Quantity == nil && req.StorageLocationID == nil && !req.ClearStorage {
		return utils.ReturnError(c, fiber.StatusBadRequest, "at least one field must be provided for update")
	}

	// Update fields if provided
	if req.ScryfallID != nil {
		item.ScryfallID = *req.ScryfallID
	}
	if req.OracleID != nil {
		item.OracleID = *req.OracleID
	}
	if req.Treatment != nil {
		item.Treatment = *req.Treatment
	}
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	}

	// Handle storage location updates
	if req.ClearStorage {
		item.StorageLocationID = nil
	} else if req.StorageLocationID != nil {
		// Validate storage location exists
		var location models.StorageLocation
		if err := h.db.WithContext(c.RequestCtx()).First(&location, *req.StorageLocationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ReturnError(c, fiber.StatusBadRequest, "storage location not found")
			}
			return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
				"Failed to validate storage location", "storage location lookup failed", err)
		}
		item.StorageLocationID = req.StorageLocationID
	}

	if err := h.db.WithContext(c.RequestCtx()).Save(&item).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update inventory item", "database update failed", err)
	}

	// Reload with storage location
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&item, item.ID).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to reload inventory item", "database query failed", err)
	}

	return c.JSON(item)
}

// Delete deletes an inventory item
func (h *InventoryHandler) Delete(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	result := h.db.WithContext(c.RequestCtx()).Delete(&models.Inventory{}, id)
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete inventory item", "database delete failed", result.Error)
	}

	if result.RowsAffected == 0 {
		return utils.ReturnError(c, fiber.StatusNotFound, "inventory item not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// InventoryCardsResponse represents paginated card results with inventory data
// tygo:export
type InventoryCardsResponse struct {
	Data       []EnhancedCardResult `json:"data"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalCards int                  `json:"total_cards"`
	TotalPages int                  `json:"total_pages"`
}

// buildEnhancedCardResult creates an EnhancedCardResult from a Scryfall card and inventory items.
// It extracts prices, images, and converts enum types to strings for JSON serialization.
func buildEnhancedCardResult(scryfallCard scryfall.Card, inventoryItems []models.Inventory) EnhancedCardResult {
	inventoryData := CardInventoryData{
		ThisPrinting:   inventoryItems,
		OtherPrintings: []models.Inventory{},
		TotalQuantity:  0,
	}
	for _, inv := range inventoryItems {
		inventoryData.TotalQuantity += inv.Quantity
	}

	cardResult := BuildCardResult(scryfallCard)
	return EnhancedCardResult{
		CardResult: cardResult,
		Inventory:  inventoryData,
	}
}

// ListAsCards returns inventory items as enhanced card results (like search)
func (h *InventoryHandler) ListAsCards(c fiber.Ctx) error {
	// Parse query params (using smaller max page size for card results)
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, DefaultCardsPageSize)

	locationID := c.Query("storage_location_id")

	// Build query
	query := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{})
	if locationID == "null" {
		query = query.Where("storage_location_id IS NULL")
	} else if locationID != "" {
		if err := utils.ValidateNumericParam(locationID, "storage_location_id"); err != nil {
			return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
		}
		query = query.Where("storage_location_id = ?", locationID)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count inventory items", "count query failed", err)
	}

	// Get paginated inventory items
	var inventoryItems []models.Inventory
	offset := utils.CalculateOffset(params.Page, params.PageSize)
	if err := query.
		Preload("StorageLocation").
		Order("created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&inventoryItems).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory items", "database query failed", err)
	}

	// Group by Scryfall ID to fetch card data
	scryfallIDs := make([]string, 0)
	inventoryMap := make(map[string][]models.Inventory)

	for _, item := range inventoryItems {
		if _, exists := inventoryMap[item.ScryfallID]; !exists {
			scryfallIDs = append(scryfallIDs, item.ScryfallID)
		}
		inventoryMap[item.ScryfallID] = append(inventoryMap[item.ScryfallID], item)
	}

	// Fetch and parse card data
	scryfallCardMap, err := models.GetScryfallCardsByIDs(h.db.WithContext(c.RequestCtx()), scryfallIDs)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch card data", "cards query failed", err)
	}

	// Build enhanced card results using card data
	enhancedResults := make([]EnhancedCardResult, 0, len(scryfallIDs))
	for _, scryfallID := range scryfallIDs {
		scryfallCard, found := scryfallCardMap[scryfallID]
		if !found {
			continue
		}

		enhancedCard := buildEnhancedCardResult(scryfallCard, inventoryMap[scryfallID])
		enhancedResults = append(enhancedResults, enhancedCard)
	}

	// Calculate total pages
	totalPages := utils.CalculateTotalPages(total, params.PageSize)

	return c.JSON(InventoryCardsResponse{
		Data:       enhancedResults,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalCards: int(total),
		TotalPages: totalPages,
	})
}

// ExistingPrintingInfo represents info about an existing printing in inventory
// tygo:export
type ExistingPrintingInfo struct {
	ScryfallID      string                  `json:"scryfall_id"`
	Treatment       string                  `json:"treatment"`
	Quantity        int                     `json:"quantity"`
	StorageLocation *models.StorageLocation `json:"storage_location,omitempty"`
}

// ByOracleResponse represents the response for checking existing printings
// tygo:export
type ByOracleResponse struct {
	OracleID  string                           `json:"oracle_id"`
	Printings []ExistingPrintingInfo           `json:"printings"`
	Locations []models.StorageLocation         `json:"locations"` // Unique locations where this card exists
}

// ByOracle returns inventory items for a given oracle ID
func (h *InventoryHandler) ByOracle(c fiber.Ctx) error {
	oracleID := c.Params("oracle_id")
	if oracleID == "" {
		return utils.ReturnError(c, fiber.StatusBadRequest, "oracle_id is required")
	}

	var items []models.Inventory
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").
		Where("oracle_id = ?", oracleID).
		Find(&items).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory items", "database query failed", err)
	}

	// Build response with printings and unique locations
	printings := make([]ExistingPrintingInfo, 0, len(items))
	locationMap := make(map[uint]models.StorageLocation)

	for _, item := range items {
		printings = append(printings, ExistingPrintingInfo{
			ScryfallID:      item.ScryfallID,
			Treatment:       item.Treatment,
			Quantity:        item.Quantity,
			StorageLocation: item.StorageLocation,
		})

		if item.StorageLocation != nil {
			locationMap[item.StorageLocation.ID] = *item.StorageLocation
		}
	}

	// Convert location map to slice
	locations := make([]models.StorageLocation, 0, len(locationMap))
	for _, loc := range locationMap {
		locations = append(locations, loc)
	}

	return c.JSON(ByOracleResponse{
		OracleID:  oracleID,
		Printings: printings,
		Locations: locations,
	})
}

// GetUnassignedCount returns the count of inventory items without a storage location
func (h *InventoryHandler) GetUnassignedCount(c fiber.Ctx) error {
	var count int64
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{}).
		Where("storage_location_id IS NULL").
		Count(&count).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count unassigned inventory items", "count query failed", err)
	}

	return c.JSON(fiber.Map{"count": count})
}

// BatchMoveRequest represents the request body for moving multiple inventory items
// tygo:export
type BatchMoveRequest struct {
	IDs               []uint `json:"ids"`
	StorageLocationID *uint  `json:"storage_location_id"`
}

// BatchMoveResponse represents the response for batch move operations
// tygo:export
type BatchMoveResponse struct {
	Updated int `json:"updated"`
}

// BatchMove moves multiple inventory items to a new storage location
func (h *InventoryHandler) BatchMove(c fiber.Ctx) error {
	var req BatchMoveRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.IDs) == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "ids array is required")
	}

	if len(req.IDs) > MaxBatchIDs {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("too many ids (max %d)", MaxBatchIDs))
	}

	// Validate storage location exists if provided
	if req.StorageLocationID != nil {
		var location models.StorageLocation
		if err := h.db.WithContext(c.RequestCtx()).First(&location, *req.StorageLocationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ReturnError(c, fiber.StatusBadRequest, "storage location not found")
			}
			return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
				"Failed to validate storage location", "storage location lookup failed", err)
		}
	}

	// Update all items in a single query
	// Use UpdateColumn to skip BeforeUpdate hooks — this is a targeted column update
	// that doesn't need full model validation (ScryfallID, OracleID, etc.)
	result := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{}).
		Where("id IN ?", req.IDs).
		UpdateColumn("storage_location_id", req.StorageLocationID)

	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to move inventory items", "database update failed", result.Error)
	}

	slog.Info("batch moved items", "component", "inventory", "count", result.RowsAffected, "storage_location_id", req.StorageLocationID)

	return c.JSON(BatchMoveResponse{Updated: int(result.RowsAffected)})
}

// BatchDeleteRequest represents the request body for deleting multiple inventory items
// tygo:export
type BatchDeleteRequest struct {
	IDs []uint `json:"ids"`
}

// BatchDeleteResponse represents the response for batch delete operations
// tygo:export
type BatchDeleteResponse struct {
	Deleted int `json:"deleted"`
}

// BatchDelete deletes multiple inventory items
func (h *InventoryHandler) BatchDelete(c fiber.Ctx) error {
	var req BatchDeleteRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.IDs) == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "ids array is required")
	}

	if len(req.IDs) > MaxBatchIDs {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("too many ids (max %d)", MaxBatchIDs))
	}

	result := h.db.WithContext(c.RequestCtx()).Delete(&models.Inventory{}, req.IDs)
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete inventory items", "database delete failed", result.Error)
	}

	slog.Info("batch deleted items", "component", "inventory", "count", result.RowsAffected)

	return c.JSON(BatchDeleteResponse{Deleted: int(result.RowsAffected)})
}

// ResortRequest represents the request body for re-sorting inventory items
// tygo:export
type ResortRequest struct {
	IDs []uint `json:"ids,omitempty"` // If empty, resort all items
}

// ResortMovement represents a single card movement during resort
// tygo:export
type ResortMovement struct {
	CardName     string  `json:"card_name"`
	Treatment    string  `json:"treatment"`
	FromLocation *string `json:"from_location"` // nil means unassigned
	ToLocation   *string `json:"to_location"`   // nil means unassigned
}

// ResortResponse represents the response for resort operations
// tygo:export
type ResortResponse struct {
	Processed int               `json:"processed"`
	Updated   int               `json:"updated"`
	Errors    int               `json:"errors"`
	Movements []ResortMovement  `json:"movements,omitempty"`
}

// resortEvalResult holds the evaluation results for batch updating after resort
type resortEvalResult struct {
	processed int
	errors    int
	movements []ResortMovement
	clearIDs  []uint            // items to unassign
	moveMap   map[uint][]uint   // locationID -> []itemID
}

// evaluateResortItems evaluates sorting rules against each inventory item and
// determines which items need to be moved or unassigned.
func evaluateResortItems(items []models.Inventory, cardMap map[string]models.Card, sortingRules []models.SortingRule, evaluator *rules.Evaluator) resortEvalResult {
	result := resortEvalResult{
		movements: make([]ResortMovement, 0),
		clearIDs:  make([]uint, 0),
		moveMap:   make(map[uint][]uint),
	}

	for _, item := range items {
		result.processed++

		card, found := cardMap[item.ScryfallID]
		if !found {
			slog.Warn("card not found in cards table", "component", "resort", "scryfall_id", item.ScryfallID)
			result.errors++
			continue
		}

		cardData, err := rules.RawJSONToRuleData(card.RawJSON, item.Treatment)
		if err != nil {
			slog.Error("error converting card", "component", "resort", "scryfall_id", item.ScryfallID, "error", err)
			result.errors++
			continue
		}

		cardName := ""
		if name, ok := cardData["name"].(string); ok {
			cardName = name
		}

		var fromLocation *string
		if item.StorageLocation != nil {
			fromLocation = &item.StorageLocation.Name
		}

		location, err := evaluator.EvaluateCardWithRules(cardData, sortingRules)
		if err != nil {
			// No matching rule — clear storage location if currently assigned
			if item.StorageLocationID != nil {
				result.clearIDs = append(result.clearIDs, item.ID)
				result.movements = append(result.movements, ResortMovement{
					CardName:     cardName,
					Treatment:    item.Treatment,
					FromLocation: fromLocation,
					ToLocation:   nil,
				})
			}
			continue
		}

		// Check if location changed
		if item.StorageLocationID == nil || *item.StorageLocationID != location.ID {
			result.moveMap[location.ID] = append(result.moveMap[location.ID], item.ID)
			result.movements = append(result.movements, ResortMovement{
				CardName:     cardName,
				Treatment:    item.Treatment,
				FromLocation: fromLocation,
				ToLocation:   &location.Name,
			})
		}
	}

	return result
}

// executeResortUpdates applies the resort evaluation results to the database in a single transaction.
func executeResortUpdates(db *gorm.DB, eval resortEvalResult) (int, error) {
	updated := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if len(eval.clearIDs) > 0 {
			result := tx.Model(&models.Inventory{}).
				Where("id IN ?", eval.clearIDs).
				UpdateColumn("storage_location_id", nil)
			if result.Error != nil {
				return result.Error
			}
			updated += int(result.RowsAffected)
		}

		for locID, ids := range eval.moveMap {
			result := tx.Model(&models.Inventory{}).
				Where("id IN ?", ids).
				UpdateColumn("storage_location_id", locID)
			if result.Error != nil {
				return result.Error
			}
			updated += int(result.RowsAffected)
		}
		return nil
	})
	return updated, err
}

// Resort re-evaluates inventory items against sorting rules
func (h *InventoryHandler) Resort(c fiber.Ctx) error {
	var req ResortRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Build query for items to process (with current storage location preloaded)
	query := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation")
	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	// Fetch all items to process
	var items []models.Inventory
	if err := query.Find(&items).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch inventory items", "database query failed", err)
	}

	if len(items) == 0 {
		return c.JSON(ResortResponse{Processed: 0, Updated: 0, Errors: 0, Movements: []ResortMovement{}})
	}

	// Get unique scryfall IDs to fetch card data
	scryfallIDs := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range items {
		if !seen[item.ScryfallID] {
			scryfallIDs = append(scryfallIDs, item.ScryfallID)
			seen[item.ScryfallID] = true
		}
	}

	// Batch fetch all card data
	cardMap, err := models.GetCardsByIDs(h.db.WithContext(c.RequestCtx()), scryfallIDs)
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch card data", "cards query failed", err)
	}

	// Pre-fetch sorting rules once for the entire batch
	var sortingRules []models.SortingRule
	if err := h.db.WithContext(c.RequestCtx()).Where("enabled = ?", true).
		Order("priority ASC").
		Preload("StorageLocation").
		Find(&sortingRules).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch sorting rules", "rules query failed", err)
	}

	// Evaluate each item against sorting rules
	evaluator := rules.NewEvaluator(h.db)
	eval := evaluateResortItems(items, cardMap, sortingRules, evaluator)

	// Execute batch updates in a transaction
	updated, txErr := executeResortUpdates(h.db.WithContext(c.RequestCtx()), eval)
	if txErr != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update inventory locations", "resort transaction failed", txErr)
	}

	slog.Info("resort completed", "component", "resort", "processed", eval.processed, "updated", updated, "errors", eval.errors)

	return c.JSON(ResortResponse{
		Processed: eval.processed,
		Updated:   updated,
		Errors:    eval.errors,
		Movements: eval.movements,
	})
}
