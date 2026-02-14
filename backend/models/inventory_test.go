package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInventoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&Inventory{}, &StorageLocation{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestInventory_ValidateInventory(t *testing.T) {
	db := setupInventoryTestDB(t)

	tests := []struct {
		name        string
		inventory   *Inventory
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Inventory",
			inventory: &Inventory{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   1,
			},
			expectError: false,
		},
		{
			name: "Valid Inventory - Zero Quantity",
			inventory: &Inventory{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				Treatment:  "foil",
				Quantity:   0,
			},
			expectError: false,
		},
		{
			name: "Invalid - Empty ScryfallID",
			inventory: &Inventory{
				ScryfallID: "",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   1,
			},
			expectError: true,
			errorMsg:    "scryfall_id cannot be empty",
		},
		{
			name: "Invalid - Empty OracleID",
			inventory: &Inventory{
				ScryfallID: "test-id",
				OracleID:   "",
				Treatment:  "nonfoil",
				Quantity:   1,
			},
			expectError: true,
			errorMsg:    "oracle_id cannot be empty",
		},
		{
			name: "Invalid - Negative Quantity",
			inventory: &Inventory{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   -1,
			},
			expectError: true,
			errorMsg:    "quantity cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.inventory.ValidateInventory(db)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestInventory_BeforeCreate(t *testing.T) {
	db := setupInventoryTestDB(t)

	tests := []struct {
		name        string
		inventory   *Inventory
		expectError bool
	}{
		{
			name: "Valid Create",
			inventory: &Inventory{
				ScryfallID: "test-id-1",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   1,
			},
			expectError: false,
		},
		{
			name: "Invalid Create - Empty ScryfallID",
			inventory: &Inventory{
				ScryfallID: "",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   1,
			},
			expectError: true,
		},
		{
			name: "Invalid Create - Negative Quantity",
			inventory: &Inventory{
				ScryfallID: "test-id-2",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   -5,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.inventory).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestInventory_BeforeUpdate(t *testing.T) {
	db := setupInventoryTestDB(t)

	tests := []struct {
		name        string
		updateFunc  func(*Inventory)
		expectError bool
	}{
		{
			name: "Valid Update - Quantity",
			updateFunc: func(i *Inventory) {
				i.Quantity = 5
			},
			expectError: false,
		},
		{
			name: "Valid Update - Treatment",
			updateFunc: func(i *Inventory) {
				i.Treatment = "foil"
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Negative Quantity",
			updateFunc: func(i *Inventory) {
				i.Quantity = -1
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Empty OracleID",
			updateFunc: func(i *Inventory) {
				i.OracleID = ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh inventory for each test
			testInventory := &Inventory{
				ScryfallID: "test-id",
				OracleID:   "oracle-id",
				Treatment:  "nonfoil",
				Quantity:   1,
			}
			if err := db.Create(testInventory).Error; err != nil {
				t.Fatalf("failed to create inventory: %v", err)
			}

			// Apply update
			tt.updateFunc(testInventory)

			// Try to save
			err := db.Save(testInventory).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestInventory_WithStorageLocation(t *testing.T) {
	db := setupInventoryTestDB(t)

	// Create storage location
	storage := &StorageLocation{
		Name:        "Test Box",
		StorageType: Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create inventory with storage
	inventory := &Inventory{
		ScryfallID:        "test-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		Quantity:          1,
		StorageLocationID: &storage.ID,
	}
	if err := db.Create(inventory).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	// Verify relationship
	var loadedInventory Inventory
	if err := db.Preload("StorageLocation").First(&loadedInventory, inventory.ID).Error; err != nil {
		t.Fatalf("failed to load inventory: %v", err)
	}

	if loadedInventory.StorageLocation == nil {
		t.Fatal("expected StorageLocation to be loaded")
	}
	if loadedInventory.StorageLocation.Name != "Test Box" {
		t.Errorf("expected storage name 'Test Box', got '%s'", loadedInventory.StorageLocation.Name)
	}
}

// LOW-VALUE: Tests that a nil pointer remains nil after a GORM DB round-trip. Verifies GORM nullable handling.
func TestInventory_NullStorageLocation(t *testing.T) {
	db := setupInventoryTestDB(t)

	// Create inventory without storage
	inventory := &Inventory{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		Treatment:  "nonfoil",
		Quantity:   1,
	}
	if err := db.Create(inventory).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	// Verify storage is nil
	if inventory.StorageLocationID != nil {
		t.Errorf("expected StorageLocationID to be nil, got %v", *inventory.StorageLocationID)
	}

	// Reload and verify
	var loadedInventory Inventory
	if err := db.First(&loadedInventory, inventory.ID).Error; err != nil {
		t.Fatalf("failed to load inventory: %v", err)
	}

	if loadedInventory.StorageLocationID != nil {
		t.Errorf("expected StorageLocationID to be nil after reload, got %v", *loadedInventory.StorageLocationID)
	}
}

// LOW-VALUE: Tests Go's zero-value behavior for int. The assertion (Quantity < 0) is always false for a zero int.
func TestInventory_DefaultQuantity(t *testing.T) {
	db := setupInventoryTestDB(t)

	// Create inventory without explicit quantity
	inventory := &Inventory{
		ScryfallID: "test-id",
		OracleID:   "oracle-id",
		Treatment:  "nonfoil",
		// Quantity not set
	}
	if err := db.Create(inventory).Error; err != nil {
		t.Fatalf("failed to create inventory: %v", err)
	}

	// Verify default quantity (should be 0 from GORM, but model allows it)
	if inventory.Quantity < 0 {
		t.Errorf("expected non-negative quantity, got %d", inventory.Quantity)
	}
}

// LOW-VALUE: Only tests that GORM can save arbitrary strings to a varchar column. No validation logic exists for Treatment.
func TestInventory_Treatments(t *testing.T) {
	db := setupInventoryTestDB(t)

	treatments := []string{"nonfoil", "foil", "etched", "showcase", "extended"}

	for _, treatment := range treatments {
		t.Run(treatment, func(t *testing.T) {
			inventory := &Inventory{
				ScryfallID: "test-id-" + treatment,
				OracleID:   "oracle-id",
				Treatment:  treatment,
				Quantity:   1,
			}
			if err := db.Create(inventory).Error; err != nil {
				t.Fatalf("failed to create inventory with treatment %s: %v", treatment, err)
			}
		})
	}
}
