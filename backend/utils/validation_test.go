package utils

import (
	"testing"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
	}{
		{"non-empty", "hello", "field", false},
		{"empty", "", "field", true},
		{"whitespace only", "   ", "field", true},
		{"has content with spaces", " hello ", "field", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired(%q, %q) error = %v, wantErr %v", tt.value, tt.fieldName, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		maxLen  int
		wantErr bool
	}{
		{"under limit", "hi", 5, false},
		{"at limit", "hello", 5, false},
		{"over limit", "hello!", 5, true},
		{"empty string", "", 5, false},
		{"zero limit", "a", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxLength(tt.value, tt.maxLen, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMaxLength(%q, %d) error = %v, wantErr %v", tt.value, tt.maxLen, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNonNegative(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"positive", 5, false},
		{"zero", 0, false},
		{"negative", -1, true},
		{"large negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonNegative(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNonNegative(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNumericParam(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid number", "42", false},
		{"zero", "0", false},
		{"empty string", "", false},
		{"negative", "-1", true},
		{"not a number", "abc", true},
		{"float", "1.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNumericParam(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNumericParam(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestCombineErrors(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		if err := CombineErrors([]error{}); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("all nil errors", func(t *testing.T) {
		if err := CombineErrors([]error{nil, nil, nil}); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("single error", func(t *testing.T) {
		err := CombineErrors([]error{ValidateRequired("", "name")})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("mixed nil and errors", func(t *testing.T) {
		errs := []error{
			nil,
			ValidateRequired("", "name"),
			nil,
			ValidateNonNegative(-1, "age"),
		}
		err := CombineErrors(errs)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
