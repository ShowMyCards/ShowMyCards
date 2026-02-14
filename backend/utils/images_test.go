package utils

import (
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
)

func TestExtractCardImageURI_StandardCard_WithPNG(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{
			PNG:    "https://example.com/card.png",
			Normal: "https://example.com/card_normal.jpg",
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/card.png" {
		t.Errorf("expected PNG URI, got %s", *result)
	}
}

func TestExtractCardImageURI_StandardCard_WithoutPNG(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{
			PNG:    "",
			Normal: "https://example.com/card_normal.jpg",
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/card_normal.jpg" {
		t.Errorf("expected Normal URI, got %s", *result)
	}
}

func TestExtractCardImageURI_DoubleFaced_WithPNG(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil, // Double-faced cards have nil ImageURIs
		CardFaces: []scryfall.CardFace{
			{
				ImageURIs: scryfall.ImageURIs{
					PNG:    "https://example.com/front.png",
					Normal: "https://example.com/front_normal.jpg",
				},
			},
			{
				ImageURIs: scryfall.ImageURIs{
					PNG:    "https://example.com/back.png",
					Normal: "https://example.com/back_normal.jpg",
				},
			},
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/front.png" {
		t.Errorf("expected front face PNG URI, got %s", *result)
	}
}

func TestExtractCardImageURI_DoubleFaced_WithoutPNG(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: nil,
		CardFaces: []scryfall.CardFace{
			{
				ImageURIs: scryfall.ImageURIs{
					PNG:    "",
					Normal: "https://example.com/front_normal.jpg",
				},
			},
		},
	}

	result := ExtractCardImageURI(card)
	if result == nil {
		t.Fatal("expected image URI, got nil")
	}
	if *result != "https://example.com/front_normal.jpg" {
		t.Errorf("expected front face Normal URI, got %s", *result)
	}
}

func TestExtractCardImageURI_NoImages(t *testing.T) {
	card := scryfall.Card{
		ImageURIs: &scryfall.ImageURIs{
			PNG:    "",
			Normal: "",
		},
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
			{
				ImageURIs: scryfall.ImageURIs{
					PNG:    "",
					Normal: "",
				},
			},
		},
	}

	result := ExtractCardImageURI(card)
	if result != nil {
		t.Errorf("expected nil, got %s", *result)
	}
}
