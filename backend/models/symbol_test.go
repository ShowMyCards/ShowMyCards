package models

import "testing"

func TestNormalizeSymbolCode(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"braced tap", "{T}", "T"},
		{"braced white", "{W}", "W"},
		{"braced numeric", "{2}", "2"},
		{"lowercase no braces", "t", "T"},
		{"already normalized", "W", "W"},
		{"surrounding whitespace", "  {U}  ", "U"},
		{"hybrid mana", "{W/U}", "W/U"},
		{"phyrexian", "{W/P}", "W/P"},
		{"empty", "", ""},
		{"empty braces", "{}", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeSymbolCode(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeSymbolCode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeSymbolCode_RoundTripConsistency(t *testing.T) {
	// Storage and lookup must agree: the stored code derived from "{T}"
	// must equal the lookup code derived from "t".
	stored := NormalizeSymbolCode("{T}")
	lookup := NormalizeSymbolCode("t")
	if stored != lookup {
		t.Errorf("stored code %q does not match lookup code %q", stored, lookup)
	}
}
