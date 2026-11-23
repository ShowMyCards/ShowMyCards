package models

import (
	"context"
	"errors"
	"fmt"
	"strings"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
)

// A deck extra is a non-playable object that travels with a paper deck but is
// never one of its cards: tokens, emblems, helper cards (The Monarch, Day/Night,
// Dungeons, ...), art cards, Jumpstart front cards, and planes, schemes and
// vanguards. Extras always live in ZoneOther and count as items, not cards.

var (
	// ErrExtraOutsideOther rejects putting a token or other extra into a card zone.
	ErrExtraOutsideOther = errors.New("tokens and other extras always live in the other zone")
	// ErrCardInOther rejects putting a real card into the other zone.
	ErrCardInOther = errors.New("only tokens and other extras can go in the other zone")
)

// extraLayouts are the Scryfall layouts used only for non-playable objects.
var extraLayouts = map[scryfall.Layout]bool{
	scryfall.LayoutToken:            true,
	scryfall.LayoutDoubleFacedToken: true,
	scryfall.LayoutEmblem:           true,
	scryfall.LayoutArtSeries:        true,
	scryfall.LayoutPlanar:           true,
	scryfall.LayoutScheme:           true,
	scryfall.LayoutVanguard:         true,
	scryfall.Layout("front_card"):   true, // Jumpstart theme cards; not in go-scryfall yet
}

// extraTypes are card types printed only on non-playable objects. They catch
// extras whose layout says otherwise: Role tokens with a flip layout, Dungeons
// and helper cards ("Card") with a normal layout, reversible tokens.
var extraTypes = map[string]bool{
	"Token":   true,
	"Emblem":  true,
	"Card":    true,
	"Dungeon": true,
}

// IsDeckExtra reports whether card is a deck extra rather than a deck card.
func IsDeckExtra(card scryfall.Card) bool {
	if extraLayouts[card.Layout] {
		return true
	}
	typeLines := strings.Split(card.TypeLine, "//")
	for _, face := range card.CardFaces {
		typeLines = append(typeLines, face.TypeLine)
	}
	for _, line := range typeLines {
		if hasExtraType(line) {
			return true
		}
	}
	return false
}

// hasExtraType reports whether a single face's type line has an extra-only type.
// Only the words before the dash are types; subtypes after it are ignored.
func hasExtraType(typeLine string) bool {
	types, _, _ := strings.Cut(typeLine, "—")
	for _, word := range strings.Fields(types) {
		if extraTypes[word] {
			return true
		}
	}
	return false
}

// CheckExtraZone enforces that extras live in ZoneOther and real cards never do.
func CheckExtraZone(isExtra bool, zone DeckZone) error {
	switch {
	case isExtra && zone != ZoneOther:
		return ErrExtraOutsideOther
	case !isExtra && zone == ZoneOther:
		return ErrCardInOther
	}
	return nil
}

// ClassifyDeckItems reports, by index into items, whether each item's card is a
// deck extra. Pinned items are classified from their printing; any-printing
// items, and pinned items whose printing is missing locally, from a
// representative printing of their Oracle. Items with no local card data are
// absent from the result.
func ClassifyDeckItems(db *gorm.DB, items []DeckItem) (map[int]bool, error) {
	result := make(map[int]bool, len(items))
	if len(items) == 0 {
		return result, nil
	}

	pinnedIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.ScryfallID != "" {
			pinnedIDs = append(pinnedIDs, item.ScryfallID)
		}
	}
	pinned, err := GetScryfallCardsByIDs(db, pinnedIDs)
	if err != nil {
		return nil, fmt.Errorf("classifying deck items: %w", err)
	}

	var oracleIDs []string
	seen := make(map[string]bool)
	for i, item := range items {
		if card, ok := pinned[item.ScryfallID]; ok {
			result[i] = IsDeckExtra(card)
			continue
		}
		if !seen[item.OracleID] {
			seen[item.OracleID] = true
			oracleIDs = append(oracleIDs, item.OracleID)
		}
	}
	if len(oracleIDs) == 0 {
		return result, nil
	}

	byOracle, err := GetPreferredCardsByOracle(db, oracleIDs, "en", false)
	if err != nil {
		return nil, fmt.Errorf("classifying deck items: %w", err)
	}
	for i, item := range items {
		if _, done := result[i]; done {
			continue
		}
		if card, ok := byOracle[item.OracleID]; ok {
			result[i] = IsDeckExtra(card)
		}
	}
	return result, nil
}

// ReclassifyResult reports what ReclassifyDeckExtras did.
type ReclassifyResult struct {
	Checked int // deck item rows examined
	Moved   int // rows moved (or merged) into their correct zone
}

// ReclassifyDeckExtras moves every deck item whose card is an extra into
// ZoneOther, and every real card found in ZoneOther back to ZoneMain, merging
// with an existing row in the target zone where needed. Items without local
// card data are left alone. It is idempotent and runs in one transaction.
//
// It runs after a Scryfall bulk import, so items added before card data was
// available (or before extras had their own zone) end up in the right place.
func ReclassifyDeckExtras(ctx context.Context, db *gorm.DB) (ReclassifyResult, error) {
	var result ReclassifyResult
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var items []DeckItem
		if err := tx.Order("id").Find(&items).Error; err != nil {
			return fmt.Errorf("loading deck items: %w", err)
		}
		result.Checked = len(items)

		extras, err := ClassifyDeckItems(tx, items)
		if err != nil {
			return err
		}

		for i, item := range items {
			isExtra, known := extras[i]
			if !known || CheckExtraZone(isExtra, item.Zone) == nil {
				continue
			}
			zone := ZoneMain
			if isExtra {
				zone = ZoneOther
			}
			// Reload: an earlier move may have merged into or changed this row.
			var source DeckItem
			if err := tx.First(&source, item.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("reloading deck item %d: %w", item.ID, err)
			}
			if source.Zone == zone {
				continue
			}
			if _, err := MoveDeckItemTx(tx, source, zone, source.DesiredQuantity); err != nil {
				return fmt.Errorf("moving deck item %d to %s: %w", item.ID, zone, err)
			}
			result.Moved++
		}
		return nil
	})
	if err != nil {
		return ReclassifyResult{}, err
	}
	return result, nil
}
