package utils

import (
	"testing"
)

type testEnum string

const (
	enumA testEnum = "alpha"
	enumB testEnum = "beta"
	enumC testEnum = "gamma"
)

func TestConvertEnumSliceToStrings(t *testing.T) {
	t.Run("converts enum values", func(t *testing.T) {
		input := []testEnum{enumA, enumB, enumC}
		result := ConvertEnumSliceToStrings(input)

		expected := []string{"alpha", "beta", "gamma"}
		if len(result) != len(expected) {
			t.Fatalf("expected %d elements, got %d", len(expected), len(result))
		}
		for i, v := range result {
			if v != expected[i] {
				t.Errorf("index %d: expected %q, got %q", i, expected[i], v)
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := ConvertEnumSliceToStrings([]testEnum{})
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(result))
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		var input []testEnum
		result := ConvertEnumSliceToStrings(input)
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(result))
		}
	})

	t.Run("single element", func(t *testing.T) {
		result := ConvertEnumSliceToStrings([]testEnum{enumA})
		if len(result) != 1 || result[0] != "alpha" {
			t.Errorf("expected [\"alpha\"], got %v", result)
		}
	})
}
