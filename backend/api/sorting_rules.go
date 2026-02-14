package api

import (
	"backend/models"
	"backend/rules"
	"backend/utils"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SortingRulesHandler handles sorting rule endpoints
type SortingRulesHandler struct {
	db *gorm.DB
}

// NewSortingRulesHandler creates a new sorting rules handler
func NewSortingRulesHandler(db *gorm.DB) *SortingRulesHandler {
	return &SortingRulesHandler{db: db}
}

// List returns sorting rules with pagination, ordered by priority
func (h *SortingRulesHandler) List(c fiber.Ctx) error {
	params := utils.ParsePaginationParams(c, utils.DefaultPageSize, utils.MaxPageSize)

	// Optional filter by enabled status
	enabled := c.Query("enabled")

	query := h.db.WithContext(c.RequestCtx()).Model(&models.SortingRule{})
	if enabled != "" {
		if enabled != "true" && enabled != "false" {
			return utils.ReturnError(c, fiber.StatusBadRequest, "enabled must be 'true' or 'false'")
		}
		query = query.Where("enabled = ?", enabled == "true")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to count sorting rules", "database count failed", err)
	}

	var rules []models.SortingRule
	offset := utils.CalculateOffset(params.Page, params.PageSize)
	if err := query.Order("priority ASC").
		Preload("StorageLocation").
		Offset(offset).
		Limit(params.PageSize).
		Find(&rules).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch sorting rules", "database query failed", err)
	}

	response := utils.NewPaginatedResponse(rules, params.Page, params.PageSize, total)
	return c.JSON(response)
}

// Get returns a single sorting rule by ID
func (h *SortingRulesHandler) Get(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var rule models.SortingRule
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "sorting rule not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch sorting rule", "database query failed", err)
	}
	return c.JSON(rule)
}

// CreateSortingRuleRequest represents the request body for creating a sorting rule
type CreateSortingRuleRequest struct {
	Name              string `json:"name"`
	Priority          int    `json:"priority"`
	Expression        string `json:"expression"`
	StorageLocationID uint   `json:"storage_location_id"`
	Enabled           *bool  `json:"enabled,omitempty"`
}

