package legality

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// unlimited is the copy limit of cards a deck may hold any number of.
const unlimited = math.MaxInt

var (
	anyNumberRe = regexp.MustCompile(`(?i)deck can have any number of cards named`)
	upToRe      = regexp.MustCompile(`(?i)deck can have up to (\w+) cards named`)
	onlyOneRe   = regexp.MustCompile(`(?i)deck can have only one copy`)
	// Scryfall gives every partner variant the "Partner" keyword; the variant
	// is only spelled out in the rules text.
	partnerWithRe    = regexp.MustCompile(`Partner with ([^(\n]+)`)
	partnerVariantRe = regexp.MustCompile(`Partner—([^(\n]+)`)
)

var numberWords = map[string]int{
	"two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
	"seven": 7, "eight": 8, "nine": 9, "ten": 10,
}

// frontFace returns the type line, rules text and power that decide what a
// card is: the front face's for multi-faced cards (CR 903.3 looks at the front
// face only).
func frontFace(c scryfall.Card) (typeLine, text string, power *string) {
	if len(c.CardFaces) > 0 {
		f := c.CardFaces[0]
		if f.OracleText != nil {
			text = *f.OracleText
		}
		return f.TypeLine, text, f.Power
	}
	return c.TypeLine, c.OracleText, c.Power
}

// allText joins the rules text of every face.
func allText(c scryfall.Card) string {
	parts := []string{c.OracleText}
	for _, f := range c.CardFaces {
		if f.OracleText != nil {
			parts = append(parts, *f.OracleText)
		}
	}
	return strings.Join(parts, "\n")
}

// splitTypeLine splits a type line such as "Legendary Creature — Time Lord
// Doctor" into its supertypes/types and its subtypes.
func splitTypeLine(typeLine string) (types, subtypes []string) {
	before, after, _ := strings.Cut(typeLine, "—")
	return strings.Fields(before), strings.Fields(after)
}

// frontTypes returns the front face's supertypes/types and subtypes.
func frontTypes(c scryfall.Card) (types, subtypes []string) {
	typeLine, _, _ := frontFace(c)
	return splitTypeLine(typeLine)
}

// isNonDeckCard reports whether the card is a printed reminder rather than a
// card a deck is built from: tokens, emblems, counter and marker cards (e.g.
// Poison Counter, Experience), art cards and Jumpstart theme cards. Most have a
// dedicated layout; the rest carry the Token supertype (e.g. Role tokens) or
// Scryfall's "Card" type line, which no game card has.
func isNonDeckCard(c scryfall.Card) bool {
	switch c.Layout {
	case scryfall.LayoutToken, scryfall.LayoutDoubleFacedToken, scryfall.LayoutEmblem,
		scryfall.LayoutArtSeries, scryfall.Layout("front_card"):
		return true
	}
	types, _ := frontTypes(c)
	return slices.Contains(types, "Token") || slices.Contains(types, "Card")
}

func hasKeyword(c scryfall.Card, keyword string) bool {
	return slices.ContainsFunc(c.Keywords, func(k string) bool { return strings.EqualFold(k, keyword) })
}

// isBasic reports whether the card has the Basic supertype. This covers
// Wastes and the Snow-Covered basics, which the copy limit also exempts.
func isBasic(c scryfall.Card) bool {
	types, _ := frontTypes(c)
	return slices.Contains(types, "Basic")
}

// copyLimit returns how many copies of the card a deck may hold, given the
// format's default limit. Basic lands and cards that say so in their rules
// text override the default (CR 100.2a, 903.5b).
func copyLimit(c scryfall.Card, defaultLimit int) int {
	if isBasic(c) {
		return unlimited
	}
	text := allText(c)
	if anyNumberRe.MatchString(text) {
		return unlimited
	}
	if m := upToRe.FindStringSubmatch(text); m != nil {
		if n, ok := numberWords[strings.ToLower(m[1])]; ok {
			return n
		}
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n
		}
	}
	if onlyOneRe.MatchString(text) {
		return 1
	}
	return defaultLimit
}

// canLead reports whether the card may be a commander under the rule
// (CR 903.3, 903.3a, 903.12c).
func canLead(c scryfall.Card, rule commanderRule) bool {
	_, text, power := frontFace(c)
	if strings.Contains(text, "can be your commander") {
		return true
	}
	types, subtypes := frontTypes(c)
	if !slices.Contains(types, "Legendary") {
		return false
	}
	switch {
	case slices.Contains(types, "Creature"), slices.Contains(subtypes, "Vehicle"):
		return true
	case slices.Contains(subtypes, "Spacecraft"):
		return power != nil // only Spacecraft with a power/toughness box
	}
	return rule == legendaryOrWalker && slices.Contains(types, "Planeswalker")
}

func isPlaneswalker(c scryfall.Card) bool {
	types, _ := frontTypes(c)
	return slices.Contains(types, "Planeswalker")
}

func isInstantOrSorcery(c scryfall.Card) bool {
	types, _ := frontTypes(c)
	return slices.Contains(types, "Instant") || slices.Contains(types, "Sorcery")
}

func isBackground(c scryfall.Card) bool {
	types, subtypes := frontTypes(c)
	return slices.Contains(types, "Legendary") && slices.Contains(types, "Enchantment") &&
		slices.Contains(subtypes, "Background")
}

func isDoctor(c scryfall.Card) bool {
	types, subtypes := frontTypes(c)
	return slices.Contains(types, "Creature") && slices.Contains(subtypes, "Time") &&
		slices.Contains(subtypes, "Lord") && slices.Contains(subtypes, "Doctor")
}

// leadsWithBackground reports whether chooser may be a commander with the
// Background bg as its second commander (CR 702.124k).
func leadsWithBackground(chooser, bg scryfall.Card, rule commanderRule) bool {
	return hasKeyword(chooser, "Choose a background") && canLead(chooser, rule) && isBackground(bg)
}

type partnerKind int

const (
	noPartner partnerKind = iota
	plainPartner
	partnerWith    // "Partner with <name>"
	partnerVariant // "Partner—Friends forever", "Partner—Survivors", ...
)

func partnerOf(c scryfall.Card) (partnerKind, string) {
	text := allText(c)
	if hasKeyword(c, "Partner with") {
		if m := partnerWithRe.FindStringSubmatch(text); m != nil {
			return partnerWith, strings.TrimSpace(m[1])
		}
	}
	if !hasKeyword(c, "Partner") {
		return noPartner, ""
	}
	if m := partnerVariantRe.FindStringSubmatch(text); m != nil {
		return partnerVariant, strings.TrimSpace(m[1])
	}
	return plainPartner, ""
}

// validPair reports whether two different cards may be commanders together
// through partner or Doctor's companion (CR 702.124). Backgrounds are handled
// by leadsWithBackground.
func validPair(a, b scryfall.Card) bool {
	ka, va := partnerOf(a)
	kb, vb := partnerOf(b)
	switch {
	case ka == partnerWith && kb == partnerWith:
		return va == b.Name && vb == a.Name
	case ka == partnerVariant && kb == partnerVariant:
		return va == vb
	case ka == plainPartner && kb == plainPartner:
		return true
	}
	return (hasKeyword(a, "Doctor's companion") && isDoctor(b)) ||
		(hasKeyword(b, "Doctor's companion") && isDoctor(a))
}
