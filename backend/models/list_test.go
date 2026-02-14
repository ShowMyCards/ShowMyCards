package models

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupListTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&List{}, &ListItem{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

// List tests

func TestList_ValidateList(t *testing.T) {
	db := setupListTestDB(t)

	tests := []struct {
		name        string
		list        *List
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid List",
			list:        &List{Name: "My Deck"},
			expectError: false,
		},
		{
			name:        "Valid List with Description",
			list:        &List{Name: "Wishlist", Description: "Cards I want to buy"},
			expectError: false,
		},
		{
			name:        "Invalid - Empty Name",
			list:        &List{Name: ""},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.ValidateList(db)
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

func TestList_BeforeCreate(t *testing.T) {
	db := setupListTestDB(t)

	tests := []struct {
		name        string
		list        *List
		expectError bool
	}{
		{
			name:        "Valid Create",
			list:        &List{Name: "Test List"},
			expectError: false,
		},
		{
			name:        "Invalid Create - Empty Name",
			list:        &List{Name: ""},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.list).Error
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

func TestList_BeforeUpdate(t *testing.T) {
	db := setupListTestDB(t)

	tests := []struct {
		name        string
		updateFunc  func(*List)
		expectError bool
	}{
		{
			name: "Valid Update - Name",
			updateFunc: func(l *List) {
				l.Name = "Updated Name"
			},
			expectError: false,
		},
		{
			name: "Valid Update - Description",
			updateFunc: func(l *List) {
				l.Description = "Updated Description"
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Empty Name",
			updateFunc: func(l *List) {
				l.Name = ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh list for each test
			testList := &List{Name: "Original Name"}
			if err := db.Create(testList).Error; err != nil {
				t.Fatalf("failed to create list: %v", err)
			}

			// Apply update
			tt.updateFunc(testList)

			// Try to save
			err := db.Save(testList).Error
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

func TestList_WithItems(t *testing.T) {
	db := setupListTestDB(t)

	// Create list
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	// Create items
	items := []*ListItem{
		{
			ListID:            list.ID,
			ScryfallID:        "id-1",
			OracleID:          "oracle-1",
			Treatment:         "nonfoil",
			DesiredQuantity:   4,
			CollectedQuantity: 2,
		},
		{
			ListID:            list.ID,
			ScryfallID:        "id-2",
			OracleID:          "oracle-2",
			Treatment:         "foil",
			DesiredQuantity:   1,
			CollectedQuantity: 0,
		},
	}

	for _, item := range items {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("failed to create list item: %v", err)
		}
	}

	// Load list with items
	var loadedList List
	if err := db.Preload("Items").First(&loadedList, list.ID).Error; err != nil {
		t.Fatalf("failed to load list: %v", err)
	}

	// Verify items
	if len(loadedList.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(loadedList.Items))
	}
}

// ListItem tests

func TestListItem_ValidateListItem(t *testing.T) {
	db := setupListTestDB(t)

	tests := []struct {
		name        string
		item        *ListItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Item",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 2,
			},
			expectError: false,
		},
		{
			name: "Valid Item - Zero Collected",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			},
			expectError: false,
		},
		{
			name: "Valid Item - Fully Collected",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 4,
			},
			expectError: false,
		},
		{
			name: "Invalid - Empty ScryfallID",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			},
			expectError: true,
			errorMsg:    "scryfall_id cannot be empty",
		},
		{
			name: "Invalid - Empty OracleID",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			},
			expectError: true,
			errorMsg:    "oracle_id cannot be empty",
		},
		{
			name: "Invalid - Zero DesiredQuantity",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   0,
				CollectedQuantity: 0,
			},
			expectError: true,
			errorMsg:    "desired_quantity must be at least 1",
		},
		{
			name: "Invalid - Negative CollectedQuantity",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: -1,
			},
			expectError: true,
			errorMsg:    "collected_quantity cannot be negative",
		},
		{
			name: "Invalid - Collected Exceeds Desired",
			item: &ListItem{
				ListID:            1,
				ScryfallID:        "test-id",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 5,
			},
			expectError: true,
			errorMsg:    "collected_quantity cannot exceed desired_quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.ValidateListItem(db)
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

func TestListItem_BeforeCreate(t *testing.T) {
	db := setupListTestDB(t)

	// Create a list first
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	tests := []struct {
		name        string
		item        *ListItem
		expectError bool
	}{
		{
			name: "Valid Create",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "test-id-1",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			},
			expectError: false,
		},
		{
			name: "Invalid Create - Collected Exceeds Desired",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "test-id-2",
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   2,
				CollectedQuantity: 3,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.item).Error
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

func TestListItem_BeforeUpdate(t *testing.T) {
	db := setupListTestDB(t)

	// Create list
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	tests := []struct {
		name        string
		updateFunc  func(*ListItem)
		expectError bool
	}{
		{
			name: "Valid Update - Increase Collected",
			updateFunc: func(item *ListItem) {
				item.CollectedQuantity = 3
			},
			expectError: false,
		},
		{
			name: "Valid Update - Increase Desired",
			updateFunc: func(item *ListItem) {
				item.DesiredQuantity = 8
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Collected Exceeds Desired",
			updateFunc: func(item *ListItem) {
				item.CollectedQuantity = 5
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Zero Desired",
			updateFunc: func(item *ListItem) {
				item.DesiredQuantity = 0
			},
			expectError: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh item for each test with unique ID
			testItem := &ListItem{
				ListID:            list.ID,
				ScryfallID:        fmt.Sprintf("test-id-%d", i),
				OracleID:          "oracle-id",
				Treatment:         "nonfoil",
				DesiredQuantity:   4,
				CollectedQuantity: 2,
			}
			if err := db.Create(testItem).Error; err != nil {
				t.Fatalf("failed to create list item: %v", err)
			}

			// Apply update
			tt.updateFunc(testItem)

			// Try to save
			err := db.Save(testItem).Error
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

func TestListItem_UniqueConstraint(t *testing.T) {
	db := setupListTestDB(t)

	// Create list
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	// Create first item
	item1 := &ListItem{
		ListID:            list.ID,
		ScryfallID:        "duplicate-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		DesiredQuantity:   4,
		CollectedQuantity: 0,
	}
	if err := db.Create(item1).Error; err != nil {
		t.Fatalf("failed to create first item: %v", err)
	}

	// Try to create duplicate (same list, scryfall_id, treatment)
	item2 := &ListItem{
		ListID:            list.ID,
		ScryfallID:        "duplicate-id",
		OracleID:          "oracle-id",
		Treatment:         "nonfoil",
		DesiredQuantity:   2,
		CollectedQuantity: 0,
	}
	err := db.Create(item2).Error
	if err == nil {
		t.Error("expected unique constraint error, got none")
	}

	// But different treatment should work
	item3 := &ListItem{
		ListID:            list.ID,
		ScryfallID:        "duplicate-id",
		OracleID:          "oracle-id",
		Treatment:         "foil",
		DesiredQuantity:   1,
		CollectedQuantity: 0,
	}
	if err := db.Create(item3).Error; err != nil {
		t.Errorf("expected different treatment to succeed, got error: %v", err)
	}
}

func TestListItem_DefaultCollectedQuantity(t *testing.T) {
	db := setupListTestDB(t)

	// Create list
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	// Create item without CollectedQuantity
	item := &ListItem{
		ListID:          list.ID,
		ScryfallID:      "test-id",
		OracleID:        "oracle-id",
		Treatment:       "nonfoil",
		DesiredQuantity: 4,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// Verify default is 0
	if item.CollectedQuantity != 0 {
		t.Errorf("expected CollectedQuantity to default to 0, got %d", item.CollectedQuantity)
	}
}
