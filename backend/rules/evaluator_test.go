package rules

import (
	"backend/models"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.StorageLocation{}, &models.SortingRule{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createTestLocation(t *testing.T, db *gorm.DB) models.StorageLocation {
	t.Helper()
	location := models.StorageLocation{
		Name:        "Test Box",
		StorageType: models.Box,
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("failed to create test location: %v", err)
	}
	return location
}

func createTestRule(t *testing.T, db *gorm.DB, name string, priority int, expression string, locationID uint, enabled bool) models.SortingRule {
	t.Helper()
	rule := models.SortingRule{
		Name:              name,
		Priority:          priority,
		Expression:        expression,
		StorageLocationID: locationID,
		Enabled:           enabled,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create test rule: %v", err)
	}
	// Explicitly update enabled field if it's false (GORM issue with boolean defaults)
	if !enabled {
		if err := db.Model(&rule).Update("enabled", false).Error; err != nil {
			t.Fatalf("failed to update enabled field: %v", err)
		}
	}
	return rule
}

// EvaluateExpression tests

func TestEvaluateExpression_SimpleComparison(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 3.5,
		},
	}

	result, err := evaluator.EvaluateExpression("prices.usd < 5.0", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected expression to evaluate to true")
	}
}

func TestEvaluateExpression_FalseResult(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 10.0,
		},
	}

	result, err := evaluator.EvaluateExpression("prices.usd < 5.0", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected expression to evaluate to false")
	}
}

func TestEvaluateExpression_StringComparison(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"rarity": "mythic",
	}

	result, err := evaluator.EvaluateExpression("rarity == 'mythic'", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected expression to evaluate to true")
	}
}

func TestEvaluateExpression_ComplexExpression(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 15.0,
		},
		"rarity": "mythic",
	}

	result, err := evaluator.EvaluateExpression("prices.usd > 10.0 && rarity == 'mythic'", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected expression to evaluate to true")
	}
}

func TestEvaluateExpression_ArrayLength(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"colors": []string{"R", "G", "B"},
	}

	result, err := evaluator.EvaluateExpression("len(colors) > 2", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected expression to evaluate to true")
	}
}

func TestEvaluateExpression_EmptyExpression(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{}

	_, err := evaluator.EvaluateExpression("", cardData)
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

func TestEvaluateExpression_InvalidSyntax(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{}

	_, err := evaluator.EvaluateExpression("invalid syntax {", cardData)
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

func TestEvaluateExpression_NonBooleanResult(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"value": 42,
	}

	_, err := evaluator.EvaluateExpression("value", cardData)
	if err == nil {
		t.Error("expected error for non-boolean result")
	}
}

func TestEvaluateExpression_MissingField(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{}

	_, err := evaluator.EvaluateExpression("prices.usd < 5.0", cardData)
	if err == nil {
		t.Error("expected error for missing field")
	}
}

// EvaluateCard tests

func TestEvaluateCard_SingleRule(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location := createTestLocation(t, db)
	createTestRule(t, db, "Cheap Cards", 1, "prices.usd < 5.0", location.ID, true)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 3.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected storage location, got nil")
	}

	if result.ID != location.ID {
		t.Errorf("expected location ID %d, got %d", location.ID, result.ID)
	}
}

func TestEvaluateCard_PriorityOrder(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location1 := createTestLocation(t, db)
	location2 := createTestLocation(t, db)

	// Create rules with different priorities (lower priority number = higher priority)
	createTestRule(t, db, "High Priority", 1, "prices.usd < 10.0", location1.ID, true)
	createTestRule(t, db, "Low Priority", 2, "prices.usd < 10.0", location2.ID, true)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 5.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Should match the first (higher priority) rule
	if result.ID != location1.ID {
		t.Errorf("expected first location (higher priority) ID %d, got %d", location1.ID, result.ID)
	}
}

func TestEvaluateCard_FirstMatchWins(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location1 := createTestLocation(t, db)
	location2 := createTestLocation(t, db)

	// Create rules where both would match, but first one wins
	createTestRule(t, db, "Rule 1", 1, "prices.usd < 10.0", location1.ID, true)
	createTestRule(t, db, "Rule 2", 2, "prices.usd < 20.0", location2.ID, true)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 5.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Should match the first rule only
	if result.ID != location1.ID {
		t.Errorf("expected first location ID %d, got %d", location1.ID, result.ID)
	}
}

func TestEvaluateCard_SkipsDisabledRules(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location1 := createTestLocation(t, db)
	location2 := createTestLocation(t, db)

	// Create disabled rule with higher priority
	createTestRule(t, db, "Disabled Rule", 1, "prices.usd < 10.0", location1.ID, false)
	createTestRule(t, db, "Enabled Rule", 2, "prices.usd < 10.0", location2.ID, true)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 5.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Should skip disabled rule and match the enabled one
	if result.ID != location2.ID {
		t.Errorf("expected location2 ID %d, got %d", location2.ID, result.ID)
	}
}

