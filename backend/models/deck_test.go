package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDeckTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&Deck{}, &DeckItem{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestDeck_ValidateDeck(t *testing.T) {
	db := setupDeckTestDB(t)

	tests := []struct {
		name        string
		deck        *Deck
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Deck",
			deck:        &Deck{Name: "Commander - Atraxa"},
			expectError: false,
		},
		{
			name:        "Valid Deck with Description",
			deck:        &Deck{Name: "Modern Burn", Description: "Aggressive red deck for Modern"},
			expectError: false,
		},
		{
			name:        "Invalid - Empty Name",
			deck:        &Deck{Name: ""},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.deck.ValidateDeck(db)
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

func TestDeck_BeforeCreate(t *testing.T) {
	db := setupDeckTestDB(t)

	tests := []struct {
		name        string
		deck        *Deck
		expectError bool
	}{
		{
			name:        "Valid Create",
			deck:        &Deck{Name: "Test Deck"},
			expectError: false,
		},
		{
			name:        "Invalid Create - Empty Name",
			deck:        &Deck{Name: ""},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.deck).Error
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

func TestDeck_BeforeUpdate(t *testing.T) {
	db := setupDeckTestDB(t)

	tests := []struct {
		name        string
		updateFunc  func(*Deck)
		expectError bool
	}{
		{
			name: "Valid Update - Name",
			updateFunc: func(d *Deck) {
				d.Name = "Updated Name"
			},
			expectError: false,
		},
		{
			name: "Valid Update - Description",
			updateFunc: func(d *Deck) {
				d.Description = "Updated Description"
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Empty Name",
			updateFunc: func(d *Deck) {
				d.Name = ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDeck := &Deck{Name: "Original Name"}
			if err := db.Create(testDeck).Error; err != nil {
				t.Fatalf("failed to create deck: %v", err)
			}

			tt.updateFunc(testDeck)

			err := db.Save(testDeck).Error
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

func TestDeck_WithItems(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	items := []*DeckItem{
		{
			DeckID:          deck.ID,
			OracleID:        "oracle-1",
			ScryfallID:      "scry-1",
			Treatment:       "nonfoil",
			Zone:            ZoneMain,
			DesiredQuantity: 4,
		},
		{
			DeckID:          deck.ID,
			OracleID:        "oracle-2",
			ScryfallID:      "",
			Treatment:       "",
			Zone:            ZoneSide,
			DesiredQuantity: 2,
		},
	}

	for _, item := range items {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("failed to create deck item: %v", err)
		}
	}

	var loadedDeck Deck
	if err := db.Preload("Items").First(&loadedDeck, deck.ID).Error; err != nil {
		t.Fatalf("failed to load deck: %v", err)
	}

	if len(loadedDeck.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(loadedDeck.Items))
	}
}

func TestDeck_CascadeDelete(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	item := &DeckItem{
		DeckID:          deck.ID,
		OracleID:        "oracle-1",
		Zone:            ZoneMain,
		DesiredQuantity: 1,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("failed to create deck item: %v", err)
	}

	// Delete via the items-first transaction pattern the handler will use.
	// The GORM CASCADE constraint also exists, but handlers delete explicitly
	// inside a transaction for atomicity.
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("deck_id = ?", deck.ID).Delete(&DeckItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(deck).Error
	})
	if err != nil {
		t.Fatalf("failed to delete deck: %v", err)
	}

	var itemCount int64
	if err := db.Model(&DeckItem{}).Where("deck_id = ?", deck.ID).Count(&itemCount).Error; err != nil {
		t.Fatalf("failed to count items: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("expected 0 items after deck delete, got %d", itemCount)
	}
}
