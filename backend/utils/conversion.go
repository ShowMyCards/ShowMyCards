package utils

// ConvertEnumSliceToStrings converts a slice of any string-based type to a slice of strings
// This is useful for converting enum types (like scryfall.Color, scryfall.Finish) to strings
func ConvertEnumSliceToStrings[T ~string](enums []T) []string {
	result := make([]string, len(enums))
	for i, e := range enums {
		result[i] = string(e)
	}
	return result
}
