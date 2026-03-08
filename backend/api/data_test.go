package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/models"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDataTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.StorageLocation{},
		&models.Inventory{},
		&models.SortingRule{},
		&models.List{},
		&models.ListItem{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	app := fiber.New()
	handler := NewDataHandler(db)

	app.Get("/api/data/export", handler.Export)
	app.Post("/api/data/import", handler.Import)

	return app, db
}

func seedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Create storage locations
	box := models.StorageLocation{Name: "Mythic Box", StorageType: models.Box}
	if err := db.Create(&box).Error; err != nil {
		t.Fatalf("failed to create storage location: %v", err)
	}
	binder := models.StorageLocation{Name: "Foil Binder", StorageType: models.Binder}
	if err := db.Create(&binder).Error; err != nil {
		t.Fatalf("failed to create storage location: %v", err)
	}

	// Create sorting rules
	rule := models.SortingRule{
		Name:              "Mythics to Box",
		Priority:          1,
		Expression:        `rarity == "mythic"`,
		StorageLocationID: box.ID,
		Enabled:           true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create sorting rule: %v", err)
	}

	// Create inventory
	inv1 := models.Inventory{
		ScryfallID:        "scry-001",
		OracleID:          "oracle-001",
		Treatment:         "nonfoil",
		Quantity:          2,
		StorageLocationID: &box.ID,
	}
	if err := db.Create(&inv1).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}
	inv2 := models.Inventory{
		ScryfallID: "scry-002",
		OracleID:   "oracle-002",
		Treatment:  "foil",
		Quantity:   1,
	}
	if err := db.Create(&inv2).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	// Create list with items
	list := models.List{Name: "Commander Deck", Description: "My Ur-Dragon deck"}
	if err := db.Create(&list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}
	item := models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "scry-001",
		OracleID:          "oracle-001",
		Treatment:         "nonfoil",
		DesiredQuantity:   4,
		CollectedQuantity: 2,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create list item: %v", err)
	}
}

// Export tests

func TestExport_Empty(t *testing.T) {
	app, _ := setupDataTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/data/export", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ExportData
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Version != CurrentExportVersion {
		t.Errorf("expected version %d, got %d", CurrentExportVersion, result.Version)
	}
	if result.ExportedAt == "" {
		t.Error("expected exported_at to be set")
	}
	if len(result.StorageLocations) != 0 {
		t.Errorf("expected 0 storage locations, got %d", len(result.StorageLocations))
	}
	if len(result.Inventory) != 0 {
		t.Errorf("expected 0 inventory items, got %d", len(result.Inventory))
	}
	if len(result.Lists) != 0 {
		t.Errorf("expected 0 lists, got %d", len(result.Lists))
	}
	if len(result.SortingRules) != 0 {
		t.Errorf("expected 0 sorting rules, got %d", len(result.SortingRules))
	}
}

