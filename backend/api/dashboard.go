package api

import (
	"backend/models"
	"backend/utils"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// DashboardHandler handles dashboard endpoints
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// calculateInventoryValue computes the total USD value of inventory items
// using treatment-aware pricing via ParsePriceFromScryfall.
func calculateInventoryValue(db *gorm.DB, items []models.Inventory) float64 {
	if len(items) == 0 {
		return 0
	}

	// Collect unique scryfall IDs
	scryfallIDs := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		if !seen[item.ScryfallID] {
			scryfallIDs = append(scryfallIDs, item.ScryfallID)
			seen[item.ScryfallID] = true
		}
	}

	// Batch fetch card data
	scryfallCardMap, err := models.GetScryfallCardsByIDs(db, scryfallIDs)
	if err != nil {
		slog.Warn("failed to fetch cards for inventory value calculation", "component", "dashboard", "error", err)
		return 0
	}

	var totalValue float64
	for _, item := range items {
		if card, ok := scryfallCardMap[item.ScryfallID]; ok {
			price := utils.ParsePriceFromScryfall(card.Prices, item.Treatment)
			totalValue += price * float64(item.Quantity)
		}
	}
	return totalValue
}

// DashboardStats represents the statistics for the dashboard
// tygo:export
type DashboardStats struct {
	TotalInventoryCards      int64   `json:"total_inventory_cards"`       // Sum of inventory.quantity
	TotalWishlistCards       int64   `json:"total_wishlist_cards"`        // Sum of list_item.collected_quantity
	TotalCollectionValue     float64 `json:"total_collection_value"`      // Value from inventory
	TotalCollectedFromLists  float64 `json:"total_collected_from_lists"`  // Value of cards collected from lists
	TotalRemainingListsValue float64 `json:"total_remaining_lists_value"` // Value of cards still needed from lists
	TotalStorageLocations    int64   `json:"total_storage_locations"`
	TotalLists               int64   `json:"total_lists"`
	UnassignedCards          int64   `json:"unassigned_cards"`
}

// listValueResult holds the computed collected and remaining values for lists.
type listValueResult struct {
	collected float64
	remaining float64
}

// calculateListValues computes the total collected and remaining values for all list items.
func calculateListValues(db *gorm.DB, listItems []models.ListItem) listValueResult {
	// Collect unique scryfall IDs from list items
	scryfallIDs := make([]string, 0, len(listItems))
	scryfallIDSet := make(map[string]bool)
	for _, item := range listItems {
		if !scryfallIDSet[item.ScryfallID] {
			scryfallIDs = append(scryfallIDs, item.ScryfallID)
			scryfallIDSet[item.ScryfallID] = true
		}
	}

	// Batch fetch card data for price information
	scryfallCardMap, err := models.GetScryfallCardsByIDs(db, scryfallIDs)
	if err != nil {
		slog.Warn("failed to fetch cards for list value calculation", "component", "dashboard", "error", err)
		return listValueResult{}
	}

	// Calculate collected and remaining values
	var result listValueResult
	for _, item := range listItems {
		if scryfallCard, ok := scryfallCardMap[item.ScryfallID]; ok {
			price := utils.ParsePriceFromScryfall(scryfallCard.Prices, item.Treatment)

			result.collected += price * float64(item.CollectedQuantity)

			remaining := item.DesiredQuantity - item.CollectedQuantity
			if remaining > 0 {
				result.remaining += price * float64(remaining)
			}
		}
	}

	return result
}

// GetStats returns comprehensive statistics for the dashboard.
//
// This endpoint aggregates data from multiple tables:
// - Storage location count
// - List count
// - Total inventory cards (sum of quantities)
// - Total wishlist cards (sum of collected quantities from list items)
// - Total inventory value (calculated from card prices)
// - Total collected from lists value (value of cards already collected from lists)
// - Total remaining lists value (value of cards still needed to complete lists)
// - Unassigned card count (inventory items without storage location)
func (h *DashboardHandler) GetStats(c fiber.Ctx) error {
	db := h.db.WithContext(c.RequestCtx())
	var stats DashboardStats

	// Count total storage locations
	var storageCount int64
	if err := db.Model(&models.StorageLocation{}).Count(&storageCount).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count storage locations", "database query failed", err)
	}
	stats.TotalStorageLocations = storageCount

	// Count lists
	var listsCount int64
	if err := db.Model(&models.List{}).Count(&listsCount).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count lists", "database query failed", err)
	}
	stats.TotalLists = listsCount

	// Sum total quantity of cards in inventory
	var inventoryCards int64
	if err := db.Model(&models.Inventory{}).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&inventoryCards).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to sum card quantities", "database query failed", err)
	}
	stats.TotalInventoryCards = inventoryCards

	// Sum collected quantity of cards in lists
	var listCards int64
	if err := db.Model(&models.ListItem{}).
		Select("COALESCE(SUM(collected_quantity), 0)").
		Scan(&listCards).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to sum list card quantities", "database query failed", err)
	}
	stats.TotalWishlistCards = listCards

	// Count unassigned cards (inventory items with null storage_location_id)
	var unassignedCount int64
	if err := db.Model(&models.Inventory{}).
		Where("storage_location_id IS NULL").
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&unassignedCount).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count unassigned cards", "database query failed", err)
	}
	stats.UnassignedCards = unassignedCount

	// Calculate total collection value from inventory
	var inventoryItems []models.Inventory
	if err := db.Find(&inventoryItems).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to calculate collection value", "database query failed", err)
	}
	stats.TotalCollectionValue = calculateInventoryValue(db, inventoryItems)

	// Calculate total wishlist values (both collected and remaining)
	var listItems []models.ListItem
	if err := db.Find(&listItems).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch list items", "database query failed", err)
	}

	listValues := calculateListValues(db, listItems)
	stats.TotalCollectedFromLists = listValues.collected
	stats.TotalRemainingListsValue = listValues.remaining

	return c.JSON(stats)
}
