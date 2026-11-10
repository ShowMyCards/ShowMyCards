package utils

import scryfall "github.com/BlueMonday/go-scryfall"

// ExtractCardImageURI returns the URL of the card image the frontend shows.
// For single-faced cards it uses card.ImageURIs. For double-faced cards
// (where card.ImageURIs is nil) it uses the front face,
// card.CardFaces[0].ImageURIs.
//
// The browser loads this URL directly from Scryfall, so it must be an image
// Scryfall allows hotlinking. PNG is therefore never returned.
func ExtractCardImageURI(card scryfall.Card) *string {
	// Try single-faced card images first
	if card.ImageURIs != nil {
		if uri := preferredImageURI(card.ImageURIs); uri != nil {
			return uri
		}
	}

	// Fallback to double-faced card images (front face)
	if len(card.CardFaces) > 0 {
		return preferredImageURI(&card.CardFaces[0].ImageURIs)
	}

	// No images found
	return nil
}

// preferredImageURI picks the first non-empty URI in order of preference:
//
//  1. Grid: WebP, 488 × 680. The size Scryfall recommends.
//  2. JPEG: the JPEG fallback in the same size.
//  3. Normal: the old name for JPEG. Cards stored before Scryfall added the
//     new keys only have this until the next bulk import refreshes them.
func preferredImageURI(uris *scryfall.ImageURIs) *string {
	candidates := []*string{
		&uris.Grid,
		&uris.JPEG,
		&uris.Normal, //nolint:staticcheck // Deprecated field read on purpose as a fallback for rows imported before the new keys existed.
	}
	for _, uri := range candidates {
		if *uri != "" {
			return uri
		}
	}
	return nil
}
