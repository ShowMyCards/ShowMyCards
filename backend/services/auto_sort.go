package services

import (
	"backend/models"
	"backend/rules"
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// AutoSortService evaluates sorting rules to determine storage locations for cards.
type AutoSortService struct {
	db *gorm.DB
}

// NewAutoSortService creates a new auto-sort service.
func NewAutoSortService(db *gorm.DB) *AutoSortService {
	return &AutoSortService{db: db}
}

// DetermineStorageLocation evaluates sorting rules for a card and returns the
// matched storage location ID, or nil if no rule matches or the card is not found.
func (s *AutoSortService) DetermineStorageLocation(ctx context.Context, scryfallID, treatment string) (*uint, error) {
	var card models.Card
	if err := s.db.WithContext(ctx).Where("scryfall_id = ?", scryfallID).First(&card).Error; err != nil {
		return nil, fmt.Errorf("card lookup failed: %w", err)
	}

	cardData, err := rules.RawJSONToRuleData(card.RawJSON, treatment)
	if err != nil {
		return nil, fmt.Errorf("card data conversion failed: %w", err)
	}

	evaluator := rules.NewEvaluator(s.db)
	location, err := evaluator.EvaluateCard(cardData)
	if err != nil {
		return nil, fmt.Errorf("no matching rule: %w", err)
	}

	slog.Info("card matched rule, assigning to storage location",
		"component", "auto_sort",
		"scryfall_id", scryfallID,
		"storage_location_id", location.ID,
		"storage_location_name", location.Name,
	)

	return &location.ID, nil
}
