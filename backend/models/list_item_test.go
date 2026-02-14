package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupListItemTestDB(t *testing.T) (*gorm.DB, *List) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&List{}, &ListItem{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Create a parent list for foreign key references
	list := &List{Name: "Test List"}
	if err := db.Create(list).Error; err != nil {
		t.Fatalf("failed to create parent list: %v", err)
	}

	return db, list
}

func TestListItem_BeforeCreate_ValidatesDesiredQuantity(t *testing.T) {
	db, list := setupListItemTestDB(t)

	tests := []struct {
		name        string
		item        *ListItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "DesiredQuantity of 1 is valid",
			item: &ListItem{
				ListID:          list.ID,
				ScryfallID:      "scry-1",
				OracleID:        "oracle-1",
				DesiredQuantity: 1,
			},
			expectError: false,
		},
		{
			name: "DesiredQuantity of 4 is valid",
			item: &ListItem{
				ListID:          list.ID,
				ScryfallID:      "scry-2",
				OracleID:        "oracle-2",
				DesiredQuantity: 4,
			},
			expectError: false,
		},
		{
			name: "DesiredQuantity of 0 is invalid",
			item: &ListItem{
				ListID:          list.ID,
				ScryfallID:      "scry-3",
				OracleID:        "oracle-3",
				DesiredQuantity: 0,
			},
			expectError: true,
			errorMsg:    "desired_quantity must be at least 1",
		},
		{
			name: "DesiredQuantity of -1 is invalid",
			item: &ListItem{
				ListID:          list.ID,
				ScryfallID:      "scry-4",
				OracleID:        "oracle-4",
				DesiredQuantity: -1,
			},
			expectError: true,
			errorMsg:    "desired_quantity must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.item).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
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

func TestListItem_BeforeCreate_ValidatesCollectedQuantity(t *testing.T) {
	db, list := setupListItemTestDB(t)

	tests := []struct {
		name        string
		item        *ListItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "CollectedQuantity of 0 is valid",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-cq-1",
				OracleID:          "oracle-cq-1",
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			},
			expectError: false,
		},
		{
			name: "CollectedQuantity of 2 with DesiredQuantity 4 is valid",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-cq-2",
				OracleID:          "oracle-cq-2",
				DesiredQuantity:   4,
				CollectedQuantity: 2,
			},
			expectError: false,
		},
		{
			name: "CollectedQuantity equal to DesiredQuantity is valid",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-cq-3",
				OracleID:          "oracle-cq-3",
				DesiredQuantity:   3,
				CollectedQuantity: 3,
			},
			expectError: false,
		},
		{
			name: "Negative CollectedQuantity is invalid",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-cq-4",
				OracleID:          "oracle-cq-4",
				DesiredQuantity:   4,
				CollectedQuantity: -1,
			},
			expectError: true,
			errorMsg:    "collected_quantity cannot be negative",
		},
		{
			name: "CollectedQuantity exceeding DesiredQuantity is invalid",
			item: &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-cq-5",
				OracleID:          "oracle-cq-5",
				DesiredQuantity:   2,
				CollectedQuantity: 3,
			},
			expectError: true,
			errorMsg:    "collected_quantity cannot exceed desired_quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.item).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
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

func TestListItem_BeforeUpdate_SameValidations(t *testing.T) {
	db, list := setupListItemTestDB(t)

	tests := []struct {
		name        string
		updateFunc  func(*ListItem)
		expectError bool
	}{
		{
			name: "Valid Update - Increase DesiredQuantity",
			updateFunc: func(li *ListItem) {
				li.DesiredQuantity = 5
			},
			expectError: false,
		},
		{
			name: "Valid Update - Increase CollectedQuantity within bounds",
			updateFunc: func(li *ListItem) {
				li.CollectedQuantity = 2
			},
			expectError: false,
		},
		{
			name: "Invalid Update - DesiredQuantity to 0",
			updateFunc: func(li *ListItem) {
				li.DesiredQuantity = 0
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Negative CollectedQuantity",
			updateFunc: func(li *ListItem) {
				li.CollectedQuantity = -1
			},
			expectError: true,
		},
		{
			name: "Invalid Update - CollectedQuantity exceeds DesiredQuantity",
			updateFunc: func(li *ListItem) {
				li.CollectedQuantity = 10
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh item for each test
			item := &ListItem{
				ListID:            list.ID,
				ScryfallID:        "scry-update-" + tt.name,
				OracleID:          "oracle-update-" + tt.name,
				DesiredQuantity:   4,
				CollectedQuantity: 0,
			}
			if err := db.Create(item).Error; err != nil {
				t.Fatalf("failed to create list item: %v", err)
			}

			// Apply update
			tt.updateFunc(item)

			// Try to save (triggers BeforeUpdate)
			err := db.Save(item).Error
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

func TestListItem_CreateSuccess(t *testing.T) {
	db, list := setupListItemTestDB(t)

	item := &ListItem{
		ListID:            list.ID,
		ScryfallID:        "scry-success",
		OracleID:          "oracle-success",
		Treatment:         "foil",
		DesiredQuantity:   2,
		CollectedQuantity: 1,
	}

	if err := db.Create(item).Error; err != nil {
		t.Fatalf("failed to create list item: %v", err)
	}

	if item.ID == 0 {
		t.Error("expected ID to be set after create")
	}
	if item.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after create")
	}

	// Verify retrieval
	var retrieved ListItem
	if err := db.First(&retrieved, item.ID).Error; err != nil {
		t.Fatalf("failed to retrieve list item: %v", err)
	}

	if retrieved.ScryfallID != "scry-success" {
		t.Errorf("expected scryfall_id %q, got %q", "scry-success", retrieved.ScryfallID)
	}
	if retrieved.OracleID != "oracle-success" {
		t.Errorf("expected oracle_id %q, got %q", "oracle-success", retrieved.OracleID)
	}
	if retrieved.Treatment != "foil" {
		t.Errorf("expected treatment %q, got %q", "foil", retrieved.Treatment)
	}
	if retrieved.DesiredQuantity != 2 {
		t.Errorf("expected desired_quantity 2, got %d", retrieved.DesiredQuantity)
	}
	if retrieved.CollectedQuantity != 1 {
		t.Errorf("expected collected_quantity 1, got %d", retrieved.CollectedQuantity)
	}
	if retrieved.ListID != list.ID {
		t.Errorf("expected list_id %d, got %d", list.ID, retrieved.ListID)
	}
}

func TestListItem_ValidateListItem_EmptyScryfallID(t *testing.T) {
	db, list := setupListItemTestDB(t)

	item := &ListItem{
		ListID:          list.ID,
		ScryfallID:      "",
		OracleID:        "oracle-1",
		DesiredQuantity: 1,
	}

	err := db.Create(item).Error
	if err == nil {
		t.Error("expected error for empty scryfall_id, got none")
	}
}

func TestListItem_ValidateListItem_EmptyOracleID(t *testing.T) {
	db, list := setupListItemTestDB(t)

	item := &ListItem{
		ListID:          list.ID,
		ScryfallID:      "scry-1",
		OracleID:        "",
		DesiredQuantity: 1,
	}

	err := db.Create(item).Error
	if err == nil {
		t.Error("expected error for empty oracle_id, got none")
	}
}
