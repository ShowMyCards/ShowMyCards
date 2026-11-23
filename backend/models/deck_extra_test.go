package models

import (
	"context"
	"fmt"
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
)

func TestIsDeckExtra(t *testing.T) {
	face := func(typeLine string) scryfall.CardFace { return scryfall.CardFace{TypeLine: typeLine} }

	tests := []struct {
		name string
		card scryfall.Card
		want bool
	}{
		// Extras by layout.
		{"token", scryfall.Card{Layout: scryfall.LayoutToken, TypeLine: "Token Creature — Rabbit"}, true},
		{"double-faced token (Day // Night)", scryfall.Card{Layout: scryfall.LayoutDoubleFacedToken, TypeLine: "Card // Card"}, true},
		{"emblem", scryfall.Card{Layout: scryfall.LayoutEmblem, TypeLine: "Emblem — Elspeth"}, true},
		{"art series", scryfall.Card{Layout: scryfall.LayoutArtSeries, TypeLine: "Card // Card"}, true},
		{"plane", scryfall.Card{Layout: scryfall.LayoutPlanar, TypeLine: "Plane — Dominaria"}, true},
		{"phenomenon", scryfall.Card{Layout: scryfall.LayoutPlanar, TypeLine: "Phenomenon"}, true},
		{"scheme", scryfall.Card{Layout: scryfall.LayoutScheme, TypeLine: "Scheme"}, true},
		{"vanguard", scryfall.Card{Layout: scryfall.LayoutVanguard, TypeLine: "Vanguard"}, true},
		{"jumpstart front card", scryfall.Card{Layout: scryfall.Layout("front_card"), TypeLine: "Card"}, true},
		{"the monarch", scryfall.Card{Layout: scryfall.LayoutToken, TypeLine: "Card"}, true},

		// Extras by type line, whatever the layout says.
		{"role token with flip layout", scryfall.Card{
			Layout:   scryfall.LayoutFlip,
			TypeLine: "Token Enchantment — Aura Role // Token Enchantment — Aura Role",
		}, true},
		{"helper card with normal layout", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Card"}, true},
		{"dungeon with normal layout", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Dungeon — Undercity"}, true},
		{"reversible token (type lines only on faces)", scryfall.Card{
			Layout:    scryfall.LayoutReversible,
			CardFaces: []scryfall.CardFace{face("Token Legendary Artifact Creature — Construct"), face("Token Legendary Artifact Creature — Construct")},
		}, true},

		// Real deck cards.
		{"creature", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Creature — Rabbit Soldier"}, false},
		{"artifact creature", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Artifact Creature — Golem"}, false},
		{"kindred instant", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Kindred Instant — Faerie"}, false},
		{"conspiracy", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Conspiracy"}, false},
		{"saga", scryfall.Card{Layout: scryfall.LayoutSaga, TypeLine: "Enchantment — Saga"}, false},
		{"mdfc", scryfall.Card{
			Layout:    scryfall.LayoutModalDFC,
			TypeLine:  "Sorcery // Land",
			CardFaces: []scryfall.CardFace{face("Sorcery"), face("Land")},
		}, false},
		{"reversible real card", scryfall.Card{
			Layout:    scryfall.LayoutReversible,
			CardFaces: []scryfall.CardFace{face("Legendary Creature — Elf"), face("Legendary Creature — Elf")},
		}, false},
		// "Token"/"Card" as a subtype word after the dash must not count.
		{"subtype mentions card", scryfall.Card{Layout: scryfall.LayoutNormal, TypeLine: "Creature — Card Shark"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeckExtra(tt.card); got != tt.want {
				t.Errorf("IsDeckExtra() = %v, want %v", got, tt.want)
			}
		})
	}
}

// setupDeckExtraTestDB builds a database with the cards table (and the generated
// columns the representative-printing query needs) plus decks and deck items.
func setupDeckExtraTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupResolveTestDB(t)
	if err := db.AutoMigrate(&Deck{}, &DeckItem{}); err != nil {
		t.Fatalf("failed to migrate deck tables: %v", err)
	}
	return db
}

func seedExtraTestCard(t *testing.T, db *gorm.DB, scryfallID, oracleID, layout, typeLine string) {
	t.Helper()
	raw := fmt.Sprintf(`{"id":%q,"oracle_id":%q,"name":%q,"set":"tst","collector_number":%q,"lang":"en","layout":%q,"type_line":%q}`,
		scryfallID, oracleID, scryfallID, scryfallID, layout, typeLine)
	if err := db.Create(&Card{ScryfallID: scryfallID, OracleID: oracleID, RawJSON: raw}).Error; err != nil {
		t.Fatalf("failed to seed card %s: %v", scryfallID, err)
	}
}

func seedExtraTestItem(t *testing.T, db *gorm.DB, deckID uint, oracleID, scryfallID string, zone DeckZone, qty int) DeckItem {
	t.Helper()
	item := DeckItem{DeckID: deckID, OracleID: oracleID, ScryfallID: scryfallID, Zone: zone, DesiredQuantity: qty}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed deck item: %v", err)
	}
	return item
}

func TestClassifyDeckItems(t *testing.T) {
	db := setupDeckExtraTestDB(t)
	seedExtraTestCard(t, db, "rabbit-1", "oracle-rabbit", "token", "Token Creature — Rabbit")
	seedExtraTestCard(t, db, "bolt-1", "oracle-bolt", "normal", "Instant")

	items := []DeckItem{
		{OracleID: "oracle-rabbit", ScryfallID: "rabbit-1"},         // pinned token
		{OracleID: "oracle-rabbit"},                                 // any-printing token
		{OracleID: "oracle-bolt", ScryfallID: "bolt-1"},             // pinned real card
		{OracleID: "oracle-bolt", ScryfallID: "bolt-missing"},       // pinned printing missing: fall back to the Oracle
		{OracleID: "oracle-unknown"},                                // no card data at all
		{OracleID: "oracle-unknown", ScryfallID: "unknown-missing"}, // no card data at all
	}

	got, err := ClassifyDeckItems(db, items)
	if err != nil {
		t.Fatalf("ClassifyDeckItems: %v", err)
	}

	want := map[int]bool{0: true, 1: true, 2: false, 3: false}
	if len(got) != len(want) {
		t.Fatalf("got %d classified items %v, want %d %v", len(got), got, len(want), want)
	}
	for i, w := range want {
		if g, ok := got[i]; !ok || g != w {
			t.Errorf("item %d: got (%v, present=%v), want %v", i, g, ok, w)
		}
	}
}

func deckZoneRows(t *testing.T, db *gorm.DB, deckID uint) map[string]int {
	t.Helper()
	var items []DeckItem
	if err := db.Where("deck_id = ?", deckID).Find(&items).Error; err != nil {
		t.Fatalf("failed to load deck items: %v", err)
	}
	rows := make(map[string]int, len(items))
	for _, it := range items {
		rows[it.OracleID+"@"+string(it.Zone)] += it.DesiredQuantity
	}
	return rows
}

func TestReclassifyDeckExtras(t *testing.T) {
	db := setupDeckExtraTestDB(t)
	seedExtraTestCard(t, db, "rabbit-1", "oracle-rabbit", "token", "Token Creature — Rabbit")
	seedExtraTestCard(t, db, "role-1", "oracle-role", "flip", "Token Enchantment — Aura Role // Token Enchantment — Aura Role")
	seedExtraTestCard(t, db, "bolt-1", "oracle-bolt", "normal", "Instant")

	deck := Deck{Name: "Rabbits"}
	if err := db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}
	// Same token printing in main and side: both move to other and merge.
	seedExtraTestItem(t, db, deck.ID, "oracle-rabbit", "rabbit-1", ZoneMain, 2)
	seedExtraTestItem(t, db, deck.ID, "oracle-rabbit", "rabbit-1", ZoneSide, 1)
	// Any-printing token on the maybe-board.
	seedExtraTestItem(t, db, deck.ID, "oracle-role", "", ZoneMaybe, 1)
	// A real card wrongly in other goes back to main.
	seedExtraTestItem(t, db, deck.ID, "oracle-bolt", "bolt-1", ZoneOther, 4)
	// No card data: left where it is.
	seedExtraTestItem(t, db, deck.ID, "oracle-unknown", "", ZoneMain, 1)

	result, err := ReclassifyDeckExtras(context.Background(), db)
	if err != nil {
		t.Fatalf("ReclassifyDeckExtras: %v", err)
	}
	if result.Checked != 5 || result.Moved != 4 {
		t.Errorf("result = %+v, want Checked 5, Moved 4", result)
	}

	want := map[string]int{
		"oracle-rabbit@other": 3,
		"oracle-role@other":   1,
		"oracle-bolt@main":    4,
		"oracle-unknown@main": 1,
	}
	got := deckZoneRows(t, db, deck.ID)
	if len(got) != len(want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %d, want %d (all rows %v)", k, got[k], w, got)
		}
	}

	// Idempotent: a second run moves nothing.
	again, err := ReclassifyDeckExtras(context.Background(), db)
	if err != nil {
		t.Fatalf("second ReclassifyDeckExtras: %v", err)
	}
	if again.Moved != 0 {
		t.Errorf("second run moved %d rows, want 0", again.Moved)
	}
}

func TestReclassifyDeckExtras_MergesIntoExistingOtherRow(t *testing.T) {
	db := setupDeckExtraTestDB(t)
	seedExtraTestCard(t, db, "rabbit-1", "oracle-rabbit", "token", "Token Creature — Rabbit")

	deck := Deck{Name: "Rabbits"}
	if err := db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}
	seedExtraTestItem(t, db, deck.ID, "oracle-rabbit", "rabbit-1", ZoneOther, 1)
	seedExtraTestItem(t, db, deck.ID, "oracle-rabbit", "rabbit-1", ZoneMain, 2)

	if _, err := ReclassifyDeckExtras(context.Background(), db); err != nil {
		t.Fatalf("ReclassifyDeckExtras: %v", err)
	}

	got := deckZoneRows(t, db, deck.ID)
	if len(got) != 1 || got["oracle-rabbit@other"] != 3 {
		t.Errorf("rows = %v, want only oracle-rabbit@other = 3", got)
	}
}

func TestReclassifyDeckExtras_EmptyDatabase(t *testing.T) {
	db := setupDeckExtraTestDB(t)
	result, err := ReclassifyDeckExtras(context.Background(), db)
	if err != nil {
		t.Fatalf("ReclassifyDeckExtras: %v", err)
	}
	if result.Checked != 0 || result.Moved != 0 {
		t.Errorf("result = %+v, want zero", result)
	}
}

func TestCheckExtraZone(t *testing.T) {
	tests := []struct {
		name    string
		isExtra bool
		zone    DeckZone
		wantErr error
	}{
		{"extra in other", true, ZoneOther, nil},
		{"extra in main", true, ZoneMain, ErrExtraOutsideOther},
		{"extra in maybe", true, ZoneMaybe, ErrExtraOutsideOther},
		{"card in main", false, ZoneMain, nil},
		{"card in other", false, ZoneOther, ErrCardInOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckExtraZone(tt.isExtra, tt.zone); err != tt.wantErr {
				t.Errorf("CheckExtraZone(%v, %q) = %v, want %v", tt.isExtra, tt.zone, err, tt.wantErr)
			}
		})
	}
}
