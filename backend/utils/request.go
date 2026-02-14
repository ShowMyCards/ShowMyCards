package utils

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// ParseIDParam extracts and validates an integer ID from route parameters.
// Returns 0 and an error if the ID is invalid or missing.
func ParseIDParam(c fiber.Ctx, param string) (uint, error) {
	id := fiber.Params[int](c, param)
	if id <= 0 {
		return 0, fmt.Errorf("invalid %s", param)
	}
	return uint(id), nil
}
