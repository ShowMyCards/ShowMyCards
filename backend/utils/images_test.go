package utils

import (
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
)

const (
	gridURI   = "https://example.com/grid/card.webp"
	jpegURI   = "https://example.com/normal/card.jpg"
	normalURI = "https://example.com/normal/card_legacy.jpg"
	pngURI    = "https://example.com/png/card.png"
)

func TestExtractCardImageURI_StandardCard(t *testing.T) {
	tests := []struct {
		name string
		uris scryfall.ImageURIs
		want string
	}{
		{
			name: "prefers grid over all other versions",
			uris: scryfall.ImageURIs{Grid: gridURI, JPEG: jpegURI, Normal: normalURI, PNG: pngURI},
			want: gridURI,
		},
		{
			name: "falls back to jpeg without grid",
			uris: scryfall.ImageURIs{JPEG: jpegURI, Normal: normalURI, PNG: pngURI},
			want: jpegURI,
		},
		{
			name: "falls back to normal for rows stored before the new keys",
			uris: scryfall.ImageURIs{Normal: normalURI, PNG: pngURI},
			want: normalURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uris := tt.uris
			result := ExtractCardImageURI(scryfall.Card{ImageURIs: &uris})
			if result == nil {
				t.Fatal("expected image URI, got nil")
			}
			if *result != tt.want {
				t.Errorf("expected %s, got %s", tt.want, *result)
			}
		})
	}
}

// PNG must never be returned: Scryfall does not allow hotlinking it, and the
// browser loads the returned URL directly.
func TestExtractCardImageURI_NeverReturnsPNG(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{PNG: pngURI},
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}

func TestExtractCardImageURI_DoubleFaced_UsesFrontFace(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil, // Double-faced cards have nil ImageURIs
		CardFaces: []scryfall.CardFace{
			{
				ImageURIs: scryfall.ImageURIs{
					Grid: "https://example.com/grid/front.webp",
					JPEG: "https://example.com/normal/front.jpg",
					PNG:  "https://example.com/png/front.png",
				},
			},
			{
				ImageURIs: scryfall.ImageURIs{
					Grid: "https://example.com/grid/back.webp",
					JPEG: "https://example.com/normal/back.jpg",
					PNG:  "https://example.com/png/back.png",
				},
			},
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/grid/front.webp" {
		t.Errorf("expected front face grid URI, got %s", *result)
	}
}

func TestExtractCardImageURI_DoubleFaced_FallsBackToNormal(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil,
		CardFaces: []scryfall.CardFace{
			{
				ImageURIs: scryfall.ImageURIs{
					Normal: "https://example.com/normal/front.jpg",
					PNG:    "https://example.com/png/front.png",
				},
			},
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/normal/front.jpg" {
		t.Errorf("expected front face normal URI, got %s", *result)
	}
}

// A card with a non-nil but empty top-level ImageURIs still falls back to
// its front face.
func TestExtractCardImageURI_EmptyTopLevel_FallsBackToFrontFace(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{},
		CardFaces: []scryfall.CardFace{
			{ImageURIs: scryfall.ImageURIs{Grid: "https://example.com/grid/front.webp"}},
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/grid/front.webp" {
		t.Errorf("expected front face grid URI, got %s", *result)
	}
}

func TestExtractCardImageURI_NoImages(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{},
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}

func TestExtractCardImageURI_EmptyCardFaces(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil,
		CardFaces: []scryfall.CardFace{},
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}

func TestExtractCardImageURI_NilImageURIs_EmptyCardFaces(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil,
		CardFaces: nil,
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}

func TestExtractCardImageURI_DoubleFaced_EmptyFaceImages(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil,
		CardFaces: []scryfall.CardFace{
			{ImageURIs: scryfall.ImageURIs{}},
		},
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}
