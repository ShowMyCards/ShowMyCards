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
	if err := db.AutoMigrate(&Card{}, &Set{}, &StorageLocation{}, &Inventory{}); err != nil {
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

// seedPrintingVariety adds several printings of one card ("oracle-op"), each
// with a treatment or set type that should rank below a plain printing.
func seedPrintingVariety(t *testing.T, db *gorm.DB) {
	t.Helper()
	sets := []Set{
		{ScryfallID: "set-mh3", Code: "mh3", Name: "Modern Horizons 3", SetType: "draft_innovation"},
		{ScryfallID: "set-pmh3", Code: "pmh3", Name: "MH3 Promos", SetType: "promo"},
		{ScryfallID: "set-unf", Code: "unf", Name: "Unfinity", SetType: "funny"},
		{ScryfallID: "set-plst", Code: "plst", Name: "The List", SetType: "masters"},
		{ScryfallID: "set-old", Code: "old", Name: "Old Expansion", SetType: "expansion"},
	}
	if err := db.Create(&sets).Error; err != nil {
		t.Fatalf("failed to seed sets: %v", err)
	}
	card := func(id, set, lang, released, extra string) Card {
		return Card{ScryfallID: id, OracleID: "oracle-op", RawJSON: `{"id":"` + id +
			`","oracle_id":"oracle-op","name":"Ocelot Pride","set":"` + set +
			`","collector_number":"1","lang":"` + lang + `","released_at":"` + released + `"` + extra + `}`}
	}
	cards := []Card{
		// Newer, but each one has something that should push it down.
		card("op-es-plain", "mh3", "es", "2024-06-14", ""),
		card("op-borderless", "mh3", "en", "2024-06-14", `,"full_art":true,"border_color":"borderless","frame_effects":["inverted"]`),
		card("op-boosterfun", "mh3", "en", "2024-06-14", `,"promo_types":["boosterfun"]`),
		card("op-promo", "pmh3", "en", "2026-01-23", `,"promo":true`),
		card("op-funny", "unf", "en", "2026-02-01", ""),
		card("op-list", "plst", "en", "2026-03-01", ""),
		card("op-digital", "mh3", "en", "2026-04-01", `,"digital":true`),
		// Plain printings: the newer one should win.
		card("op-old-plain", "old", "en", "2020-01-01", `,"frame_effects":["legendary"]`),
		card("op-plain", "mh3", "en", "2024-06-14", ""),
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("failed to seed cards: %v", err)
	}
}

func TestGetAnyCardsByOracle(t *testing.T) {
	db := setupResolveTestDB(t)
	seedResolveCards(t, db)

	got, err := GetAnyCardsByOracle(db, []string{"oracle-aa", "oracle-sr", "oracle-missing"})
	if err != nil {
		t.Fatalf("GetAnyCardsByOracle: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected one card for each of the 2 known oracles, got %d", len(got))
	}
	// oracle-aa has an English and a German printing; either will do.
	if card := got["oracle-aa"]; card.OracleID != "oracle-aa" || card.Name == "" {
		t.Errorf("expected a printing of oracle-aa, got %+v", card)
	}
	if card := got["oracle-sr"]; card.Name != "Sol Ring" {
		t.Errorf("expected Sol Ring for oracle-sr, got %q", card.Name)
	}

	empty, err := GetAnyCardsByOracle(db, nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("expected an empty map for no oracles, got %v, %v", empty, err)
	}
}

func TestGetPreferredCardsByOracle_PrefersPlainNewestPrinting(t *testing.T) {
	db := setupResolveTestDB(t)
	seedPrintingVariety(t, db)

	got, err := GetPreferredCardsByOracle(db, []string{"oracle-op", "oracle-missing"}, "en", true)
	if err != nil {
		t.Fatalf("GetPreferredCardsByOracle: %v", err)
	}
	if card := got["oracle-op"]; card.ID != "op-plain" {
		t.Errorf("expected op-plain, got %q", card.ID)
	}
	if _, ok := got["oracle-missing"]; ok {
		t.Error("expected an unknown oracle to be absent")
	}
}

func TestGetPreferredCardsByOracle_PrefersRequestedLanguage(t *testing.T) {
	db := setupResolveTestDB(t)
	seedPrintingVariety(t, db)

	got, err := GetPreferredCardsByOracle(db, []string{"oracle-op"}, "es", false)
	if err != nil {
		t.Fatalf("GetPreferredCardsByOracle: %v", err)
	}
	if card := got["oracle-op"]; card.ID != "op-es-plain" {
		t.Errorf("expected the Spanish printing for lang=es, got %q", card.ID)
	}

	// A language with no printings falls back to English, not to Spanish.
	got, err = GetPreferredCardsByOracle(db, []string{"oracle-op"}, "ja", false)
	if err != nil {
		t.Fatalf("GetPreferredCardsByOracle: %v", err)
	}
	if card := got["oracle-op"]; card.ID != "op-plain" {
		t.Errorf("expected the English fallback for lang=ja, got %q", card.ID)
	}
}

func TestGetPreferredCardsByOracle_PrefersOwnedPrinting(t *testing.T) {
	db := setupResolveTestDB(t)
	seedPrintingVariety(t, db)
	owned := Inventory{ScryfallID: "op-borderless", OracleID: "oracle-op", Treatment: "nonfoil", Quantity: 1}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}

	got, err := GetPreferredCardsByOracle(db, []string{"oracle-op"}, "en", true)
	if err != nil {
		t.Fatalf("GetPreferredCardsByOracle: %v", err)
	}
	if card := got["oracle-op"]; card.ID != "op-borderless" {
		t.Errorf("expected the owned printing, got %q", card.ID)
	}

	got, err = GetPreferredCardsByOracle(db, []string{"oracle-op"}, "en", false)
	if err != nil {
		t.Fatalf("GetPreferredCardsByOracle: %v", err)
	}
	if card := got["oracle-op"]; card.ID != "op-plain" {
		t.Errorf("expected ownership to be ignored without preferOwned, got %q", card.ID)
	}
}

func TestGetScryfallCardsByNames_PrefersPlainPrinting(t *testing.T) {
	db := setupResolveTestDB(t)
	seedPrintingVariety(t, db)

	got, err := GetScryfallCardsByNames(db, []string{"Ocelot Pride"}, "en")
	if err != nil {
		t.Fatalf("GetScryfallCardsByNames: %v", err)
	}
	if card := got["ocelot pride"]; card.ID != "op-plain" {
		t.Errorf("expected op-plain, got %q", card.ID)
	}
}
