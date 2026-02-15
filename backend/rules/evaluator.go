// Package rules provides rule evaluation engine for automatic card sorting.
// It uses expr-lang to evaluate expressions against Scryfall card data.
package rules

import (
	"backend/models"
	"context"
	"fmt"

	"github.com/expr-lang/expr"
	"gorm.io/gorm"
)

// Evaluator handles rule evaluation for card sorting.
// A new Evaluator should be created per request; it is not safe for concurrent use.
type Evaluator struct {
	db *gorm.DB
}

// NewEvaluator creates a new rule evaluator
func NewEvaluator(db *gorm.DB) *Evaluator {
	return &Evaluator{db: db}
}

// EvaluateCard evaluates a card against all enabled rules and returns the matching storage location.
// It fetches rules from the database on each call — use EvaluateCardWithRules for batch operations.
func (e *Evaluator) EvaluateCard(ctx context.Context, cardData map[string]interface{}) (*models.StorageLocation, error) {
	var rules []models.SortingRule
	if err := e.db.WithContext(ctx).Where("enabled = ?", true).
		Order("priority ASC").
		Preload("StorageLocation").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sorting rules: %w", err)
	}
	return e.EvaluateCardWithRules(cardData, rules)
}

// EvaluateCardWithRules evaluates a card against the provided rules and returns the matching storage location.
// Use this for batch operations to avoid re-fetching rules on every call.
func (e *Evaluator) EvaluateCardWithRules(cardData map[string]interface{}, rules []models.SortingRule) (*models.StorageLocation, error) {
	for _, rule := range rules {
		matches, err := e.evaluateExpression(rule.Expression, cardData)
		if err != nil {
			continue
		}

		if matches {
			return &rule.StorageLocation, nil
		}
	}

	return nil, fmt.Errorf("no matching rule found for card")
}

// EvaluateExpression evaluates a single expression against card data
func (e *Evaluator) EvaluateExpression(expression string, cardData map[string]interface{}) (bool, error) {
	return e.evaluateExpression(expression, cardData)
}

// evaluateExpression is the internal implementation
func (e *Evaluator) evaluateExpression(expression string, cardData map[string]interface{}) (bool, error) {
	if expression == "" {
		return false, fmt.Errorf("expression cannot be empty")
	}

	// Add helper functions to the environment
	env := make(map[string]interface{})
	for k, v := range cardData {
		env[k] = v
	}

	// Add helper functions
	env["hasColor"] = func(color string) bool {
		return hasColor(cardData, color)
	}
	env["isMonoColor"] = func() bool {
		return isMonoColor(cardData)
	}
	env["isMultiColor"] = func() bool {
		return isMultiColor(cardData)
	}
	env["isColorless"] = func() bool {
		return isColorless(cardData)
	}
	env["isColor"] = func(colors ...string) bool {
		return isColor(cardData, colors...)
	}

	// Compile the expression
	program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("failed to compile expression: %w", err)
	}

	// Evaluate the expression
	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	// Assert result as boolean
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("expression did not return a boolean value")
	}

	return result, nil
}

// Helper functions for rule expressions
// These check color_identity which works correctly for both single-faced and double-faced cards

// hasColor checks if a card has a specific color in its color identity
// Works with both single-faced cards and double-faced cards
// Usage: hasColor("W") or hasColor("U")
func hasColor(cardData map[string]interface{}, targetColor string) bool {
	// Check color_identity (works for all cards, including DFCs)
	if colorIdentity, ok := cardData["color_identity"].([]interface{}); ok {
		for _, c := range colorIdentity {
			if str, ok := c.(string); ok && str == targetColor {
				return true
			}
		}
	}
	return false
}

// isMonoColor checks if a card is exactly one color
// Usage: isMonoColor()
func isMonoColor(cardData map[string]interface{}) bool {
	if colorIdentity, ok := cardData["color_identity"].([]interface{}); ok {
		return len(colorIdentity) == 1
	}
	return false
}

