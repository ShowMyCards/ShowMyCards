package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"backend/models"
	"backend/scryfall"
	"backend/services"

	goscryfall "github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testBrimazOracle = "oracle-brimaz"

type printingsCardFixture struct {
	id, oracle, rawJSON string
}

var (
	fixtureBrimazEN = printingsCardFixture{
		id:     "f54eb705-6326-4bac-bf0e-68d42db7a270",
		oracle: testBrimazOracle,
		rawJSON: `{
			"id": "f54eb705-6326-4bac-bf0e-68d42db7a270",
			"oracle_id": "oracle-brimaz",
			"name": "Brimaz, King of Oreskos",
			"set": "bng",
			"set_name": "Born of the Gods",
			"collector_number": "5",
			"lang": "en",
			"released_at": "2014-02-07",
			"finishes": ["nonfoil", "foil"],
			"prices": {"usd": "12.50", "eur": "9.00"}
		}`,
	}
	fixtureBrimazJA = printingsCardFixture{
		id:     "ce852c3d-ca93-47c6-88f1-08ed437fba35",
		oracle: testBrimazOracle,
		rawJSON: `{
			"id": "ce852c3d-ca93-47c6-88f1-08ed437fba35",
			"oracle_id": "oracle-brimaz",
			"name": "Brimaz, King of Oreskos",
			"printed_name": "オレスコスの王、ブリマーズ",
			"set": "bng",
			"set_name": "Born of the Gods",
			"collector_number": "5",
			"lang": "ja",
			"released_at": "2014-02-07",
			"finishes": ["nonfoil", "foil"],
			"prices": {}
		}`,
	}
	fixtureBrimazPromoEN = printingsCardFixture{
		id:     "brimaz-promo-en",
		oracle: testBrimazOracle,
		rawJSON: `{
			"id": "brimaz-promo-en",
			"oracle_id": "oracle-brimaz",
			"name": "Brimaz, King of Oreskos",
			"set": "sld",
			"set_name": "Secret Lair Drop",
			"collector_number": "100",
			"lang": "en",
			"released_at": "2023-05-01",
			"finishes": ["foil"],
			"prices": {"usd_foil": "15.00"}
		}`,
	}
)

func unmarshalFixtureCard(t *testing.T, f printingsCardFixture) goscryfall.Card {
	t.Helper()
	var card goscryfall.Card
	if err := json.Unmarshal([]byte(f.rawJSON), &card); err != nil {
		t.Fatalf("failed to unmarshal card fixture %s: %v", f.id, err)
	}
	return card
}

type printingsCardSource struct {
	searchCards  []goscryfall.Card
	searchErr    error
	byID         map[string]goscryfall.Card
	byIDErr      error
	getByIDCalls int
	lastQuery    string
	lastOpts     scryfall.SearchOptions
}

func (p *printingsCardSource) SearchWithOptions(_ context.Context, query string, opts scryfall.SearchOptions) (scryfall.SearchResult, error) {
	p.lastQuery = query
	p.lastOpts = opts
	if p.searchErr != nil {
		return scryfall.SearchResult{}, p.searchErr
	}
	return scryfall.SearchResult{
		Cards:      p.searchCards,
		TotalCards: len(p.searchCards),
		Page:       opts.Page,
	}, nil
}

func (p *printingsCardSource) GetByID(_ context.Context, id string) (goscryfall.Card, error) {
	p.getByIDCalls++
	if p.byIDErr != nil {
		return goscryfall.Card{}, p.byIDErr
	}
	if card, ok := p.byID[id]; ok {
		return card, nil
	}
	return goscryfall.Card{}, errors.New("not found")
}

func (p *printingsCardSource) Autocomplete(context.Context, string) ([]string, error) {
	return nil, nil
}

type printingsTestSetup struct {
	app  *fiber.App
	db   *gorm.DB
	boxA models.StorageLocation
}

func setupPrintingsTest(t *testing.T, source cardSource) (*printingsTestSetup, *services.SettingsService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.StorageLocation{},
		&models.Inventory{},
		&models.Deck{},
		&models.DeckItem{},
		&models.Card{},
		&models.Setting{},
	); err != nil {
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

	boxA := models.StorageLocation{Name: "Box A", StorageType: models.Box}
	if err := db.Create(&boxA).Error; err != nil {
		t.Fatalf("failed to create storage location: %v", err)
	}

	app := fiber.New()
	settingsSvc := services.NewSettingsService(db)
	handler := NewSearchHandler(source, db, settingsSvc)
	app.Get("/api/cards/printings/:oracle_id", handler.Printings)

	return &printingsTestSetup{app: app, db: db, boxA: boxA}, settingsSvc
}

func (s *printingsTestSetup) addCard(t *testing.T, f printingsCardFixture) {
	t.Helper()
	if err := s.db.Create(&models.Card{
		ScryfallID: f.id,
		OracleID:   f.oracle,
		RawJSON:    f.rawJSON,
	}).Error; err != nil {
		t.Fatalf("failed to seed card %s: %v", f.id, err)
	}
}

