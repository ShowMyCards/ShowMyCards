package models

import (
	"fmt"
	"testing"
)

func TestDeckZone_IsValid(t *testing.T) {
	tests := []struct {
		zone DeckZone
		want bool
	}{
		{ZoneMain, true},
		{ZoneSide, true},
		{ZoneCommand, true},
		{ZoneMaybe, true},
		{DeckZone(""), false},
		{DeckZone("unknown"), false},
		{DeckZone("MAIN"), false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.zone), func(t *testing.T) {
			if got := tt.zone.IsValid(); got != tt.want {
				t.Errorf("DeckZone(%q).IsValid() = %v, want %v", tt.zone, got, tt.want)
			}
		})
	}
}

func TestDeckZone_CountsAsDemand(t *testing.T) {
	tests := []struct {
		zone DeckZone
		want bool
	}{
		{ZoneMain, true},
		{ZoneSide, true},
		{ZoneCommand, true},
		{ZoneMaybe, false}, // critical: maybe-board does NOT count as demand
		{DeckZone(""), false},
		{DeckZone("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.zone), func(t *testing.T) {
			if got := tt.zone.CountsAsDemand(); got != tt.want {
				t.Errorf("DeckZone(%q).CountsAsDemand() = %v, want %v", tt.zone, got, tt.want)
			}
		})
	}
}

