// Package api provides HTTP handlers for the ShowMyCards REST API.
// It contains handlers for inventory, storage, lists, sorting rules, and other endpoints.
package api

import (
	"backend/models"
	"backend/utils"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// StorageHandler handles storage location endpoints
type StorageHandler struct {
	db *gorm.DB
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(db *gorm.DB) *StorageHandler {
	return &StorageHandler{db: db}
}

// List returns storage locations with pagination
func (h *StorageHandler) List(c fiber.Ctx) error {
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, utils.MaxPageSize)

	var total int64
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.StorageLocation{}).Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count storage locations", "database count failed", err)
	}

	var locations []models.StorageLocation
	offset := utils.CalculateOffset(params.Page, params.PageSize)
	if err := h.db.WithContext(c.RequestCtx()).Offset(offset).Limit(params.PageSize).Find(&locations).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch storage locations", "database query failed", err)
	}

	response := utils.NewPaginatedResponse(locations, params.Page, params.PageSize, total)
	return c.JSON(response)
}

// Get returns a single storage location by ID
func (h *StorageHandler) Get(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var location models.StorageLocation
	if err := h.db.WithContext(c.RequestCtx()).First(&location, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "storage location not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch storage location", "database query failed", err)
	}
	return c.JSON(location)
}

// CreateRequest represents the request body for creating a storage location
type CreateRequest struct {
	Name        string             `json:"name"`
	StorageType models.StorageType `json:"storage_type"`
}

// Create creates a new storage location
func (h *StorageHandler) Create(c fiber.Ctx) error {
	var req CreateRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if !req.StorageType.IsValid() {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid storage type, must be 'Box' or 'Binder'")
	}

	location := models.StorageLocation{
		Name:        req.Name,
		StorageType: req.StorageType,
	}

	if err := h.db.WithContext(c.RequestCtx()).Create(&location).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create storage location", "database insert failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(location)
}

// Update updates an existing storage location
func (h *StorageHandler) Update(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var location models.StorageLocation
	if err := h.db.WithContext(c.RequestCtx()).First(&location, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "storage location not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch storage location", "database query failed", err)
	}

	var req CreateRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Update name if provided
	if req.Name != "" {
		location.Name = req.Name
	}

	// Update storage type if valid
	if req.StorageType != "" {
		if !req.StorageType.IsValid() {
			return utils.ReturnError(c, fiber.StatusBadRequest, "invalid storage type, must be 'Box' or 'Binder'")
		}
		location.StorageType = req.StorageType
	}
	if err := h.db.WithContext(c.RequestCtx()).Save(&location).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update storage location", "database update failed", err)
	}

	return c.JSON(location)
}

// Delete deletes a storage location
func (h *StorageHandler) Delete(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Check for inventory references
	var inventoryCount int64
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{}).
		Where("storage_location_id = ?", id).
		Count(&inventoryCount).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to check inventory references", "database count failed", err)
	}

	// Check for sorting rule references
	var ruleCount int64
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.SortingRule{}).
		Where("storage_location_id = ?", id).
		Count(&ruleCount).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to check sorting rule references", "database count failed", err)
	}

	// Prevent deletion if there are references
	if inventoryCount > 0 || ruleCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":              fmt.Sprintf("Cannot delete storage location: %d inventory items and %d sorting rules reference this location", inventoryCount, ruleCount),
			"inventory_count":    inventoryCount,
			"sorting_rule_count": ruleCount,
		})
	}

	result := h.db.WithContext(c.RequestCtx()).Delete(&models.StorageLocation{}, id)
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete storage location", "database delete failed", result.Error)
	}

	if result.RowsAffected == 0 {
		return utils.ReturnError(c, fiber.StatusNotFound, "storage location not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// StorageLocationWithCount represents a storage location with its card count
// tygo:export
type StorageLocationWithCount struct {
	ID          uint               `json:"id"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
	Name        string             `json:"name"`
	StorageType models.StorageType `json:"storage_type"`
	CardCount   int                `json:"card_count"`   // Sum of quantities
	ItemCount   int                `json:"item_count"`   // Count of distinct records
	TotalValue  float64            `json:"total_value"`  // USD total value
}

// ListWithCounts returns all storage locations with card counts, item counts, and total values
func (h *StorageHandler) ListWithCounts(c fiber.Ctx) error {
	// Step 1: Get all storage locations
	var locations []models.StorageLocation
	if err := h.db.WithContext(c.RequestCtx()).Order("storage_type ASC, id ASC").Find(&locations).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch storage locations", "query failed", err)
	}

	// Step 2: Aggregate counts via SQL (avoids loading all inventory into memory)
	type locationCount struct {
		StorageLocationID uint `gorm:"column:storage_location_id"`
		ItemCount         int  `gorm:"column:item_count"`
		CardCount         int  `gorm:"column:card_count"`
	}
	var counts []locationCount
	if err := h.db.WithContext(c.RequestCtx()).Model(&models.Inventory{}).
		Select("storage_location_id, COUNT(*) as item_count, SUM(quantity) as card_count").
		Where("storage_location_id IS NOT NULL").
		Group("storage_location_id").
		Scan(&counts).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to aggregate inventory counts", "query failed", err)
	}
	countMap := make(map[uint]locationCount, len(counts))
	for _, c := range counts {
		countMap[c.StorageLocationID] = c
	}

	// Step 3: Fetch only assigned inventory for value calculation
	var assignedInventory []models.Inventory
	if err := h.db.WithContext(c.RequestCtx()).
		Where("storage_location_id IS NOT NULL").
		Find(&assignedInventory).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch assigned inventory", "query failed", err)
	}

	// Step 4: Group by location and collect unique scryfall IDs
	inventoryByLocation := make(map[uint][]models.Inventory)
	scryfallIDSet := make(map[string]bool)
	for _, inv := range assignedInventory {
		inventoryByLocation[*inv.StorageLocationID] = append(
			inventoryByLocation[*inv.StorageLocationID], inv)
		scryfallIDSet[inv.ScryfallID] = true
	}

	scryfallIDs := make([]string, 0, len(scryfallIDSet))
	for id := range scryfallIDSet {
		scryfallIDs = append(scryfallIDs, id)
	}
	cardMap, err := models.GetCardsByIDs(h.db.WithContext(c.RequestCtx()), scryfallIDs)
	if err != nil {
		slog.Warn("failed to fetch some cards", "component", "storage", "error", err)
	}

	// Step 5: Build results with counts and values
	results := make([]StorageLocationWithCount, len(locations))
	for i, location := range locations {
		lc := countMap[location.ID]
		totalValue := 0.0

		for _, item := range inventoryByLocation[location.ID] {
			if card, ok := cardMap[item.ScryfallID]; ok {
				scryfallCard, parseErr := card.ToScryfallCard()
				if parseErr == nil {
					price := utils.ParsePriceFromScryfall(scryfallCard.Prices, item.Treatment)
					totalValue += price * float64(item.Quantity)
				}
			}
		}

		results[i] = StorageLocationWithCount{
			ID:          location.ID,
			CreatedAt:   location.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   location.UpdatedAt.Format(time.RFC3339),
			Name:        location.Name,
			StorageType: location.StorageType,
			CardCount:   lc.CardCount,
			ItemCount:   lc.ItemCount,
			TotalValue:  totalValue,
		}
	}

	return c.JSON(results)
}
