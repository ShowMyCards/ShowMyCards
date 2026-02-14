package utils

import (
	"math"
	"testing"
)

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		pageSize int
		expected int
	}{
		{"zero items", 0, 20, 0},
		{"one item", 1, 20, 1},
		{"exactly one page", 20, 20, 1},
		{"one over page boundary", 21, 20, 2},
		{"exact multiple", 100, 10, 10},
		{"large dataset", 999, 50, 20},
		{"page size zero returns zero", 100, 0, 0},
		{"page size negative returns zero", 100, -1, 0},
		{"single item pages", 5, 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTotalPages(tt.total, tt.pageSize)
			if result != tt.expected {
				t.Errorf("CalculateTotalPages(%d, %d) = %d, want %d", tt.total, tt.pageSize, result, tt.expected)
			}
		})
	}
}

func TestCalculateOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		expected int
	}{
		{"first page", 1, 20, 0},
		{"second page", 2, 20, 20},
		{"third page size 10", 3, 10, 20},
		{"page zero clamps to 1", 0, 20, 0},
		{"negative page clamps to 1", -1, 20, 0},
		{"page size zero uses default", 2, 0, DefaultPageSize},
		{"page size negative uses default", 2, -1, DefaultPageSize},
		{"large page number", 100, 50, 4950},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateOffset(tt.page, tt.pageSize)
			if result != tt.expected {
				t.Errorf("CalculateOffset(%d, %d) = %d, want %d", tt.page, tt.pageSize, result, tt.expected)
			}
		})
	}
}

func TestCalculateOffset_Overflow(t *testing.T) {
	// When page * pageSize would overflow, should return a safe value
	result := CalculateOffset(math.MaxInt, 20)
	if result < 0 {
		t.Errorf("overflow case returned negative offset: %d", result)
	}
	if result != math.MaxInt-20 {
		t.Errorf("expected MaxInt-20 (%d), got %d", math.MaxInt-20, result)
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	data := []string{"a", "b", "c"}
	resp := NewPaginatedResponse(data, 2, 10, 25)

	if resp.Page != 2 {
		t.Errorf("expected Page 2, got %d", resp.Page)
	}
	if resp.PageSize != 10 {
		t.Errorf("expected PageSize 10, got %d", resp.PageSize)
	}
	if resp.TotalItems != 25 {
		t.Errorf("expected TotalItems 25, got %d", resp.TotalItems)
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected TotalPages 3, got %d", resp.TotalPages)
	}

	// Verify data is passed through (now typed as []string directly)
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 items in Data, got %d", len(resp.Data))
	}
}
