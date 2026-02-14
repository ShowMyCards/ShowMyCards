package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSortingRuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&SortingRule{}, &StorageLocation{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestSortingRule_ValidateSortingRule(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	tests := []struct {
		name        string
		rule        *SortingRule
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Rule",
			rule: &SortingRule{
				Name:              "High Value Cards",
				Priority:          1,
				Expression:        "prices.usd > 10",
				StorageLocationID: 1,
				Enabled:           true,
			},
			expectError: false,
		},
		{
			name: "Valid Rule - Disabled",
			rule: &SortingRule{
				Name:              "Low Priority",
				Priority:          100,
				Expression:        "true",
				StorageLocationID: 1,
				Enabled:           false,
			},
			expectError: false,
		},
		{
			name: "Invalid - Empty Name",
			rule: &SortingRule{
				Name:              "",
				Priority:          1,
				Expression:        "true",
				StorageLocationID: 1,
				Enabled:           true,
			},
			expectError: true,
			errorMsg:    "rule name cannot be empty",
		},
		{
			name: "Invalid - Empty Expression",
			rule: &SortingRule{
				Name:              "Test Rule",
				Priority:          1,
				Expression:        "",
				StorageLocationID: 1,
				Enabled:           true,
			},
			expectError: true,
			errorMsg:    "rule expression cannot be empty",
		},
		{
			name: "Invalid - Zero StorageLocationID",
			rule: &SortingRule{
				Name:              "Test Rule",
				Priority:          1,
				Expression:        "true",
				StorageLocationID: 0,
				Enabled:           true,
			},
			expectError: true,
			errorMsg:    "storage location ID must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.ValidateSortingRule(db)
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

func TestSortingRule_BeforeCreate(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name        string
		rule        *SortingRule
		expectError bool
	}{
		{
			name: "Valid Create",
			rule: &SortingRule{
				Name:              "Test Rule",
				Priority:          1,
				Expression:        "true",
				StorageLocationID: storage.ID,
				Enabled:           true,
			},
			expectError: false,
		},
		{
			name: "Invalid Create - Empty Name",
			rule: &SortingRule{
				Name:              "",
				Priority:          1,
				Expression:        "true",
				StorageLocationID: storage.ID,
				Enabled:           true,
			},
			expectError: true,
		},
		{
			name: "Invalid Create - Empty Expression",
			rule: &SortingRule{
				Name:              "Test",
				Priority:          1,
				Expression:        "",
				StorageLocationID: storage.ID,
				Enabled:           true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.rule).Error
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

func TestSortingRule_BeforeUpdate(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name        string
		updateFunc  func(*SortingRule)
		expectError bool
	}{
		{
			name: "Valid Update - Name",
			updateFunc: func(r *SortingRule) {
				r.Name = "Updated Name"
			},
			expectError: false,
		},
		{
			name: "Valid Update - Priority",
			updateFunc: func(r *SortingRule) {
				r.Priority = 10
			},
			expectError: false,
		},
		{
			name: "Valid Update - Expression",
			updateFunc: func(r *SortingRule) {
				r.Expression = "rarity == 'mythic'"
			},
			expectError: false,
		},
		{
			name: "Valid Update - Enabled",
			updateFunc: func(r *SortingRule) {
				r.Enabled = false
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Empty Name",
			updateFunc: func(r *SortingRule) {
				r.Name = ""
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Empty Expression",
			updateFunc: func(r *SortingRule) {
				r.Expression = ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh rule for each test
			testRule := &SortingRule{
				Name:              "Original Name",
				Priority:          1,
				Expression:        "true",
				StorageLocationID: storage.ID,
				Enabled:           true,
			}
			if err := db.Create(testRule).Error; err != nil {
				t.Fatalf("failed to create rule: %v", err)
			}

			// Apply update
			tt.updateFunc(testRule)

			// Try to save
			err := db.Save(testRule).Error
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

func TestSortingRule_WithStorageLocation(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{
		Name:        "High Value Box",
		StorageType: Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create sorting rule
	rule := &SortingRule{
		Name:              "Expensive Cards",
		Priority:          1,
		Expression:        "prices.usd > 50",
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Verify relationship
	var loadedRule SortingRule
	if err := db.Preload("StorageLocation").First(&loadedRule, rule.ID).Error; err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	if loadedRule.StorageLocation.Name != "High Value Box" {
		t.Errorf("expected storage name 'High Value Box', got '%s'", loadedRule.StorageLocation.Name)
	}
	if loadedRule.StorageLocation.StorageType != Box {
		t.Errorf("expected storage type Box, got %v", loadedRule.StorageLocation.StorageType)
	}
}

func TestSortingRule_DefaultEnabled(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create rule without explicitly setting Enabled
	rule := &SortingRule{
		Name:              "Test Rule",
		Priority:          1,
		Expression:        "true",
		StorageLocationID: storage.ID,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Verify default is true
	var loadedRule SortingRule
	if err := db.First(&loadedRule, rule.ID).Error; err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	if !loadedRule.Enabled {
		t.Error("expected Enabled to default to true")
	}
}

func TestSortingRule_PriorityOrdering(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create rules with different priorities
	rules := []*SortingRule{
		{Name: "Priority 3", Priority: 3, Expression: "true", StorageLocationID: storage.ID, Enabled: true},
		{Name: "Priority 1", Priority: 1, Expression: "true", StorageLocationID: storage.ID, Enabled: true},
		{Name: "Priority 2", Priority: 2, Expression: "true", StorageLocationID: storage.ID, Enabled: true},
	}

	for _, rule := range rules {
		if err := db.Create(rule).Error; err != nil {
			t.Fatalf("failed to create rule: %v", err)
		}
	}

	// Query rules ordered by priority
	var loadedRules []SortingRule
	if err := db.Order("priority ASC").Find(&loadedRules).Error; err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Verify ordering
	if len(loadedRules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(loadedRules))
	}

	expectedPriorities := []int{1, 2, 3}
	for i, rule := range loadedRules {
		if rule.Priority != expectedPriorities[i] {
			t.Errorf("expected priority %d at index %d, got %d", expectedPriorities[i], i, rule.Priority)
		}
	}
}

func TestSortingRule_ComplexExpressions(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Test various expression formats
	expressions := []string{
		"prices.usd > 10",
		"rarity == 'mythic'",
		"len(colors) > 2",
		"set_type == 'core'",
		"prices.usd > 5 && rarity == 'rare'",
		"prices.usd == nil || prices.usd < 1",
	}

	for i, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			rule := &SortingRule{
				Name:              "Test Expression " + expr,
				Priority:          i + 1,
				Expression:        expr,
				StorageLocationID: storage.ID,
				Enabled:           true,
			}
			if err := db.Create(rule).Error; err != nil {
				t.Fatalf("failed to create rule with expression %q: %v", expr, err)
			}

			// Verify it was stored correctly
			var loadedRule SortingRule
			if err := db.First(&loadedRule, rule.ID).Error; err != nil {
				t.Fatalf("failed to load rule: %v", err)
			}

			if loadedRule.Expression != expr {
				t.Errorf("expected expression %q, got %q", expr, loadedRule.Expression)
			}
		})
	}
}

func TestSortingRule_EnabledFiltering(t *testing.T) {
	db := setupSortingRuleTestDB(t)

	// Create storage location
	storage := &StorageLocation{Name: "Test Box", StorageType: Box}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create enabled rules
	enabledRule1 := &SortingRule{
		Name:              "Enabled 1",
		Priority:          1,
		Expression:        "true",
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := db.Create(enabledRule1).Error; err != nil {
		t.Fatalf("failed to create enabled rule 1: %v", err)
	}

	// Create disabled rule (explicitly set to false)
	disabledRule := &SortingRule{
		Name:              "Disabled 1",
		Priority:          2,
		Expression:        "true",
		StorageLocationID: storage.ID,
		Enabled:           false,
	}
	if err := db.Create(disabledRule).Error; err != nil {
		t.Fatalf("failed to create disabled rule: %v", err)
	}

	// Update to set enabled = false (since default is true)
	if err := db.Model(disabledRule).Update("enabled", false).Error; err != nil {
		t.Fatalf("failed to update disabled rule: %v", err)
	}

	enabledRule2 := &SortingRule{
		Name:              "Enabled 2",
		Priority:          3,
		Expression:        "true",
		StorageLocationID: storage.ID,
		Enabled:           true,
	}
	if err := db.Create(enabledRule2).Error; err != nil {
		t.Fatalf("failed to create enabled rule 2: %v", err)
	}

	// Query only enabled rules
	var enabledRules []SortingRule
	if err := db.Where("enabled = ?", true).Find(&enabledRules).Error; err != nil {
		t.Fatalf("failed to load enabled rules: %v", err)
	}

	// Verify only enabled rules were returned
	if len(enabledRules) != 2 {
		t.Errorf("expected 2 enabled rules, got %d", len(enabledRules))
	}

	// Query only disabled rules
	var disabledRules []SortingRule
	if err := db.Where("enabled = ?", false).Find(&disabledRules).Error; err != nil {
		t.Fatalf("failed to load disabled rules: %v", err)
	}

	if len(disabledRules) != 1 {
		t.Errorf("expected 1 disabled rule, got %d", len(disabledRules))
	}
}