// isMultiColor checks if a card is multicolored (2+ colors)
// Usage: isMultiColor()
func isMultiColor(cardData map[string]interface{}) bool {
	if colorIdentity, ok := cardData["color_identity"].([]interface{}); ok {
		return len(colorIdentity) >= 2
	}
	return false
}

// isColorless checks if a card is colorless
// Usage: isColorless()
func isColorless(cardData map[string]interface{}) bool {
	if colorIdentity, ok := cardData["color_identity"].([]interface{}); ok {
		return len(colorIdentity) == 0
	}
	return false
}

// isColor checks if a card's color identity matches exactly the provided colors
// Checks both colors and color_identity fields, returns true if either matches
// Usage: isColor("W") for mono-white, isColor("W", "U") for Azorius
func isColor(cardData map[string]interface{}, targetColors ...string) bool {
	// Helper to check if two color sets match exactly (order independent)
	colorsMatch := func(cardColors []interface{}, targets []string) bool {
		if len(cardColors) != len(targets) {
			return false
		}

		// Build a set of target colors
		targetSet := make(map[string]bool)
		for _, t := range targets {
			targetSet[t] = true
		}

		// Check all card colors are in target set
		for _, c := range cardColors {
			if str, ok := c.(string); ok {
				if !targetSet[str] {
					return false
				}
				delete(targetSet, str)
			}
		}

		// All targets should be matched
		return len(targetSet) == 0
	}

	// Check color_identity first (most reliable for commander-style sorting)
	if colorIdentity, ok := cardData["color_identity"].([]interface{}); ok {
		if colorsMatch(colorIdentity, targetColors) {
			return true
		}
	}

	// Also check colors field
	if colors, ok := cardData["colors"].([]interface{}); ok {
		if colorsMatch(colors, targetColors) {
			return true
		}
	}

	return false
}

// ValidateExpression validates an expression without evaluating it
func (e *Evaluator) ValidateExpression(expression string) error {
	if expression == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	if len(expression) > 1000 {
		return fmt.Errorf("expression too long (max 1000 characters)")
	}

	// Check nesting depth as a complexity heuristic
	depth := 0
	maxDepth := 0
	for _, ch := range expression {
		if ch == '(' {
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		} else if ch == ')' {
			depth--
		}
	}
	if maxDepth > 20 {
		return fmt.Errorf("expression too complex (max 20 levels of nesting)")
	}

	// Try to compile with a comprehensive sample environment matching Scryfall API + inventory fields
	sampleEnv := map[string]interface{}{
		// Price fields
		"prices": map[string]interface{}{
			"usd":        0.0,
			"usd_foil":   0.0,
			"usd_etched": 0.0,
			"eur":        0.0,
			"eur_foil":   0.0,
			"tix":        0.0,
		},
		// Card properties
		"name":             "",
		"rarity":           "",
		"set":              "",
		"set_name":         "",
		"set_type":         "",
		"type_line":        "",
		"oracle_text":      "",
		"mana_cost":        "",
		"cmc":              0.0,
		"power":            "",
		"toughness":        "",
		"colors":           []string{},
		"color_identity":   []string{},
		"keywords":         []string{},
		"finishes":         []string{},
		"promo_types":      []string{},
		"edhrec_rank":      0,
		"artist":           "",
		"collector_number": "",
		"frame":            "",
		"border_color":     "",
		"layout":           "",
		"reserved":         false,
		"foil":             false,
		"nonfoil":          false,
		"oversized":        false,
		"promo":            false,
		"reprint":          false,
		"digital":          false,
		"full_art":         false,
		"textless":         false,
		"booster":          false,

		// Inventory-specific fields
		"treatment": "", // "foil", "nonfoil", "etched", etc.
		"quantity":  0,

		// Helper functions
		"hasColor": func(color string) bool {
			return false
		},
		"isMonoColor": func() bool {
			return false
		},
		"isMultiColor": func() bool {
			return false
		},
		"isColorless": func() bool {
			return false
		},
		"isColor": func(colors ...string) bool {
			return false
		},
	}

	_, err := expr.Compile(expression, expr.Env(sampleEnv), expr.AsBool())
	if err != nil {
		return fmt.Errorf("invalid expression: %w", err)
	}

	return nil
}