func (s *printingsTestSetup) addRow(t *testing.T, f printingsCardFixture, treatment string, qty int, location *uint) {
	t.Helper()
	row := models.Inventory{
		ScryfallID:        f.id,
		OracleID:          f.oracle,
		Treatment:         treatment,
		Quantity:          qty,
		StorageLocationID: location,
	}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatalf("failed to create inventory row: %v", err)
	}
	if qty == 0 {
		// Quantity has gorm:"default:1", so GORM replaces zero values on Create.
		if err := s.db.Model(&row).Update("quantity", 0).Error; err != nil {
			t.Fatalf("failed to set zero quantity on inventory row: %v", err)
		}
	}
}

func getPrintings(t *testing.T, s *printingsTestSetup, target string) (int, []InventoryCard) {
	t.Helper()
	resp, err := s.app.Test(httptest.NewRequest(http.MethodGet, target, nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var cards []InventoryCard
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&cards); err != nil {
			t.Fatalf("failed to decode printings response: %v", err)
		}
	}
	return resp.StatusCode, cards
}

func TestPrintings_IncludesCrossLanguageOwnedPrintingsAndDeduplicates(t *testing.T) {
	brimazEN := unmarshalFixtureCard(t, fixtureBrimazEN)
	brimazPromoEN := unmarshalFixtureCard(t, fixtureBrimazPromoEN)

	source := &printingsCardSource{
		searchCards: []goscryfall.Card{brimazEN, brimazPromoEN},
	}
	s, _ := setupPrintingsTest(t, source)
	s.addCard(t, fixtureBrimazEN)
	s.addCard(t, fixtureBrimazJA)
	s.addRow(t, fixtureBrimazEN, "nonfoil", 2, &s.boxA.ID)
	s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)

	deck := models.Deck{Name: "Mono White"}
	if err := s.db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}
	addDeckItemDemand(t, s.db, deck.ID, testBrimazOracle, 1)

	status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=en")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if source.lastQuery != "oracleid:"+testBrimazOracle+" game:paper l:en" {
		t.Errorf("query = %q, want %q", source.lastQuery, "oracleid:"+testBrimazOracle+" game:paper l:en")
	}
	if source.lastOpts.UniqueMode != goscryfall.UniqueModePrints {
		t.Errorf("UniqueMode = %q, want %q", source.lastOpts.UniqueMode, goscryfall.UniqueModePrints)
	}

	var gotIDs []string
	for _, c := range cards {
		gotIDs = append(gotIDs, c.ID)
	}
	// Both bng/5 printings share released_at 2014-02-07, set bng, cn 5; lang "en" < "ja".
	wantIDs := []string{fixtureBrimazEN.id, fixtureBrimazJA.id, fixtureBrimazPromoEN.id}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
	}

	jaCard := cards[1]
	if jaCard.Prices.USD != "12.50" {
		t.Errorf("expected English price backfill on Japanese printing, got %+v", jaCard.Prices)
	}
	if len(jaCard.Inventory.ThisPrinting) != 1 || len(jaCard.Inventory.OtherPrintings) != 1 || jaCard.Inventory.TotalQuantity != 3 {
		t.Errorf("unexpected Japanese printing inventory = %+v", jaCard.Inventory)
	}
	if jaCard.Deck.Owned != 3 || jaCard.Deck.Decked != 1 || jaCard.Deck.Free != 2 {
		t.Errorf("unexpected Japanese printing deck = %+v", jaCard.Deck)
	}
}

func TestPrintings_JapaneseViewIncludesOwnedEnglishPrinting(t *testing.T) {
	brimazJA := unmarshalFixtureCard(t, fixtureBrimazJA)
	source := &printingsCardSource{
		searchCards: []goscryfall.Card{brimazJA},
	}
	s, _ := setupPrintingsTest(t, source)
	s.addCard(t, fixtureBrimazEN)
	s.addCard(t, fixtureBrimazJA)
	s.addRow(t, fixtureBrimazEN, "nonfoil", 1, &s.boxA.ID)
	s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)

	status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=ja")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if source.lastQuery != "oracleid:"+testBrimazOracle+" game:paper l:ja" {
		t.Errorf("query = %q, want %q", source.lastQuery, "oracleid:"+testBrimazOracle+" game:paper l:ja")
	}

	var gotIDs []string
	for _, c := range cards {
		gotIDs = append(gotIDs, c.ID)
	}
	wantIDs := []string{fixtureBrimazEN.id, fixtureBrimazJA.id}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
	}
}