func TestDeckItem_ValidateDeckItem(t *testing.T) {
	db := setupDeckTestDB(t)

	tests := []struct {
		name        string
		item        *DeckItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Item",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				ScryfallID:      "scry-id",
				Treatment:       "nonfoil",
				Zone:            ZoneMain,
				DesiredQuantity: 4,
			},
			expectError: false,
		},
		{
			name: "Valid Item - Any Printing (empty ScryfallID and Treatment)",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				ScryfallID:      "",
				Treatment:       "",
				Zone:            ZoneMain,
				DesiredQuantity: 1,
			},
			expectError: false,
		},
		{
			name: "Invalid - Empty OracleID",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "",
				Zone:            ZoneMain,
				DesiredQuantity: 1,
			},
			expectError: true,
			errorMsg:    "oracle_id cannot be empty",
		},
		{
			name: "Invalid - Zero DesiredQuantity",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				Zone:            ZoneMain,
				DesiredQuantity: 0,
			},
			expectError: true,
			errorMsg:    "desired_quantity must be at least 1",
		},
		{
			name: "Invalid - Negative DesiredQuantity",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				Zone:            ZoneMain,
				DesiredQuantity: -1,
			},
			expectError: true,
			errorMsg:    "desired_quantity must be at least 1",
		},
		{
			name: "Invalid - Unknown Zone",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				Zone:            DeckZone("invalid"),
				DesiredQuantity: 1,
			},
			expectError: true,
			errorMsg:    "zone must be one of main, side, command, maybe",
		},
		{
			name: "Invalid - Empty Zone",
			item: &DeckItem{
				DeckID:          1,
				OracleID:        "oracle-id",
				Zone:            DeckZone(""),
				DesiredQuantity: 1,
			},
			expectError: true,
			errorMsg:    "zone must be one of main, side, command, maybe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.ValidateDeckItem(db)
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

func TestDeckItem_BeforeCreate(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	tests := []struct {
		name        string
		item        *DeckItem
		expectError bool
	}{
		{
			name: "Valid Create",
			item: &DeckItem{
				DeckID:          deck.ID,
				OracleID:        "oracle-id",
				Zone:            ZoneMain,
				DesiredQuantity: 4,
			},
			expectError: false,
		},
		{
			name: "Invalid Create - Empty OracleID",
			item: &DeckItem{
				DeckID:          deck.ID,
				OracleID:        "",
				Zone:            ZoneMain,
				DesiredQuantity: 1,
			},
			expectError: true,
		},
		{
			name: "Invalid Create - Bad Zone",
			item: &DeckItem{
				DeckID:          deck.ID,
				OracleID:        "oracle-id",
				Zone:            DeckZone("garbage"),
				DesiredQuantity: 1,
			},
			expectError: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a unique oracle_id per case to dodge the composite index
			// when several valid rows would otherwise collide.
			if tt.item.OracleID != "" {
				tt.item.OracleID = fmt.Sprintf("%s-%d", tt.item.OracleID, i)
			}
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

func TestDeckItem_BeforeUpdate(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	tests := []struct {
		name        string
		updateFunc  func(*DeckItem)
		expectError bool
	}{
		{
			name: "Valid Update - DesiredQuantity",
			updateFunc: func(item *DeckItem) {
				item.DesiredQuantity = 3
			},
			expectError: false,
		},
		{
			name: "Valid Update - Zone",
			updateFunc: func(item *DeckItem) {
				item.Zone = ZoneSide
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Zero DesiredQuantity",
			updateFunc: func(item *DeckItem) {
				item.DesiredQuantity = 0
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Bad Zone",
			updateFunc: func(item *DeckItem) {
				item.Zone = DeckZone("garbage")
			},
			expectError: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testItem := &DeckItem{
				DeckID:          deck.ID,
				OracleID:        fmt.Sprintf("oracle-update-%d", i),
				Zone:            ZoneMain,
				DesiredQuantity: 2,
			}
			if err := db.Create(testItem).Error; err != nil {
				t.Fatalf("failed to create deck item: %v", err)
			}

			tt.updateFunc(testItem)

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

// TestDeckItem_UniqueConstraint exercises the five-column composite index
// idx_deck_card over (deck_id, oracle_id, scryfall_id, treatment, zone).
//
// The empty-string handling for ScryfallID and Treatment is load-bearing
// here: SQLite treats NULLs as distinct in a unique index, so storing those
// fields as "" rather than NULL is what lets the constraint reject duplicate
// "any printing" rows.
func TestDeckItem_UniqueConstraint(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	// Baseline: any-printing, nonfoil, main, x4.
	base := &DeckItem{
		DeckID:          deck.ID,
		OracleID:        "oracle-A",
		ScryfallID:      "",
		Treatment:       "",
		Zone:            ZoneMain,
		DesiredQuantity: 4,
	}
	if err := db.Create(base).Error; err != nil {
		t.Fatalf("failed to create base item: %v", err)
	}

	t.Run("exact duplicate rejected", func(t *testing.T) {
		dup := &DeckItem{
			DeckID:          deck.ID,
			OracleID:        "oracle-A",
			ScryfallID:      "",
			Treatment:       "",
			Zone:            ZoneMain,
			DesiredQuantity: 1,
		}
		if err := db.Create(dup).Error; err == nil {
			t.Error("expected unique constraint error for exact duplicate, got none")
		}
	})

	t.Run("different zone allowed", func(t *testing.T) {
		other := &DeckItem{
			DeckID:          deck.ID,
			OracleID:        "oracle-A",
			ScryfallID:      "",
			Treatment:       "",
			Zone:            ZoneSide,
			DesiredQuantity: 1,
		}
		if err := db.Create(other).Error; err != nil {
			t.Errorf("expected different zone to succeed, got error: %v", err)
		}
	})

	t.Run("different scryfall_id allowed", func(t *testing.T) {
		other := &DeckItem{
			DeckID:          deck.ID,
			OracleID:        "oracle-A",
			ScryfallID:      "scry-specific",
			Treatment:       "",
			Zone:            ZoneMain,
			DesiredQuantity: 1,
		}
		if err := db.Create(other).Error; err != nil {
			t.Errorf("expected different scryfall_id to succeed, got error: %v", err)
		}
	})

	t.Run("different treatment allowed", func(t *testing.T) {
		other := &DeckItem{
			DeckID:          deck.ID,
			OracleID:        "oracle-A",
			ScryfallID:      "",
			Treatment:       "foil",
			Zone:            ZoneMain,
			DesiredQuantity: 1,
		}
		if err := db.Create(other).Error; err != nil {
			t.Errorf("expected different treatment to succeed, got error: %v", err)
		}
	})

	t.Run("different deck allowed", func(t *testing.T) {
		otherDeck := &Deck{Name: "Second Deck"}
		if err := db.Create(otherDeck).Error; err != nil {
			t.Fatalf("failed to create second deck: %v", err)
		}
		other := &DeckItem{
			DeckID:          otherDeck.ID,
			OracleID:        "oracle-A",
			ScryfallID:      "",
			Treatment:       "",
			Zone:            ZoneMain,
			DesiredQuantity: 1,
		}
		if err := db.Create(other).Error; err != nil {
			t.Errorf("expected different deck to succeed, got error: %v", err)
		}
	})
}

func TestDeckItem_DefaultDesiredQuantity(t *testing.T) {
	db := setupDeckTestDB(t)

	deck := &Deck{Name: "Test Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}

	// Validation enforces DesiredQuantity >= 1 before the column default
	// kicks in, so an unset DesiredQuantity is rejected at the hook layer.
	// This test pins that behaviour so a future change to the default does
	// not accidentally weaken the validation contract.
	item := &DeckItem{
		DeckID:   deck.ID,
		OracleID: "oracle-default",
		Zone:     ZoneMain,
	}
	if err := db.Create(item).Error; err == nil {
		t.Error("expected validation error when DesiredQuantity is unset, got none")
	}
}