func TestEvaluateCard_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location := createTestLocation(t, db)
	createTestRule(t, db, "Expensive Cards", 1, "prices.usd > 100.0", location.ID, true)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 5.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err == nil {
		t.Error("expected error when no rules match")
	}
	if result != nil {
		t.Error("expected nil result when no rules match")
	}
}

func TestEvaluateCard_NoRules(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"prices": map[string]interface{}{
			"usd": 5.0,
		},
	}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err == nil {
		t.Error("expected error when no rules exist")
	}
	if result != nil {
		t.Error("expected nil result when no rules exist")
	}
}

func TestEvaluateCard_InvalidExpression(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	location := createTestLocation(t, db)
	createTestRule(t, db, "Bad Rule", 1, "invalid syntax {", location.ID, true)

	cardData := map[string]interface{}{}

	result, err := evaluator.EvaluateCard(context.Background(), cardData)
	if err == nil {
		t.Error("expected error when rule has invalid expression")
	}
	if result != nil {
		t.Error("expected nil result when rule has invalid expression")
	}
}

// ValidateExpression tests

func TestValidateExpression_Valid(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	err := evaluator.ValidateExpression("prices.usd < 5.0")
	if err != nil {
		t.Errorf("expected valid expression, got error: %v", err)
	}
}

func TestValidateExpression_ComplexValid(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	err := evaluator.ValidateExpression("prices.usd > 10.0 && rarity == 'mythic' && len(colors) > 1")
	if err != nil {
		t.Errorf("expected valid expression, got error: %v", err)
	}
}

func TestValidateExpression_Empty(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	err := evaluator.ValidateExpression("")
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

func TestValidateExpression_InvalidSyntax(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	err := evaluator.ValidateExpression("invalid { syntax")
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

// Helper function tests

func TestHelperFunction_HasColor_White(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
	}

	result, err := evaluator.EvaluateExpression("hasColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected hasColor('W') to return true for white card")
	}
}

func TestHelperFunction_HasColor_MultiColor(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Simulate Heliod - double-faced card with W and U
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"U", "W"},
	}

	// Should match white
	result, err := evaluator.EvaluateExpression("hasColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if !result {
		t.Error("expected hasColor('W') to return true for WU card")
	}

	// Should match blue
	result, err = evaluator.EvaluateExpression("hasColor('U')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if !result {
		t.Error("expected hasColor('U') to return true for WU card")
	}

	// Should not match red
	result, err = evaluator.EvaluateExpression("hasColor('R')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if result {
		t.Error("expected hasColor('R') to return false for WU card")
	}
}

func TestHelperFunction_IsMonoColor_True(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
	}

	result, err := evaluator.EvaluateExpression("isMonoColor()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isMonoColor() to return true for mono-white card")
	}
}

func TestHelperFunction_IsMonoColor_False(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"U", "W"},
	}

	result, err := evaluator.EvaluateExpression("isMonoColor()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isMonoColor() to return false for multicolor card")
	}
}

func TestHelperFunction_IsMultiColor_True(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"U", "W"},
	}

	result, err := evaluator.EvaluateExpression("isMultiColor()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isMultiColor() to return true for WU card")
	}
}

func TestHelperFunction_IsMultiColor_False(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
	}

	result, err := evaluator.EvaluateExpression("isMultiColor()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isMultiColor() to return false for mono-white card")
	}
}

func TestHelperFunction_IsColorless_True(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{},
	}

	result, err := evaluator.EvaluateExpression("isColorless()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColorless() to return true for colorless card")
	}
}

func TestHelperFunction_IsColorless_False(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
	}

	result, err := evaluator.EvaluateExpression("isColorless()", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isColorless() to return false for white card")
	}
}

func TestHelperFunction_CombinedWithOtherConditions(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
		"prices": map[string]interface{}{
			"usd": 3.0,
		},
		"rarity": "rare",
	}

	// Test combining helper with price and rarity checks
	result, err := evaluator.EvaluateExpression("hasColor('W') && prices.usd < 5.0 && rarity == 'rare'", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected complex expression with hasColor() to return true")
	}
}

func TestHelperFunction_HeliodRealWorldCase(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Simulate Heliod, the Radiant Dawn (WU double-faced card)
	heliodData := map[string]interface{}{
		"name":           "Heliod, the Radiant Dawn",
		"color_identity": []interface{}{"U", "W"},
		"layout":         "transform",
	}

	// Now a simple rule "hasColor('W')" should match
	result, err := evaluator.EvaluateExpression("hasColor('W')", heliodData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected Heliod to match hasColor('W') rule")
	}

	// Should also be detected as multicolor
	result, err = evaluator.EvaluateExpression("isMultiColor()", heliodData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected Heliod to be detected as multicolor")
	}
}

