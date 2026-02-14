package database

import (
	"backend/models"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient_Success(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify client was created
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.DB == nil {
		t.Fatal("expected DB to be non-nil")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file to exist at %s", dbPath)
	}
}

func TestNewClient_CreatesDirectory(t *testing.T) {
	// Create temporary parent directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify nested directory was created
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected directory to exist at %s", dir)
	}
}

func TestNewClient_InMemoryDatabase(t *testing.T) {
	client, err := NewClient(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory client: %v", err)
	}
	defer client.Close()

	// Verify client was created
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.DB == nil {
		t.Fatal("expected DB to be non-nil")
	}
}

func TestNewClient_RunsMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify all tables were created
	expectedTables := []string{
		"storage_locations",
		"sorting_rules",
		"inventories",
		"lists",
		"list_items",
		"settings",
		"jobs",
		"cards",
	}

	for _, tableName := range expectedTables {
		if !client.DB.Migrator().HasTable(tableName) {
			t.Errorf("expected table %s to exist", tableName)
		}
	}
}

func TestNewClient_ConnectionPoolSettings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	sqlDB, err := client.DB.DB()
	if err != nil {
		t.Fatalf("failed to get database instance: %v", err)
	}

	// Verify connection pool settings
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("expected MaxOpenConnections to be 1, got %d", stats.MaxOpenConnections)
	}
}

func TestClose_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("expected Close to succeed, got error: %v", err)
	}
}

func TestMigrate_AllModels(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify we can create records in each table
	tests := []struct {
		name  string
		model interface{}
	}{
		{"StorageLocation", &models.StorageLocation{Name: "Test Box", StorageType: models.Box}},
		{"SortingRule", &models.SortingRule{Name: "Test Rule", Expression: "true", Priority: 1, StorageLocationID: 1, Enabled: true}},
		{"Inventory", &models.Inventory{ScryfallID: "test-id", OracleID: "oracle-id", Treatment: "nonfoil", Quantity: 1}},
		{"List", &models.List{Name: "Test List"}},
		{"Setting", &models.Setting{Key: "test_key", Value: "test_value"}},
		{"Job", &models.Job{Type: models.JobTypeBulkDataImport, Status: models.JobStatusPending}},
		{"Card", &models.Card{ScryfallID: "card-id", OracleID: "oracle-id", RawJSON: `{"name":"Test Card","set":"tst"}`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.DB.Create(tt.model)
			if result.Error != nil {
				t.Errorf("failed to create %s: %v", tt.name, result.Error)
			}
		})
	}
}

func TestCustomMigrations_GeneratedColumns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Create a card with raw JSON
	card := &models.Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    `{"name": "Lightning Bolt", "set": "lea"}`,
	}

	if err := client.DB.Create(card).Error; err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Verify generated columns were populated using raw SQL
	// (GORM's Select("*") doesn't include gorm:"-" tagged fields)
	var name, setCode string
	err = client.DB.Raw("SELECT name, set_code FROM cards WHERE scryfall_id = ?", "test-id").Row().Scan(&name, &setCode)
	if err != nil {
		t.Fatalf("failed to query generated columns: %v", err)
	}

	// Generated columns should be automatically populated from JSON
	if name != "Lightning Bolt" {
		t.Errorf("expected name 'Lightning Bolt', got '%s'", name)
	}
	if setCode != "lea" {
		t.Errorf("expected set_code 'lea', got '%s'", setCode)
	}
}

func TestCustomMigrations_Indexes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify indexes exist
	expectedIndexes := []string{
		"idx_cards_name",
		"idx_cards_set_code",
	}

	for _, indexName := range expectedIndexes {
		var count int64
		client.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", indexName).Scan(&count)
		if count == 0 {
			t.Errorf("expected index %s to exist", indexName)
		}
	}
}

func TestCustomMigrations_DropsLegacyTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a client to set up the database
	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Manually create legacy table
	if err := client.DB.Exec("CREATE TABLE IF NOT EXISTS bulk_cards (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatalf("failed to create legacy table: %v", err)
	}

	// Verify legacy table exists
	if !client.DB.Migrator().HasTable("bulk_cards") {
		t.Fatal("expected legacy table to exist before migration")
	}

	client.Close()

	// Run migrations again (simulating app restart)
	client, err = NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify legacy table was dropped
	if client.DB.Migrator().HasTable("bulk_cards") {
		t.Error("expected legacy table to be dropped")
	}
}

func TestCustomMigrations_IdempotentColumnAddition(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create initial client
	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.Close()

	// Run migrations again (should not error)
	client, err = NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to run migrations second time: %v", err)
	}
	defer client.Close()

	// Verify columns still exist and work
	card := &models.Card{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		RawJSON:    `{"name": "Test Card", "set": "tst"}`,
	}

	if err := client.DB.Create(card).Error; err != nil {
		t.Fatalf("failed to create card after re-migration: %v", err)
	}

	// Verify generated columns using raw SQL
	var name string
	err = client.DB.Raw("SELECT name FROM cards WHERE scryfall_id = ?", "test-id").Row().Scan(&name)
	if err != nil {
		t.Fatalf("failed to query generated column: %v", err)
	}

	if name != "Test Card" {
		t.Errorf("expected name 'Test Card', got '%s'", name)
	}
}

