package services

import (
	"backend/models"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAutoSortTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Card{}, &models.SortingRule{}, &models.StorageLocation{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

const testCardRawJSON = `{"name":"Test Card","set":"test","set_name":"Test Set","rarity":"rare","type_line":"Creature","oracle_text":"Flying","mana_cost":"{2}{W}","cmc":3.0,"layout":"normal","promo":false,"reprint":false,"digital":false,"reserved":false,"foil":false,"nonfoil":true,"oversized":false,"full_art":false,"booster":true,"frame":"2015","border_color":"black","collector_number":"1","artist":"Test Artist","power":"2","toughness":"3","colors":["W"],"color_identity":["W"],"keywords":["Flying"],"finishes":["nonfoil"],"promo_types":[],"prices":{"usd":"1.00","usd_foil":null,"usd_etched":null,"eur":null,"eur_foil":null,"tix":null}}`

func setupAutoSortTestData(t *testing.T, db *gorm.DB) (*models.Card, *models.StorageLocation, *models.SortingRule) {
	t.Helper()

	// Create a test card
	card := &models.Card{
		ScryfallID: "test-card-001",
		OracleID:   "test-oracle-001",
		RawJSON:    testCardRawJSON,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("failed to create test card: %v", err)
	}

	// Create a storage location
	storage := &models.StorageLocation{
		Name:        "White Cards Box",
		StorageType: models.Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage location: %v", err)
	}

	// Create a sorting rule that matches white cards
	rule := &models.SortingRule{
		Name:              "White Cards Rule",
		Priority:          1,
		Expression:        `hasColor("W")`,
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("failed to create sorting rule: %v", err)
	}

	return card, storage, rule
}

func TestAutoSort_DetermineStorageLocation_RuleMatches(t *testing.T) {
	db := setupAutoSortTestDB(t)
	card, storage, _ := setupAutoSortTestData(t, db)

	service := NewAutoSortService(db)
	ctx := context.Background()

	locationID, err := service.DetermineStorageLocation(ctx, card.ScryfallID, "nonfoil")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if locationID == nil {
		t.Fatal("expected a storage location ID, got nil")
	}
	if *locationID != storage.ID {
		t.Errorf("expected storage location ID %d, got %d", storage.ID, *locationID)
	}
}

func TestAutoSort_DetermineStorageLocation_CardNotFound(t *testing.T) {
	db := setupAutoSortTestDB(t)
	// Set up data but use a nonexistent card ID
	setupAutoSortTestData(t, db)

	service := NewAutoSortService(db)
	ctx := context.Background()

	locationID, err := service.DetermineStorageLocation(ctx, "nonexistent-card-id", "nonfoil")
	if err == nil {
		t.Error("expected error for nonexistent card, got nil")
	}
	if locationID != nil {
		t.Errorf("expected nil location ID, got %v", *locationID)
	}
}

func TestAutoSort_DetermineStorageLocation_NoMatchingRule(t *testing.T) {
	db := setupAutoSortTestDB(t)

	// Create a card that is blue (not white)
	blueCardJSON := `{"name":"Blue Card","set":"test","set_name":"Test Set","rarity":"common","type_line":"Creature","oracle_text":"","mana_cost":"{U}","cmc":1.0,"layout":"normal","promo":false,"reprint":false,"digital":false,"reserved":false,"foil":false,"nonfoil":true,"oversized":false,"full_art":false,"booster":true,"frame":"2015","border_color":"black","collector_number":"2","artist":"Test Artist","power":"1","toughness":"1","colors":["U"],"color_identity":["U"],"keywords":[],"finishes":["nonfoil"],"promo_types":[],"prices":{"usd":"0.10","usd_foil":null,"usd_etched":null,"eur":null,"eur_foil":null,"tix":null}}`

	blueCard := &models.Card{
		ScryfallID: "blue-card-001",
		OracleID:   "blue-oracle-001",
		RawJSON:    blueCardJSON,
	}
	if err := db.Create(blueCard).Error; err != nil {
		t.Fatalf("failed to create blue card: %v", err)
	}

	// Create a storage location and a rule that only matches red cards
	storage := &models.StorageLocation{
		Name:        "Red Cards Box",
		StorageType: models.Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage location: %v", err)
	}

	rule := &models.SortingRule{
		Name:              "Red Cards Only",
		Priority:          1,
		Expression:        `hasColor("R")`,
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("failed to create sorting rule: %v", err)
	}

	service := NewAutoSortService(db)
	ctx := context.Background()

	locationID, err := service.DetermineStorageLocation(ctx, blueCard.ScryfallID, "nonfoil")
	if err == nil {
		t.Error("expected error when no rule matches, got nil")
	}
	if locationID != nil {
		t.Errorf("expected nil location ID when no rule matches, got %v", *locationID)
	}
}

func TestAutoSort_DetermineStorageLocation_DisabledRuleSkipped(t *testing.T) {
	db := setupAutoSortTestDB(t)

	// Create a test card
	card := &models.Card{
		ScryfallID: "card-disabled-rule",
		OracleID:   "oracle-disabled-rule",
		RawJSON:    testCardRawJSON,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Create storage locations
	disabledStorage := &models.StorageLocation{
		Name:        "Disabled Storage",
		StorageType: models.Box,
	}
	if err := db.Create(disabledStorage).Error; err != nil {
		t.Fatalf("failed to create disabled storage: %v", err)
	}

	enabledStorage := &models.StorageLocation{
		Name:        "Enabled Storage",
		StorageType: models.Box,
	}
	if err := db.Create(enabledStorage).Error; err != nil {
		t.Fatalf("failed to create enabled storage: %v", err)
	}

	// Create a disabled rule with higher priority (lower number)
	disabledRule := &models.SortingRule{
		Name:              "Disabled White Rule",
		Priority:          1,
		Expression:        `hasColor("W")`,
		StorageLocationID: disabledStorage.ID,
		Enabled:           true,
	}
	if err := db.Create(disabledRule).Error; err != nil {
		t.Fatalf("failed to create disabled rule: %v", err)
	}
	// Disable it after creation (since default is true)
	if err := db.Model(disabledRule).Update("enabled", false).Error; err != nil {
		t.Fatalf("failed to disable rule: %v", err)
	}

	// Create an enabled rule with lower priority (higher number)
	enabledRule := &models.SortingRule{
		Name:              "Enabled White Rule",
		Priority:          2,
		Expression:        `hasColor("W")`,
		StorageLocationID: enabledStorage.ID,
		Enabled:           true,
	}
	if err := db.Create(enabledRule).Error; err != nil {
		t.Fatalf("failed to create enabled rule: %v", err)
	}

	service := NewAutoSortService(db)
	ctx := context.Background()

	locationID, err := service.DetermineStorageLocation(ctx, card.ScryfallID, "nonfoil")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if locationID == nil {
		t.Fatal("expected location ID, got nil")
	}
	// Should match the enabled rule's storage, not the disabled one
	if *locationID != enabledStorage.ID {
		t.Errorf("expected storage location ID %d (enabled), got %d", enabledStorage.ID, *locationID)
	}
}

func TestAutoSort_DetermineStorageLocation_PriorityOrder(t *testing.T) {
	db := setupAutoSortTestDB(t)

	card := &models.Card{
		ScryfallID: "card-priority",
		OracleID:   "oracle-priority",
		RawJSON:    testCardRawJSON,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Create two storage locations
	highPrioStorage := &models.StorageLocation{
		Name:        "High Priority Box",
		StorageType: models.Box,
	}
	if err := db.Create(highPrioStorage).Error; err != nil {
		t.Fatalf("failed to create high priority storage: %v", err)
	}

	lowPrioStorage := &models.StorageLocation{
		Name:        "Low Priority Box",
		StorageType: models.Box,
	}
	if err := db.Create(lowPrioStorage).Error; err != nil {
		t.Fatalf("failed to create low priority storage: %v", err)
	}

	// Create rules -- both match, but priority 1 should win
	lowPrioRule := &models.SortingRule{
		Name:              "Low Priority White",
		Priority:          10,
		Expression:        `hasColor("W")`,
		StorageLocationID: lowPrioStorage.ID,
		Enabled:           true,
	}
	if err := db.Create(lowPrioRule).Error; err != nil {
		t.Fatalf("failed to create low prio rule: %v", err)
	}

	highPrioRule := &models.SortingRule{
		Name:              "High Priority White",
		Priority:          1,
		Expression:        `hasColor("W")`,
		StorageLocationID: highPrioStorage.ID,
		Enabled:           true,
	}
	if err := db.Create(highPrioRule).Error; err != nil {
		t.Fatalf("failed to create high prio rule: %v", err)
	}

	service := NewAutoSortService(db)
	ctx := context.Background()

	locationID, err := service.DetermineStorageLocation(ctx, card.ScryfallID, "nonfoil")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if locationID == nil {
		t.Fatal("expected location ID, got nil")
	}
	if *locationID != highPrioStorage.ID {
		t.Errorf("expected highest priority storage ID %d, got %d", highPrioStorage.ID, *locationID)
	}
}