// isColor helper function tests

func TestHelperFunction_IsColor_SingleColor_Matches(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W"},
	}

	result, err := evaluator.EvaluateExpression("isColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('W') to return true for mono-white card")
	}
}

func TestHelperFunction_IsColor_SingleColor_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"U"},
	}

	result, err := evaluator.EvaluateExpression("isColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isColor('W') to return false for mono-blue card")
	}
}

func TestHelperFunction_IsColor_TwoColors_ExactMatch(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W", "U"},
	}

	result, err := evaluator.EvaluateExpression("isColor('W', 'U')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('W', 'U') to return true for WU card")
	}
}

func TestHelperFunction_IsColor_TwoColors_OrderIndependent(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Card has colors in U, W order
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"U", "W"},
	}

	// Query with W, U order - should still match
	result, err := evaluator.EvaluateExpression("isColor('W', 'U')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('W', 'U') to return true for UW card (order independent)")
	}
}

func TestHelperFunction_IsColor_TwoColors_PartialNoMatch(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Card is WU
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W", "U"},
	}

	// Looking for exactly W (single color) - should NOT match WU
	result, err := evaluator.EvaluateExpression("isColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isColor('W') to return false for WU card (not exact match)")
	}
}

func TestHelperFunction_IsColor_ThreeColors_ExactMatch(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W", "U", "B"},
	}

	result, err := evaluator.EvaluateExpression("isColor('W', 'U', 'B')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('W', 'U', 'B') to return true for Esper card")
	}
}

func TestHelperFunction_IsColor_ThreeColors_WrongCount(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Card is WUB (3 colors)
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W", "U", "B"},
	}

	// Looking for WU (2 colors) - should NOT match
	result, err := evaluator.EvaluateExpression("isColor('W', 'U')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isColor('W', 'U') to return false for WUB card (different count)")
	}
}

func TestHelperFunction_IsColor_UsesColorsField(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Card with colors field but no color_identity
	cardData := map[string]interface{}{
		"colors": []interface{}{"R", "G"},
	}

	result, err := evaluator.EvaluateExpression("isColor('R', 'G')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('R', 'G') to return true using colors field")
	}
}

func TestHelperFunction_IsColor_PrefersColorIdentity(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Card with different color_identity and colors
	// (e.g., a land that produces R but has R in identity)
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"R"},
		"colors":         []interface{}{}, // colorless by casting cost
	}

	result, err := evaluator.EvaluateExpression("isColor('R')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor('R') to match color_identity")
	}
}

func TestHelperFunction_IsColor_Colorless(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Colorless card
	cardData := map[string]interface{}{
		"color_identity": []interface{}{},
	}

	// isColor with any color should not match colorless
	result, err := evaluator.EvaluateExpression("isColor('W')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected isColor('W') to return false for colorless card")
	}
}

func TestHelperFunction_IsColor_FiveColor(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Five-color card (WUBRG)
	cardData := map[string]interface{}{
		"color_identity": []interface{}{"W", "U", "B", "R", "G"},
	}

	result, err := evaluator.EvaluateExpression("isColor('W', 'U', 'B', 'R', 'G')", cardData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected isColor with all 5 colors to return true for WUBRG card")
	}
}

func TestHelperFunction_IsColor_RealWorldHeliod(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Heliod, the Radiant Dawn - WU double-faced card
	heliodData := map[string]interface{}{
		"name":           "Heliod, the Radiant Dawn",
		"color_identity": []interface{}{"U", "W"},
		"layout":         "transform",
	}

	// Should match exactly WU
	result, err := evaluator.EvaluateExpression("isColor('W', 'U')", heliodData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if !result {
		t.Error("expected Heliod to match isColor('W', 'U')")
	}

	// Should NOT match just W (partial)
	result, err = evaluator.EvaluateExpression("isColor('W')", heliodData)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if result {
		t.Error("expected Heliod to NOT match isColor('W') - it's multicolor")
	}
}

func TestValidateExpression_IsColorHelper(t *testing.T) {
	db := setupTestDB(t)
	evaluator := NewEvaluator(db)

	// Validate that isColor expressions compile
	err := evaluator.ValidateExpression("isColor('W')")
	if err != nil {
		t.Errorf("expected isColor('W') to be valid, got error: %v", err)
	}

	err = evaluator.ValidateExpression("isColor('W', 'U')")
	if err != nil {
		t.Errorf("expected isColor('W', 'U') to be valid, got error: %v", err)
	}

	err = evaluator.ValidateExpression("isColor('W', 'U', 'B', 'R', 'G')")
	if err != nil {
		t.Errorf("expected isColor with 5 colors to be valid, got error: %v", err)
	}
}
