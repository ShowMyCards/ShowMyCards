package api

import (
	"testing"

	"backend/models"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildCardPrices(t *testing.T) {
	prices := scryfall.Prices{
		USD:       "10.50",
		USDFoil:   "25.00",
		USDEtched: "30.00",
		EUR:       "9.00",
		EURFoil:   "22.00",
		Tix:       "5.00",
	}

	result := BuildCardPrices(prices)

	if result.USD != "10.50" {
		t.Errorf("USD: expected %q, got %q", "10.50", result.USD)
	}
	if result.USDFoil != "25.00" {
		t.Errorf("USDFoil: expected %q, got %q", "25.00", result.USDFoil)
	}
	if result.USDEtched != "30.00" {
		t.Errorf("USDEtched: expected %q, got %q", "30.00", result.USDEtched)
	}
	if result.EUR != "9.00" {
		t.Errorf("EUR: expected %q, got %q", "9.00", result.EUR)
	}
	if result.EURFoil != "22.00" {
		t.Errorf("EURFoil: expected %q, got %q", "22.00", result.EURFoil)
	}
	if result.Tix != "5.00" {
		t.Errorf("Tix: expected %q, got %q", "5.00", result.Tix)
	}
}

func TestBuildCardPrices_Empty(t *testing.T) {
	result := BuildCardPrices(scryfall.Prices{})

	if result.USD != "" || result.USDFoil != "" || result.USDEtched != "" {
		t.Errorf("expected empty prices, got %+v", result)
	}
}

func TestBuildCardResult(t *testing.T) {
	rank := 42
	card := scryfall.Card{
		ID:              "test-id",
		OracleID:        "oracle-id",
		Name:            "Lightning Bolt",
		Set:             "lea",
		SetName:         "Limited Edition Alpha",
		CollectorNumber: "161",
		ColorIdentity:   []scryfall.Color{"R"},
		Finishes:        []scryfall.Finish{"nonfoil"},
		EDHRECRank:      &rank,
		Prices: scryfall.Prices{
			USD: "500.00",
		},
	}

	result := BuildCardResult(card)

	if result.ID != "test-id" {
		t.Errorf("ID: expected %q, got %q", "test-id", result.ID)
	}
	if result.OracleID != "oracle-id" {
		t.Errorf("OracleID: expected %q, got %q", "oracle-id", result.OracleID)
	}
	if result.Name != "Lightning Bolt" {
		t.Errorf("Name: expected %q, got %q", "Lightning Bolt", result.Name)
	}
	if result.SetCode != "lea" {
		t.Errorf("SetCode: expected %q, got %q", "lea", result.SetCode)
	}
	if result.SetName != "Limited Edition Alpha" {
		t.Errorf("SetName: expected %q, got %q", "Limited Edition Alpha", result.SetName)
	}
	if result.CollectorNumber != "161" {
		t.Errorf("CollectorNumber: expected %q, got %q", "161", result.CollectorNumber)
	}
	if len(result.ColorIdentity) != 1 || result.ColorIdentity[0] != "R" {
		t.Errorf("ColorIdentity: expected [R], got %v", result.ColorIdentity)
	}
	if len(result.Finishes) != 1 || result.Finishes[0] != "nonfoil" {
		t.Errorf("Finishes: expected [nonfoil], got %v", result.Finishes)
	}
	if result.EDHRECRank == nil || *result.EDHRECRank != 42 {
		t.Errorf("EDHRECRank: expected 42, got %v", result.EDHRECRank)
	}
	if result.Prices.USD != "500.00" {
		t.Errorf("Prices.USD: expected %q, got %q", "500.00", result.Prices.USD)
	}
	if result.PrintedName != nil {
		t.Errorf("PrintedName: expected nil for an English card, got %q", *result.PrintedName)
	}
}

func TestBuildCardResult_PrintedName(t *testing.T) {
	printedName := "稲妻"
	card := scryfall.Card{
		ID:          "test-id",
		Name:        "Lightning Bolt",
		Lang:        "ja",
		PrintedName: &printedName,
	}

	result := BuildCardResult(card)

	if result.Name != "Lightning Bolt" {
		t.Errorf("Name: expected canonical English name %q, got %q", "Lightning Bolt", result.Name)
	}
	if result.PrintedName == nil {
		t.Fatal("PrintedName: expected localized name, got nil")
	}
	if *result.PrintedName != printedName {
		t.Errorf("PrintedName: expected %q, got %q", printedName, *result.PrintedName)
	}
}

func TestBuildCardResult_EmptyCard(t *testing.T) {
	result := BuildCardResult(scryfall.Card{})

	if result.ID != "" {
		t.Errorf("expected empty ID, got %q", result.ID)
	}
	if result.Name != "" {
		t.Errorf("expected empty Name, got %q", result.Name)
	}
}

func TestCardPrices_IsEmpty(t *testing.T) {
	if !(CardPrices{}).IsEmpty() {
		t.Error("zero-value CardPrices should be empty")
	}
	if (CardPrices{USD: "1.00"}).IsEmpty() {
		t.Error("CardPrices with a usd price should not be empty")
	}
	if (CardPrices{Tix: "0.01"}).IsEmpty() {
		t.Error("CardPrices with only a tix price should not be empty")
	}
}

// setupBackfillTestDB builds a cards table with the generated columns the
// English-price fallback relies on, mirroring the production migration.
func setupBackfillTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	for _, stmt := range []string{
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

func TestBackfillEnglishPrices(t *testing.T) {
	db := setupBackfillTestDB(t)

	if err := db.Create(&models.Card{
		ScryfallID: "en-id",
		OracleID:   "oracle-1",
		RawJSON:    `{"id":"en-id","set":"c21","collector_number":"263","lang":"en","prices":{"usd":"1.73","eur":"0.92"}}`,
	}).Error; err != nil {
		t.Fatalf("failed to seed english card: %v", err)
	}

	german := &CardResult{ID: "de-id", Language: "de", SetCode: "c21", CollectorNumber: "263", Prices: CardPrices{}}
	englishPriced := &CardResult{ID: "en-id", Language: "en", SetCode: "c21", CollectorNumber: "263", Prices: CardPrices{USD: "1.73"}}
	frenchNoEnglish := &CardResult{ID: "fr-id", Language: "fr", SetCode: "xxx", CollectorNumber: "999", Prices: CardPrices{}}
	germanWithOwnPrice := &CardResult{ID: "de-id-2", Language: "de", SetCode: "c21", CollectorNumber: "263", Prices: CardPrices{USD: "9.99"}}

	BackfillEnglishPrices(db, []*CardResult{german, englishPriced, frenchNoEnglish, germanWithOwnPrice})

	if german.Prices.USD != "1.73" || german.Prices.EUR != "0.92" {
		t.Errorf("german: expected backfilled usd 1.73 / eur 0.92, got %+v", german.Prices)
	}
	if englishPriced.Prices.USD != "1.73" {
		t.Errorf("english: expected untouched usd 1.73, got %+v", englishPriced.Prices)
	}
	if !frenchNoEnglish.Prices.IsEmpty() {
		t.Errorf("french with no english printing: expected unchanged (empty), got %+v", frenchNoEnglish.Prices)
	}
	if germanWithOwnPrice.Prices.USD != "9.99" {
		t.Errorf("german with own price: expected untouched usd 9.99, got %+v", germanWithOwnPrice.Prices)
	}
}
