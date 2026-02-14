package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ValidateRequired checks field is not empty
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateMaxLength checks field length
func ValidateMaxLength(value string, maxLen int, fieldName string) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s must be %d characters or less", fieldName, maxLen)
	}
	return nil
}

// ValidateNonNegative checks number is non-negative
func ValidateNonNegative(value int, fieldName string) error {
	if value < 0 {
		return fmt.Errorf("%s cannot be negative", fieldName)
	}
	return nil
}

// ValidateNumericParam checks that a query parameter is a valid positive integer (if non-empty)
func ValidateNumericParam(value, fieldName string) error {
	if value == "" {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fmt.Errorf("%s must be a valid non-negative integer", fieldName)
	}
	return nil
}

// CombineErrors combines multiple validation errors.
// Returns nil if all errors are nil. The returned error preserves the
// original errors for inspection with errors.Is and errors.As.
func CombineErrors(errs []error) error {
	nonNil := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	return errors.Join(nonNil...)
}