func TestMigrate_Relationships(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Create parent records
	storage := &models.StorageLocation{Name: "Test Box", StorageType: models.Box}
	if err := client.DB.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	list := &models.List{Name: "Test List"}
	if err := client.DB.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	// Create child records with foreign keys
	sortingRule := &models.SortingRule{
		Name:              "Test Rule",
		Expression:        "true",
		Priority:          1,
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := client.DB.Create(sortingRule).Error; err != nil {
		t.Fatalf("failed to create sorting rule: %v", err)
	}

	inventory := &models.Inventory{
		ScryfallID:        "test-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		Quantity:          1,
		StorageLocationID: &storage.ID,
	}
	if err := client.DB.Create(inventory).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	listItem := &models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "test-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		DesiredQuantity:   4,
		CollectedQuantity: 1,
	}
	if err := client.DB.Create(listItem).Error; err != nil {
		t.Fatalf("failed to create list item: %v", err)
	}

	// Verify relationships work via preloading
	var loadedRule models.SortingRule
	if err := client.DB.Preload("StorageLocation").First(&loadedRule, sortingRule.ID).Error; err != nil {
		t.Fatalf("failed to preload sorting rule: %v", err)
	}
	if loadedRule.StorageLocation.Name != "Test Box" {
		t.Errorf("expected storage location name 'Test Box', got '%s'", loadedRule.StorageLocation.Name)
	}

	var loadedInventory models.Inventory
	if err := client.DB.Preload("StorageLocation").First(&loadedInventory, inventory.ID).Error; err != nil {
		t.Fatalf("failed to preload inventory: %v", err)
	}
	if loadedInventory.StorageLocation.Name != "Test Box" {
		t.Errorf("expected storage location name 'Test Box', got '%s'", loadedInventory.StorageLocation.Name)
	}

	var loadedList models.List
	if err := client.DB.Preload("Items").First(&loadedList, list.ID).Error; err != nil {
		t.Fatalf("failed to preload list: %v", err)
	}
	if len(loadedList.Items) != 1 {
		t.Errorf("expected 1 list item, got %d", len(loadedList.Items))
	}
}

func TestMigrate_CascadeDeletes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Enable foreign key constraints (required for ON DELETE CASCADE to work)
	if err := client.DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Create list with items
	list := &models.List{Name: "Test List"}
	if err := client.DB.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	listItem := &models.ListItem{
		ListID:            list.ID,
		ScryfallID:        "test-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		DesiredQuantity:   4,
		CollectedQuantity: 1,
	}
	if err := client.DB.Create(listItem).Error; err != nil {
		t.Fatalf("failed to create list item: %v", err)
	}

	// Delete list (should cascade delete items)
	// Note: Need to use Select(clause.Associations) for cascade to work with GORM
	if err := client.DB.Select("Items").Delete(list).Error; err != nil {
		t.Fatalf("failed to delete list: %v", err)
	}

	// Verify list was deleted
	var deletedList models.List
	err = client.DB.First(&deletedList, list.ID).Error
	if err == nil {
		t.Error("expected list to be deleted")
	}

	// Verify list item was cascade deleted (via foreign key constraint)
	var count int64
	client.DB.Model(&models.ListItem{}).Where("list_id = ?", list.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected list items to be cascade deleted, found %d", count)
	}
}

func TestMigrate_SetNullDeletes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Enable foreign key constraints (required for ON DELETE SET NULL to work)
	if err := client.DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Create storage with inventory
	storage := &models.StorageLocation{Name: "Test Box", StorageType: models.Box}
	if err := client.DB.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	inventory := &models.Inventory{
		ScryfallID:        "test-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		Quantity:          1,
		StorageLocationID: &storage.ID,
	}
	if err := client.DB.Create(inventory).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	// Delete storage (should set inventory.storage_location_id to NULL via foreign key constraint)
	if err := client.DB.Delete(storage).Error; err != nil {
		t.Fatalf("failed to delete storage: %v", err)
	}

	// Verify storage was deleted
	var deletedStorage models.StorageLocation
	err = client.DB.First(&deletedStorage, storage.ID).Error
	if err == nil {
		t.Error("expected storage to be deleted")
	}

	// Verify inventory still exists but storage_location_id is NULL (via ON DELETE SET NULL)
	var loadedInventory models.Inventory
	if err := client.DB.First(&loadedInventory, inventory.ID).Error; err != nil {
		t.Fatalf("failed to retrieve inventory: %v", err)
	}
	if loadedInventory.StorageLocationID != nil {
		t.Errorf("expected storage_location_id to be NULL after storage deletion, got %v", *loadedInventory.StorageLocationID)
	}
}