func TestPrintings_IgnoresZeroQuantityInventoryRows(t *testing.T) {
	brimazEN := unmarshalFixtureCard(t, fixtureBrimazEN)
	source := &printingsCardSource{
		searchCards: []goscryfall.Card{brimazEN},
	}
	s, _ := setupPrintingsTest(t, source)
	s.addCard(t, fixtureBrimazEN)
	s.addCard(t, fixtureBrimazJA)
	s.addRow(t, fixtureBrimazJA, "nonfoil", 0, nil)

	status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=en")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(cards) != 1 || cards[0].ID != fixtureBrimazEN.id {
		t.Fatalf("got %d cards (%v), want only %s", len(cards), cards, fixtureBrimazEN.id)
	}
}

func TestPrintings_FallsBackToGetByIDAndSkipsUnresolvable(t *testing.T) {
	brimazEN := unmarshalFixtureCard(t, fixtureBrimazEN)
	brimazJA := unmarshalFixtureCard(t, fixtureBrimazJA)

	t.Run("resolves missing local card via GetByID", func(t *testing.T) {
		source := &printingsCardSource{
			searchCards: []goscryfall.Card{brimazEN},
			byID:        map[string]goscryfall.Card{fixtureBrimazJA.id: brimazJA},
		}
		s, _ := setupPrintingsTest(t, source)
		s.addCard(t, fixtureBrimazEN)
		s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)

		status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=en")
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if len(cards) != 2 || cards[0].ID != fixtureBrimazJA.id || cards[1].ID != fixtureBrimazEN.id {
			t.Fatalf("cards = %+v, want [JA, EN]", cards)
		}
	})

	t.Run("skips owned card when GetByID fails", func(t *testing.T) {
		source := &printingsCardSource{
			searchCards: []goscryfall.Card{brimazEN},
			byIDErr:     errors.New("scryfall unavailable"),
		}
		s, _ := setupPrintingsTest(t, source)
		s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)

		status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=en")
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if len(cards) != 1 || cards[0].ID != fixtureBrimazEN.id {
			t.Fatalf("cards = %+v, want only catalog EN printing", cards)
		}
	})
}

func TestPrintings_ScryfallErrors(t *testing.T) {
	t.Run("404 from Scryfall returns owned printings", func(t *testing.T) {
		source := &printingsCardSource{
			searchErr: &goscryfall.Error{Status: http.StatusNotFound, Code: "not_found"},
		}
		s, _ := setupPrintingsTest(t, source)
		s.addCard(t, fixtureBrimazJA)
		s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)

		status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle+"?lang=de")
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if len(cards) != 1 || cards[0].ID != fixtureBrimazJA.id {
			t.Fatalf("cards = %+v, want owned JA printing", cards)
		}
	})

	t.Run("404 from Scryfall with no owned printings returns empty array", func(t *testing.T) {
		source := &printingsCardSource{
			searchErr: &goscryfall.Error{Status: http.StatusNotFound, Code: "not_found"},
		}
		s, _ := setupPrintingsTest(t, source)

		resp, err := s.app.Test(httptest.NewRequest(http.MethodGet, "/api/cards/printings/"+testBrimazOracle, nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var raw json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if string(raw) != "[]" {
			t.Fatalf("body = %s, want []", string(raw))
		}
	})

	t.Run("500 from Scryfall still returns local owned printings and skips GetByID fallback", func(t *testing.T) {
		source := &printingsCardSource{
			searchErr: &goscryfall.Error{Status: http.StatusInternalServerError, Code: "internal_error"},
			byID:      map[string]goscryfall.Card{fixtureBrimazEN.id: unmarshalFixtureCard(t, fixtureBrimazEN)},
		}
		s, _ := setupPrintingsTest(t, source)
		s.addCard(t, fixtureBrimazJA)
		s.addRow(t, fixtureBrimazJA, "nonfoil", 1, nil)
		// Owned printing absent from local cards table should not trigger GetByID when Scryfall already failed.
		s.addRow(t, fixtureBrimazEN, "nonfoil", 1, &s.boxA.ID)

		status, cards := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if len(cards) != 1 || cards[0].ID != fixtureBrimazJA.id {
			t.Fatalf("cards = %+v, want local owned JA printing only", cards)
		}
		if source.getByIDCalls != 0 {
			t.Errorf("getByIDCalls = %d, want 0", source.getByIDCalls)
		}
	})
}

func TestPrintings_UsesSettingsDefaults(t *testing.T) {
	source := &printingsCardSource{searchCards: []goscryfall.Card{unmarshalFixtureCard(t, fixtureBrimazJA)}}
	s, settingsSvc := setupPrintingsTest(t, source)
	ctx := context.Background()
	if err := settingsSvc.Set(ctx, "scryfall_default_language", "ja"); err != nil {
		t.Fatalf("failed to set language setting: %v", err)
	}
	if err := settingsSvc.Set(ctx, "scryfall_default_search", "-is:digital"); err != nil {
		t.Fatalf("failed to set default search setting: %v", err)
	}

	status, _ := getPrintings(t, s, "/api/cards/printings/"+testBrimazOracle)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	want := "oracleid:" + testBrimazOracle + " -is:digital l:ja"
	if source.lastQuery != want {
		t.Errorf("query = %q, want %q", source.lastQuery, want)
	}
}
