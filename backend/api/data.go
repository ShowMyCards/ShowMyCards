package api

import (
	"backend/models"
	"backend/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// CurrentExportVersion is the latest export format version.
// Bump this when the export schema changes and add a migration function.
const CurrentExportVersion = 1

// DataHandler handles data import and export endpoints
type DataHandler struct {
	db *gorm.DB
}

// NewDataHandler creates a new data handler
func NewDataHandler(db *gorm.DB) *DataHandler {
	return &DataHandler{db: db}
}

// ExportData represents the full application data export
// tygo:export
type ExportData struct {
	Version          int                    `json:"version"`
	ExportedAt       string                 `json:"exported_at"`
	StorageLocations []ExportStorageLocation `json:"storage_locations"`
	SortingRules     []ExportSortingRule     `json:"sorting_rules"`
	Inventory        []ExportInventoryItem   `json:"inventory"`
	Lists            []ExportList            `json:"lists"`
}

// ExportStorageLocation represents a storage location in export format
// tygo:export
type ExportStorageLocation struct {
	RefID       uint               `json:"ref_id"`
	Name        string             `json:"name"`
	StorageType models.StorageType `json:"storage_type"`
}

// ExportSortingRule represents a sorting rule in export format
// tygo:export
type ExportSortingRule struct {
	Name                 string `json:"name"`
	Priority             int    `json:"priority"`
	Expression           string `json:"expression"`
	StorageLocationRefID uint   `json:"storage_location_ref_id"`
	Enabled              bool   `json:"enabled"`
}

// ExportInventoryItem represents an inventory item in export format
// tygo:export
type ExportInventoryItem struct {
	ScryfallID           string `json:"scryfall_id"`
	OracleID             string `json:"oracle_id"`
	Treatment            string `json:"treatment"`
	Quantity             int    `json:"quantity"`
	StorageLocationRefID *uint  `json:"storage_location_ref_id,omitempty"`
}

// ExportList represents a list with its items in export format
// tygo:export
type ExportList struct {
	RefID       uint             `json:"ref_id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Items       []ExportListItem `json:"items"`
}

// ExportListItem represents a list item in export format
// tygo:export
type ExportListItem struct {
	ScryfallID        string `json:"scryfall_id"`
	OracleID          string `json:"oracle_id"`
	Treatment         string `json:"treatment"`
	DesiredQuantity   int    `json:"desired_quantity"`
	CollectedQuantity int    `json:"collected_quantity"`
}

// ImportResponse represents the result of an import operation
// tygo:export
type ImportResponse struct {
	StorageLocationsCreated int      `json:"storage_locations_created"`
	SortingRulesCreated     int      `json:"sorting_rules_created"`
	InventoryItemsCreated   int      `json:"inventory_items_created"`
	ListsCreated            int      `json:"lists_created"`
	ListItemsCreated        int      `json:"list_items_created"`
	Warnings                []string `json:"warnings,omitempty"`
}

// Export returns all user data as a JSON file download
func (h *DataHandler) Export(c fiber.Ctx) error {
	var storageLocations []models.StorageLocation
	var sortingRules []models.SortingRule
	var inventory []models.Inventory
	var lists []models.List

	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Find(&storageLocations).Error; err != nil {
			return fmt.Errorf("storage locations: %w", err)
		}
		if err := tx.Find(&sortingRules).Error; err != nil {
			return fmt.Errorf("sorting rules: %w", err)
		}
		if err := tx.Find(&inventory).Error; err != nil {
			return fmt.Errorf("inventory: %w", err)
		}
		if err := tx.Preload("Items").Find(&lists).Error; err != nil {
			return fmt.Errorf("lists: %w", err)
		}
		return nil
	})
	if err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to export data", "database query failed", err)
	}

	// Build export data
	exportLocations := make([]ExportStorageLocation, len(storageLocations))
	for i, loc := range storageLocations {
		exportLocations[i] = ExportStorageLocation{
			RefID:       loc.ID,
			Name:        loc.Name,
			StorageType: loc.StorageType,
		}
	}

	exportRules := make([]ExportSortingRule, len(sortingRules))
	for i, rule := range sortingRules {
		exportRules[i] = ExportSortingRule{
			Name:                 rule.Name,
			Priority:             rule.Priority,
			Expression:           rule.Expression,
			StorageLocationRefID: rule.StorageLocationID,
			Enabled:              rule.Enabled,
		}
	}

	exportInventory := make([]ExportInventoryItem, len(inventory))
	for i, inv := range inventory {
		exportInventory[i] = ExportInventoryItem{
			ScryfallID: inv.ScryfallID,
			OracleID:   inv.OracleID,
			Treatment:  inv.Treatment,
			Quantity:    inv.Quantity,
		}
		if inv.StorageLocationID != nil {
			exportInventory[i].StorageLocationRefID = inv.StorageLocationID
		}
	}

	exportLists := make([]ExportList, len(lists))
	for i, list := range lists {
		items := make([]ExportListItem, len(list.Items))
		for j, item := range list.Items {
			items[j] = ExportListItem{
				ScryfallID:        item.ScryfallID,
				OracleID:          item.OracleID,
				Treatment:         item.Treatment,
				DesiredQuantity:   item.DesiredQuantity,
				CollectedQuantity: item.CollectedQuantity,
			}
		}
		exportLists[i] = ExportList{
			RefID:       list.ID,
			Name:        list.Name,
			Description: list.Description,
			Items:       items,
		}
	}

	data := ExportData{
		Version:          CurrentExportVersion,
		ExportedAt:       time.Now().UTC().Format(time.RFC3339),
		StorageLocations: exportLocations,
		SortingRules:     exportRules,
		Inventory:        exportInventory,
		Lists:            exportLists,
	}

	filename := fmt.Sprintf("showmycards-export-%s.json", time.Now().UTC().Format("2006-01-02"))
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.JSON(data)
}

// Import accepts exported JSON data and creates records additively
func (h *DataHandler) Import(c fiber.Ctx) error {
	// Parse raw JSON first for version checking and potential migration
	var raw map[string]any
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid JSON in request body")
	}

	versionVal, ok := raw["version"]
	if !ok {
		return utils.ReturnError(c, fiber.StatusBadRequest, "missing 'version' field in export data")
	}

	versionNum, ok := versionVal.(float64)
	if !ok {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid 'version' field: must be a number")
	}
	version := int(versionNum)

	if version > CurrentExportVersion {
		return utils.ReturnError(c, fiber.StatusBadRequest,
			fmt.Sprintf("export version %d is not supported by this version of ShowMyCards — please update to the latest version", version))
	}

	if version < 1 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid export version: must be at least 1")
	}

	// Apply migrations if needed (future-proofing)
	if version < CurrentExportVersion {
		if err := applyMigrations(raw, version); err != nil {
			slog.Error("export data migration failed", "from_version", version, "to_version", CurrentExportVersion, "error", err)
			return utils.ReturnError(c, fiber.StatusBadRequest,
				fmt.Sprintf("failed to migrate export data from version %d to %d", version, CurrentExportVersion))
		}
	}

	// Re-marshal and unmarshal into typed struct after potential migration
	migrated, err := json.Marshal(raw)
	if err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "failed to process export data")
	}

	var data ExportData
	if err := json.Unmarshal(migrated, &data); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid export data structure")
	}

	response := ImportResponse{}
	storageRefMap := make(map[uint]uint) // old ref_id -> new DB id

	err = h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		// 1. Storage Locations — created first because inventory and rules reference them
		for _, loc := range data.StorageLocations {
			newLoc := models.StorageLocation{
				Name:        loc.Name,
				StorageType: loc.StorageType,
			}
			if err := tx.Create(&newLoc).Error; err != nil {
				return fmt.Errorf("failed to create storage location %q: %w", loc.Name, err)
			}
			storageRefMap[loc.RefID] = newLoc.ID
			response.StorageLocationsCreated++
		}

		// 2. Sorting Rules — reference storage locations via ref_id
		for _, rule := range data.SortingRules {
			newLocID, ok := storageRefMap[rule.StorageLocationRefID]
			if !ok {
				response.Warnings = append(response.Warnings,
					fmt.Sprintf("sorting rule %q references unknown storage location ref %d, skipped",
						rule.Name, rule.StorageLocationRefID))
				continue
			}
			newRule := models.SortingRule{
				Name:              rule.Name,
				Priority:          rule.Priority,
				Expression:        rule.Expression,
				StorageLocationID: newLocID,
				Enabled:           rule.Enabled,
			}
			if err := tx.Create(&newRule).Error; err != nil {
				return fmt.Errorf("failed to create sorting rule %q: %w", rule.Name, err)
			}
			response.SortingRulesCreated++
		}

		// 3. Lists + Items — items nested under their parent list
		for _, list := range data.Lists {
			newList := models.List{
				Name:        list.Name,
				Description: list.Description,
			}
			if err := tx.Create(&newList).Error; err != nil {
				return fmt.Errorf("failed to create list %q: %w", list.Name, err)
			}
			response.ListsCreated++

			for _, item := range list.Items {
				newItem := models.ListItem{
					ListID:            newList.ID,
					ScryfallID:        item.ScryfallID,
					OracleID:          item.OracleID,
					Treatment:         item.Treatment,
					DesiredQuantity:   item.DesiredQuantity,
					CollectedQuantity: item.CollectedQuantity,
				}
				if err := tx.Create(&newItem).Error; err != nil {
					if isDuplicateError(err) {
						response.Warnings = append(response.Warnings,
							fmt.Sprintf("skipped duplicate list item %s in list %q",
								item.ScryfallID, list.Name))
						continue
					}
					return fmt.Errorf("failed to create list item %s in list %q: %w",
						item.ScryfallID, list.Name, err)
				}
				response.ListItemsCreated++
			}
		}

		// 4. Inventory — reference storage locations via ref_id
		for _, inv := range data.Inventory {
			var storageLocID *uint
			if inv.StorageLocationRefID != nil {
				if newID, ok := storageRefMap[*inv.StorageLocationRefID]; ok {
					storageLocID = &newID
				} else {
					response.Warnings = append(response.Warnings,
						fmt.Sprintf("inventory item %s references unknown storage location ref %d, imported without location",
							inv.ScryfallID, *inv.StorageLocationRefID))
				}
			}
			newInv := models.Inventory{
				ScryfallID:        inv.ScryfallID,
				OracleID:          inv.OracleID,
				Treatment:         inv.Treatment,
				Quantity:          inv.Quantity,
				StorageLocationID: storageLocID,
			}
			if err := tx.Create(&newInv).Error; err != nil {
				if isDuplicateError(err) {
					response.Warnings = append(response.Warnings,
						fmt.Sprintf("skipped duplicate inventory item %s (treatment: %s)",
							inv.ScryfallID, inv.Treatment))
					continue
				}
				return fmt.Errorf("failed to create inventory item %s: %w", inv.ScryfallID, err)
			}
			response.InventoryItemsCreated++
		}

		return nil
	})

	if err != nil {
		slog.Error("import failed", "error", err)
		return utils.ReturnError(c, fiber.StatusInternalServerError,
			"Import failed, all changes have been rolled back")
	}

	return c.JSON(response)
}

// isDuplicateError checks if a GORM error is a UNIQUE constraint violation.
// NOTE: This matches the SQLite error string. If the database backend changes,
// this function must be updated to match the new driver's constraint error format.
func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// migrations maps each version to a function that migrates data to the next version.
// When bumping CurrentExportVersion, add a migration here.
var migrations = map[int]func(map[string]any) error{}

// applyMigrations runs sequential migration functions to bring data up to CurrentExportVersion
func applyMigrations(data map[string]any, fromVersion int) error {
	for v := fromVersion; v < CurrentExportVersion; v++ {
		migrate, ok := migrations[v]
		if !ok {
			return fmt.Errorf("no migration defined for version %d to %d", v, v+1)
		}
		if err := migrate(data); err != nil {
			return fmt.Errorf("migration v%d→v%d failed: %w", v, v+1, err)
		}
	}
	data["version"] = float64(CurrentExportVersion)
	return nil
}