func TestExport_WithData(t *testing.T) {
	app, db := setupDataTestApp(t)
	seedTestData(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/data/export", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check Content-Disposition header
	contentDisp := resp.Header.Get("Content-Disposition")
	if contentDisp == "" {
		t.Error("expected Content-Disposition header to be set")
	}

	var result ExportData
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Version != CurrentExportVersion {
		t.Errorf("expected version %d, got %d", CurrentExportVersion, result.Version)
	}
	if len(result.StorageLocations) != 2 {
		t.Errorf("expected 2 storage locations, got %d", len(result.StorageLocations))
	}
	if len(result.SortingRules) != 1 {
		t.Errorf("expected 1 sorting rule, got %d", len(result.SortingRules))
	}
	if len(result.Inventory) != 2 {
		t.Errorf("expected 2 inventory items, got %d", len(result.Inventory))
	}
	if len(result.Lists) != 1 {
		t.Errorf("expected 1 list, got %d", len(result.Lists))
	}
	if len(result.Lists[0].Items) != 1 {
		t.Errorf("expected 1 list item, got %d", len(result.Lists[0].Items))
	}
}

func TestExport_StorageLocationRefIDs(t *testing.T) {
	app, db := setupDataTestApp(t)
	seedTestData(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/data/export", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result ExportData
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify inventory items have correct ref_id linkage
	for _, inv := range result.Inventory {
		if inv.ScryfallID == "scry-001" {
			if inv.StorageLocationRefID == nil {
				t.Error("expected scry-001 to have a storage_location_ref_id")
			}
		}
		if inv.ScryfallID == "scry-002" {
			if inv.StorageLocationRefID != nil {
				t.Error("expected scry-002 to have no storage_location_ref_id")
			}
		}
	}

	// Verify sorting rule references a valid storage location ref_id
	if len(result.SortingRules) > 0 {
		ruleRefID := result.SortingRules[0].StorageLocationRefID
		found := false
		for _, loc := range result.StorageLocations {
			if loc.RefID == ruleRefID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sorting rule references storage_location_ref_id %d which doesn't exist in export", ruleRefID)
		}
	}
}

// Import tests

func TestImport_ValidData(t *testing.T) {
	app, db := setupDataTestApp(t)

	importData := ExportData{
		Version:    1,
		ExportedAt: "2026-01-01T00:00:00Z",
		StorageLocations: []ExportStorageLocation{
			{RefID: 10, Name: "Imported Box", StorageType: models.Box},
		},
		SortingRules: []ExportSortingRule{
			{Name: "Test Rule", Priority: 1, Expression: `rarity == "mythic"`, StorageLocationRefID: 10, Enabled: true},
		},
		Inventory: []ExportInventoryItem{
			{ScryfallID: "scry-100", OracleID: "oracle-100", Treatment: "nonfoil", Quantity: 3, StorageLocationRefID: uintPtr(10)},
		},
		Lists: []ExportList{
			{
				RefID: 20, Name: "Imported List", Description: "A test list",
				Items: []ExportListItem{
					{ScryfallID: "scry-100", OracleID: "oracle-100", Treatment: "nonfoil", DesiredQuantity: 4, CollectedQuantity: 1},
				},
			},
		},
	}

	body, _ := json.Marshal(importData)
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationsCreated != 1 {
		t.Errorf("expected 1 storage location created, got %d", result.StorageLocationsCreated)
	}
	if result.SortingRulesCreated != 1 {
		t.Errorf("expected 1 sorting rule created, got %d", result.SortingRulesCreated)
	}
	if result.InventoryItemsCreated != 1 {
		t.Errorf("expected 1 inventory item created, got %d", result.InventoryItemsCreated)
	}
	if result.ListsCreated != 1 {
		t.Errorf("expected 1 list created, got %d", result.ListsCreated)
	}
	if result.ListItemsCreated != 1 {
		t.Errorf("expected 1 list item created, got %d", result.ListItemsCreated)
	}

	// Verify data actually exists in database
	var locCount int64
	db.Model(&models.StorageLocation{}).Count(&locCount)
	if locCount != 1 {
		t.Errorf("expected 1 storage location in DB, got %d", locCount)
	}

	var invCount int64
	db.Model(&models.Inventory{}).Count(&invCount)
	if invCount != 1 {
		t.Errorf("expected 1 inventory item in DB, got %d", invCount)
	}

	// Verify inventory has correct storage location mapping
	var inv models.Inventory
	db.First(&inv)
	if inv.StorageLocationID == nil {
		t.Error("expected inventory to have a storage location")
	}
}

func TestImport_FutureVersion(t *testing.T) {
	app, _ := setupDataTestApp(t)

	body := `{"version": 999, "exported_at": "2026-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["error"] == "" {
		t.Error("expected error message about unsupported version")
	}
}

func TestImport_MissingVersion(t *testing.T) {
	app, _ := setupDataTestApp(t)

	body := `{"exported_at": "2026-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestImport_InvalidJSON(t *testing.T) {
	app, _ := setupDataTestApp(t)

	body := `{not valid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestImport_UnknownStorageLocationRef(t *testing.T) {
	app, _ := setupDataTestApp(t)

	importData := ExportData{
		Version:    1,
		ExportedAt: "2026-01-01T00:00:00Z",
		SortingRules: []ExportSortingRule{
			{Name: "Orphan Rule", Priority: 1, Expression: "true", StorageLocationRefID: 999, Enabled: true},
		},
		Inventory: []ExportInventoryItem{
			{ScryfallID: "scry-100", OracleID: "oracle-100", Treatment: "nonfoil", Quantity: 1, StorageLocationRefID: uintPtr(999)},
		},
	}

	body, _ := json.Marshal(importData)
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Rule should be skipped with warning
	if result.SortingRulesCreated != 0 {
		t.Errorf("expected 0 sorting rules created, got %d", result.SortingRulesCreated)
	}
	// Inventory should be imported without location with warning
	if result.InventoryItemsCreated != 1 {
		t.Errorf("expected 1 inventory item created (without location), got %d", result.InventoryItemsCreated)
	}
	if len(result.Warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(result.Warnings))
	}
}

func TestImport_AdditiveWithExistingData(t *testing.T) {
	app, db := setupDataTestApp(t)

	// Pre-populate with existing data
	existing := models.StorageLocation{Name: "Existing Box", StorageType: models.Box}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("failed to create existing location: %v", err)
	}

	importData := ExportData{
		Version:    1,
		ExportedAt: "2026-01-01T00:00:00Z",
		StorageLocations: []ExportStorageLocation{
			{RefID: 1, Name: "New Box", StorageType: models.Box},
		},
	}

	body, _ := json.Marshal(importData)
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify both exist (additive merge)
	var count int64
	db.Model(&models.StorageLocation{}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 storage locations after additive import, got %d", count)
	}
}

func TestImport_RoundTrip(t *testing.T) {
	app, db := setupDataTestApp(t)
	seedTestData(t, db)

	// Step 1: Export
	exportReq := httptest.NewRequest(http.MethodGet, "/api/data/export", nil)
	exportResp, err := app.Test(exportReq)
	if err != nil {
		t.Fatalf("export request failed: %v", err)
	}
	defer func() { _ = exportResp.Body.Close() }()

	var exportedData ExportData
	if err := json.NewDecoder(exportResp.Body).Decode(&exportedData); err != nil {
		t.Fatalf("failed to decode export: %v", err)
	}

	// Step 2: Import into a fresh database
	freshApp, freshDB := setupDataTestApp(t)

	body, _ := json.Marshal(exportedData)
	importReq := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	importReq.Header.Set("Content-Type", "application/json")

	importResp, err := freshApp.Test(importReq)
	if err != nil {
		t.Fatalf("import request failed: %v", err)
	}
	defer func() { _ = importResp.Body.Close() }()

	if importResp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, importResp.StatusCode)
	}

	var importResult ImportResponse
	if err := json.NewDecoder(importResp.Body).Decode(&importResult); err != nil {
		t.Fatalf("failed to decode import response: %v", err)
	}

	// Step 3: Verify counts match
	var origLocCount, newLocCount int64
	db.Model(&models.StorageLocation{}).Count(&origLocCount)
	freshDB.Model(&models.StorageLocation{}).Count(&newLocCount)
	if origLocCount != newLocCount {
		t.Errorf("storage location count mismatch: original %d, imported %d", origLocCount, newLocCount)
	}

	var origInvCount, newInvCount int64
	db.Model(&models.Inventory{}).Count(&origInvCount)
	freshDB.Model(&models.Inventory{}).Count(&newInvCount)
	if origInvCount != newInvCount {
		t.Errorf("inventory count mismatch: original %d, imported %d", origInvCount, newInvCount)
	}

	var origListCount, newListCount int64
	db.Model(&models.List{}).Count(&origListCount)
	freshDB.Model(&models.List{}).Count(&newListCount)
	if origListCount != newListCount {
		t.Errorf("list count mismatch: original %d, imported %d", origListCount, newListCount)
	}

	var origItemCount, newItemCount int64
	db.Model(&models.ListItem{}).Count(&origItemCount)
	freshDB.Model(&models.ListItem{}).Count(&newItemCount)
	if origItemCount != newItemCount {
		t.Errorf("list item count mismatch: original %d, imported %d", origItemCount, newItemCount)
	}

	var origRuleCount, newRuleCount int64
	db.Model(&models.SortingRule{}).Count(&origRuleCount)
	freshDB.Model(&models.SortingRule{}).Count(&newRuleCount)
	if origRuleCount != newRuleCount {
		t.Errorf("sorting rule count mismatch: original %d, imported %d", origRuleCount, newRuleCount)
	}

	// Step 4: Verify data integrity
	var importedInv models.Inventory
	freshDB.Where("scryfall_id = ?", "scry-001").First(&importedInv)
	if importedInv.Quantity != 2 {
		t.Errorf("expected quantity 2 for scry-001, got %d", importedInv.Quantity)
	}
	if importedInv.StorageLocationID == nil {
		t.Error("expected imported scry-001 to have a storage location")
	}

	var importedList models.List
	freshDB.Preload("Items").Where("name = ?", "Commander Deck").First(&importedList)
	if len(importedList.Items) != 1 {
		t.Errorf("expected 1 list item, got %d", len(importedList.Items))
	}
	if len(importedList.Items) > 0 && importedList.Items[0].DesiredQuantity != 4 {
		t.Errorf("expected desired_quantity 4, got %d", importedList.Items[0].DesiredQuantity)
	}
}

func TestImport_EmptyData(t *testing.T) {
	app, _ := setupDataTestApp(t)

	importData := ExportData{
		Version:    1,
		ExportedAt: "2026-01-01T00:00:00Z",
	}

	body, _ := json.Marshal(importData)
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StorageLocationsCreated != 0 || result.InventoryItemsCreated != 0 ||
		result.ListsCreated != 0 || result.ListItemsCreated != 0 || result.SortingRulesCreated != 0 {
		t.Error("expected all counts to be 0 for empty import")
	}
}

func TestImport_InventoryWithoutStorageLocation(t *testing.T) {
	app, _ := setupDataTestApp(t)

	importData := ExportData{
		Version:    1,
		ExportedAt: "2026-01-01T00:00:00Z",
		Inventory: []ExportInventoryItem{
			{ScryfallID: "scry-100", OracleID: "oracle-100", Treatment: "nonfoil", Quantity: 1},
		},
	}

	body, _ := json.Marshal(importData)
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.InventoryItemsCreated != 1 {
		t.Errorf("expected 1 inventory item created, got %d", result.InventoryItemsCreated)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
	}
}

func TestImport_InvalidVersion(t *testing.T) {
	app, _ := setupDataTestApp(t)

	body := `{"version": 0, "exported_at": "2026-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/data/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// Version migration tests

func TestApplyMigrations_NoMigrationsNeeded(t *testing.T) {
	data := map[string]any{"version": float64(CurrentExportVersion)}
	err := applyMigrations(data, CurrentExportVersion)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func uintPtr(v uint) *uint {
	return &v
}
