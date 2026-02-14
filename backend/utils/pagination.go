package utils

import (
	"math"

	"github.com/gofiber/fiber/v3"
)

// Default pagination constants
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Page     int
	PageSize int
}

// ParsePaginationParams extracts and validates pagination from request
func ParsePaginationParams(c fiber.Ctx, defaultSize, maxSize int) PaginationParams {
	page := fiber.Query[int](c, "page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := fiber.Query[int](c, "page_size", defaultSize)
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// CalculateTotalPages computes total pages (consistent formula)
func CalculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

// CalculateOffset computes database offset with overflow protection
func CalculateOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if page > math.MaxInt/pageSize {
		return math.MaxInt - pageSize
	}
	return (page - 1) * pageSize
}

// PaginatedResponse standard response wrapper
type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginatedResponse creates response
func NewPaginatedResponse[T any](data []T, page, pageSize int, totalItems int64) PaginatedResponse[T] {
	return PaginatedResponse[T]{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: CalculateTotalPages(totalItems, pageSize),
	}
}