// Create creates a new sorting rule
func (h *SortingRulesHandler) Create(c fiber.Ctx) error {
	var req CreateSortingRuleRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	var validationErrors []error
	validationErrors = append(validationErrors, utils.ValidateRequired(req.Name, "name"))
	validationErrors = append(validationErrors, utils.ValidateRequired(req.Expression, "expression"))

	if err := utils.CombineErrors(validationErrors); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validate expression syntax
	evaluator := rules.NewEvaluator(h.db)
	if err := evaluator.ValidateExpression(req.Expression); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid expression: "+err.Error())
	}

	// Validate storage location exists
	var location models.StorageLocation
	if err := h.db.WithContext(c.RequestCtx()).First(&location, req.StorageLocationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusBadRequest, "storage location not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to validate storage location", "storage location lookup failed", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := models.SortingRule{
		Name:              req.Name,
		Priority:          req.Priority,
		Expression:        req.Expression,
		StorageLocationID: req.StorageLocationID,
		Enabled:           enabled,
	}

	if err := h.db.WithContext(c.RequestCtx()).Create(&rule).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to create sorting rule", "database insert failed", err)
	}

	// Reload with storage location
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&rule, rule.ID).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to reload sorting rule", "database query failed", err)
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateSortingRuleRequest represents the request body for updating a sorting rule
type UpdateSortingRuleRequest struct {
	Name              *string `json:"name,omitempty"`
	Priority          *int    `json:"priority,omitempty"`
	Expression        *string `json:"expression,omitempty"`
	StorageLocationID *uint   `json:"storage_location_id,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
}

// Update updates an existing sorting rule
func (h *SortingRulesHandler) Update(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	var rule models.SortingRule
	if err := h.db.WithContext(c.RequestCtx()).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, "sorting rule not found")
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to fetch sorting rule", "database query failed", err)
	}

	var req UpdateSortingRuleRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Update fields if provided
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Expression != nil {
		rule.Expression = *req.Expression
	}
	if req.StorageLocationID != nil {
		// Validate storage location exists
		var location models.StorageLocation
		if err := h.db.WithContext(c.RequestCtx()).First(&location, *req.StorageLocationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ReturnError(c, fiber.StatusBadRequest, "storage location not found")
			}
			return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
				"Failed to validate storage location", "database query failed", err)
		}
		rule.StorageLocationID = *req.StorageLocationID
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.db.WithContext(c.RequestCtx()).Save(&rule).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update sorting rule", "database update failed", err)
	}

	// Reload with storage location
	if err := h.db.WithContext(c.RequestCtx()).Preload("StorageLocation").First(&rule, rule.ID).Error; err != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to reload sorting rule", "database query failed", err)
	}

	return c.JSON(rule)
}

// Delete deletes a sorting rule
func (h *SortingRulesHandler) Delete(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid id")
	}

	result := h.db.WithContext(c.RequestCtx()).Delete(&models.SortingRule{}, id)
	if result.Error != nil {
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to delete sorting rule", "database delete failed", result.Error)
	}

	if result.RowsAffected == 0 {
		return utils.ReturnError(c, fiber.StatusNotFound, "sorting rule not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// EvaluateRequest represents the request body for evaluating a card against rules
type EvaluateRequest struct {
	CardData  map[string]interface{} `json:"card_data"`
	Treatment string                 `json:"treatment,omitempty"` // Optional treatment (foil, nonfoil, etched, etc.)
}

// EvaluateResponse represents the response for rule evaluation
type EvaluateResponse struct {
	Matched         bool                    `json:"matched"`
	StorageLocation *models.StorageLocation `json:"storage_location,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

// Evaluate evaluates card data against all enabled rules and returns the matching storage location
func (h *SortingRulesHandler) Evaluate(c fiber.Ctx) error {
	var req EvaluateRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.CardData) == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "card_data is required")
	}

	// Merge treatment into card data if provided
	cardData := req.CardData
	if req.Treatment != "" {
		cardData["treatment"] = req.Treatment
	}

	// Convert price strings to numbers for expression evaluation
	// Scryfall returns prices as strings (e.g., "83.73"), but expressions need numbers
	if prices, ok := cardData["prices"].(map[string]interface{}); ok {
		convertedPrices := make(map[string]interface{})
		for key, value := range prices {
			if strValue, ok := value.(string); ok {
				// Parse string to float64
				var floatValue float64
				if _, err := fmt.Sscanf(strValue, "%f", &floatValue); err == nil {
					convertedPrices[key] = floatValue
				} else {
					convertedPrices[key] = nil // Keep null values as nil
				}
			} else {
				convertedPrices[key] = value // Keep non-string values as-is
			}
		}
		cardData["prices"] = convertedPrices
	}

	evaluator := rules.NewEvaluator(h.db)
	location, err := evaluator.EvaluateCard(cardData)

	if err != nil {
		return c.JSON(EvaluateResponse{
			Matched: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(EvaluateResponse{
		Matched:         true,
		StorageLocation: location,
	})
}

// ValidateExpressionRequest represents the request body for validating an expression
type ValidateExpressionRequest struct {
	Expression string `json:"expression"`
}

// ValidateExpressionResponse represents the response for expression validation
type ValidateExpressionResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// ValidateExpression validates a rule expression without evaluating it
func (h *SortingRulesHandler) ValidateExpression(c fiber.Ctx) error {
	var req ValidateExpressionRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	evaluator := rules.NewEvaluator(h.db)
	err := evaluator.ValidateExpression(req.Expression)

	if err != nil {
		return c.JSON(ValidateExpressionResponse{
			Valid: false,
			Error: err.Error(),
		})
	}

	return c.JSON(ValidateExpressionResponse{
		Valid: true,
	})
}

// BatchUpdatePriorityItem represents a single rule priority update
type BatchUpdatePriorityItem struct {
	ID       uint `json:"id"`
	Priority int  `json:"priority"`
}

// BatchUpdatePrioritiesRequest represents the request body for batch updating priorities
type BatchUpdatePrioritiesRequest struct {
	Updates []BatchUpdatePriorityItem `json:"updates"`
}

// BatchUpdatePrioritiesResponse represents the response for batch priority updates
type BatchUpdatePrioritiesResponse struct {
	UpdatedCount int    `json:"updated_count"`
	Error        string `json:"error,omitempty"`
}

// BatchUpdatePriorities updates priorities for multiple rules in a single transaction
func (h *SortingRulesHandler) BatchUpdatePriorities(c fiber.Ctx) error {
	var req BatchUpdatePrioritiesRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.ReturnError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.Updates) == 0 {
		return utils.ReturnError(c, fiber.StatusBadRequest, "updates array cannot be empty")
	}

	// Use a transaction to ensure all updates succeed or all fail
	err := h.db.WithContext(c.RequestCtx()).Transaction(func(tx *gorm.DB) error {
		for _, update := range req.Updates {
			// Check if rule exists
			var rule models.SortingRule
			if err := tx.First(&rule, update.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("sorting rule with id %d not found: %w", update.ID, err)
				}
				return fmt.Errorf("failed to fetch sorting rule %d: %w", update.ID, err)
			}

			// Update priority
			if err := tx.Model(&rule).Update("priority", update.Priority).Error; err != nil {
				return fmt.Errorf("failed to update priority for rule %d: %w", update.ID, err)
			}
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ReturnError(c, fiber.StatusNotFound, err.Error())
		}
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to update priorities", "batch priority update failed", err)
	}

	return c.JSON(BatchUpdatePrioritiesResponse{
		UpdatedCount: len(req.Updates),
	})
}
