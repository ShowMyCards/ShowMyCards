package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupResolveTestDB builds an in-memory cards table with the generated columns
// (name, set_code, collector_number, lang) the resolution queries rely on.
func setupResolveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE cards ADD COLUMN name TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.name')) VIRTUAL`,
		`ALTER TABLE cards ADD COLUMN set_code TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.set')) VIRTUAL`,
		`ALTER TABLE cards ADD COLUMN collector_number TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.collector_number')) VIRTUAL`,
		`ALTER TABLE cards ADD COLUMN lang TEXT GENERATED ALWAYS AS (json_extract(raw_json, '$.lang')) VIRTUAL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to add generated column: %v", err)
		}
	}
	return db
}

func seedResolveCards(t *testing.T, db *gorm.DB) {
	t.Helper()
	cards := []Card{
		// Promo-stamped printing: collector number literally "33p". Scryfall's cn:
		// search normalises the suffix away, but the local column stores it verbatim.
		{ScryfallID: "vov-ptdm-33p", OracleID: "oracle-vov", RawJSON: `{"id":"vov-ptdm-33p","oracle_id":"oracle-vov","set":"ptdm","collector_number":"33p","lang":"en","name":"Voice of Victory","finishes":["foil"]}`},
		// English + German share (brr, 67); resolution must prefer the requested lang.
		{ScryfallID: "aa-brr-67", OracleID: "oracle-aa", RawJSON: `{"id":"aa-brr-67","oracle_id":"oracle-aa","set":"brr","collector_number":"67","lang":"en","name":"Ashnod's Altar","finishes":["nonfoil","foil"]}`},
		{ScryfallID: "aa-brr-67-de", OracleID: "oracle-aa", RawJSON: `{"id":"aa-brr-67-de","oracle_id":"oracle-aa","set":"brr","collector_number":"67","lang":"de","name":"Ashnods Altar DE","finishes":["foil"]}`},
		{ScryfallID: "sr-cmm-703", OracleID: "oracle-sr", RawJSON: `{"id":"sr-cmm-703","oracle_id":"oracle-sr","set":"cmm","collector_number":"703","lang":"en","name":"Sol Ring","finishes":["nonfoil","foil"]}`},
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("failed to seed cards: %v", err)
	}
}

func TestGetScryfallCardsByPrints(t *testing.T) {
	db := setupResolveTestDB(t)
	seedResolveCards(t, db)

	got, err := GetScryfallCardsByPrints(db, []PrintKey{
		{SetCode: "ptdm", CollectorNumber: "33p"},
		{SetCode: "brr", CollectorNumber: "67"},
		{SetCode: "brr", CollectorNumber: "999"}, // miss
	}, "en")
	if err != nil {
		t.Fatalf("GetScryfallCardsByPrints: %v", err)
	}

	// The promo-stamped suffix resolves by literal collector-number match.
	if card, ok := got["ptdm|33p"]; !ok || card.ID != "vov-ptdm-33p" {
		t.Errorf("expected ptdm|33p -> vov-ptdm-33p, got %+v (ok=%v)", got["ptdm|33p"], ok)
	}
	// The English printing is returned for a print shared across languages.
	if card, ok := got["brr|67"]; !ok || card.ID != "aa-brr-67" {
		t.Errorf("expected brr|67 -> aa-brr-67 (english), got %+v (ok=%v)", got["brr|67"], ok)
	}
	// A collector number that does not exist is absent (caller falls back).
	if _, ok := got["brr|999"]; ok {
		t.Errorf("expected brr|999 to be absent")
	}
}

func TestGetScryfallCardsByPrints_LanguageFilter(t *testing.T) {
	db := setupResolveTestDB(t)
	seedResolveCards(t, db)

	got, err := GetScryfallCardsByPrints(db, []PrintKey{{SetCode: "brr", CollectorNumber: "67"}}, "de")
	if err != nil {
		t.Fatalf("GetScryfallCardsByPrints: %v", err)
	}
	if card, ok := got["brr|67"]; !ok || card.ID != "aa-brr-67-de" {
		t.Errorf("expected german printing aa-brr-67-de, got %+v (ok=%v)", got["brr|67"], ok)
	}
}

func TestGetScryfallCardsByNames(t *testing.T) {
	db := setupResolveTestDB(t)
	seedResolveCards(t, db)

	got, err := GetScryfallCardsByNames(db, []string{"Sol Ring", "Nonexistent Card"}, "en")
	if err != nil {
		t.Fatalf("GetScryfallCardsByNames: %v", err)
	}
	if card, ok := got["sol ring"]; !ok || card.ID != "sr-cmm-703" {
		t.Errorf("expected sol ring -> sr-cmm-703, got %+v (ok=%v)", got["sol ring"], ok)
	}
	if _, ok := got["nonexistent card"]; ok {
		t.Errorf("expected nonexistent card to be absent")
	}
}
