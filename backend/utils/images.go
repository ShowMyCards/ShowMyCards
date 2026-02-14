package utils

import scryfall "github.com/BlueMonday/go-scryfall"

// ExtractCardImageURI extracts the best available image URI from a Scryfall card.
// For single-faced cards, uses card.ImageURIs. For double-faced cards, uses
// the front face (card.CardFaces[0].ImageURIs). Prefers PNG over Normal.
func ExtractCardImageURI(card scryfall.Card) *string {
	// Try single-faced card images first
	if card.ImageURIs != nil {
		if card.ImageURIs.PNG != "" {
			return &card.ImageURIs.PNG
		}
		if card.ImageURIs.Normal != "" {
			return &card.ImageURIs.Normal
		}
	}

	// Fallback to double-faced card images (front face)
	if len(card.CardFaces) > 0 {
		frontFace := card.CardFaces[0]
		if frontFace.ImageURIs.PNG != "" {
			return &frontFace.ImageURIs.PNG
		}
		if frontFace.ImageURIs.Normal != "" {
			return &frontFace.ImageURIs.Normal
		}
	}

	// No images found
	return nil
}
