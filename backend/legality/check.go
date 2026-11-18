// Package legality checks a deck against the deck-construction rules of Magic:
// The Gathering formats.
//
// Per-card legality (ban lists, card pools, Pauper's rarity rule, restricted
// lists) comes from the Scryfall legalities stored with each card. Check adds
// the construction rules Scryfall does not encode: deck and sideboard size,
// copy limits (with the Basic supertype and "A deck can have any number of
// cards named …" exemptions), and for commander formats the command zone,
// partner pairs and color identity.
//
// Tokens and emblems are ignored: they are reminders of what the deck creates,
// not cards in it.
//
// Sources: Comprehensive Rules 100.2a, 100.4a, 100.5, 702.124 and 903
// (effective 2026-11-06); magic.wizards.com/en/formats; oathbreakermtg.org/rules.
//
// Known limitations, which can make a legal deck show as not legal or the
// reverse:
//   - Rulebreaker commanders (e.g. Whtz, the Bibliophile) and commanders that
//     choose a color (e.g. Clara Oswald) are checked by the normal rules.
//   - Commanders that are creatures only outside the battlefield (e.g. Grist,
//     the Hunger Tide) are not recognised.
//   - Companion deckbuilding conditions are not checked; one companion may sit
//     in the sideboard of a commander format.
//   - Standard Brawl's rule that a colorless commander allows any number of one
//     basic land type is not implemented.
//   - Partner oathbreakers are not supported.
//   - Scryfall's "competitivebrawl" is not checked: go-scryfall v0.11.0 drops it
//     when cards are stored (models.FromScryfallCard).
package legality

import (
	"backend/models"
	"cmp"
	"fmt"
	"slices"
	"strings"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// maxNames caps the card names listed in one reason.
const maxNames = 10

// Result is a deck's verdict for one format.
type Result struct {
	Format  string
	Name    string
	Group   string
	Legal   bool
	Reasons []string
}

// Check evaluates the deck against every format, in display order. cards maps
// Oracle ID to card data; a deck item whose Oracle ID is missing has no local
// card data and makes the deck not legal anywhere. Maybe-board items are
// ignored.
func Check(items []models.DeckItem, cards map[string]scryfall.Card) []Result {
	d := newDeck(items, cards)
	results := make([]Result, len(formats))
	for i, f := range formats {
		reasons := d.check(f)
		results[i] = Result{
			Format:  f.key,
			Name:    f.name,
			Group:   f.group,
			Legal:   len(reasons) == 0,
			Reasons: reasons,
		}
	}
	return results
}

// zoneCounts maps Oracle ID to the number of copies in a zone, summed over
// printings and finishes.
type zoneCounts map[string]int

func (z zoneCounts) total() int {
	n := 0
	for _, q := range z {
		n += q
	}
	return n
}

type deck struct {
	cards               map[string]scryfall.Card
	main, side, command zoneCounts
	missing             int // distinct Oracle IDs without card data
}

func newDeck(items []models.DeckItem, cards map[string]scryfall.Card) deck {
	d := deck{cards: cards, main: zoneCounts{}, side: zoneCounts{}, command: zoneCounts{}}
	missing := make(map[string]bool)
	for _, item := range items {
		var z zoneCounts
		switch item.Zone {
		case models.ZoneMain:
			z = d.main
		case models.ZoneSide:
			z = d.side
		case models.ZoneCommand:
			z = d.command
		default:
			continue
		}
		c, ok := cards[item.OracleID]
		if ok && isToken(c) {
			continue
		}
		if !ok {
			missing[item.OracleID] = true
		}
		z[item.OracleID] += item.DesiredQuantity
	}
	d.missing = len(missing)
	return d
}

// reasons collects the violations found for one format.
type reasons []string

func (r *reasons) add(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}

func (d deck) check(f format) []string {
	r := reasons{}
	if d.missing > 0 {
		r.add("No card data yet: %s", cardCount(d.missing))
	}
	d.checkCardLegality(f, &r)
	switch f.commander {
	case noCommander:
		d.checkConstructed(f, &r)
	case oathbreaker:
		d.checkOathbreaker(f, &r)
	default:
		d.checkCommander(f, &r)
	}
	return r
}

// checkCardLegality reports cards Scryfall marks banned or not legal in the
// format. Restricted cards are legal here and limited by checkCopies.
func (d deck) checkCardLegality(f format, r *reasons) {
	var banned, notLegal []string
	for _, z := range []zoneCounts{d.main, d.side, d.command} {
		for oracleID := range z {
			c, ok := d.cards[oracleID]
			if !ok {
				continue
			}
			switch f.legality(c.Legalities) {
			case scryfall.LegalityLegal, scryfall.LegalityRestricted:
			case scryfall.LegalityBanned:
				banned = append(banned, c.Name)
			default: // not_legal, or no data for this format
				notLegal = append(notLegal, c.Name)
			}
		}
	}
	if len(banned) > 0 {
		r.add("Banned: %s", list(banned))
	}
	if len(notLegal) > 0 {
		r.add("Not legal: %s", list(notLegal))
	}
}

func (d deck) checkConstructed(f format, r *reasons) {
	if d.command.total() > 0 {
		r.add("%s has no command zone", f.name)
	}
	if n := d.main.total(); n < f.size {
		r.add("Deck has %s (needs at least %d)", cardCount(n), f.size)
	}
	if n := d.side.total(); n > maxSideboard {
		r.add("Sideboard has %s (max %d)", cardCount(n), maxSideboard)
	}
	d.checkCopies(f, r, constructedCopies, d.main, d.side)
}

func (d deck) checkCommander(f format, r *reasons) {
	identity, ok := d.checkCommandZone(f, r)
	d.checkExactSize(f, r)
	d.checkNoSideboard(r)
	d.checkCopies(f, r, 1, d.main, d.command)
	if ok {
		d.checkIdentity(r, "commander", identity)
	}
}

// checkCommandZone validates the commander(s) and returns their combined color
// identity. ok is false when the command zone is invalid or has cards without
// data, in which case color identity is not checked.
func (d deck) checkCommandZone(f format, r *reasons) (identity map[scryfall.Color]bool, ok bool) {
	leaders, known := d.expand(d.command)
	switch n := d.command.total(); {
	case n == 0:
		r.add("No commander")
		return nil, false
	case n > 2:
		r.add("Too many commanders: %d (max 2)", n)
		return nil, false
	case !known:
		return nil, false
	}

	if len(leaders) == 2 &&
		(leadsWithBackground(leaders[0], leaders[1], f.commander) ||
			leadsWithBackground(leaders[1], leaders[0], f.commander)) {
		return identityOf(leaders), true
	}

	var invalid []string
	for _, c := range leaders {
		if !canLead(c, f.commander) {
			invalid = append(invalid, c.Name)
		}
	}
	if len(invalid) > 0 {
		r.add("Not a valid commander: %s", list(invalid))
		return nil, false
	}
	if len(leaders) == 2 && (leaders[0].OracleID == leaders[1].OracleID || !validPair(leaders[0], leaders[1])) {
		r.add("Not a valid commander pair: %s + %s", leaders[0].Name, leaders[1].Name)
		return nil, false
	}
	return identityOf(leaders), true
}

func (d deck) checkOathbreaker(f format, r *reasons) {
	var identity map[scryfall.Color]bool
	leaders, known := d.expand(d.command)
	if known {
		if len(leaders) == 2 && isInstantOrSorcery(leaders[0]) && isPlaneswalker(leaders[1]) {
			leaders[0], leaders[1] = leaders[1], leaders[0]
		}
		if len(leaders) == 2 && isPlaneswalker(leaders[0]) && isInstantOrSorcery(leaders[1]) {
			identity = identityOf(leaders[:1])
		} else {
			r.add("Command zone needs one planeswalker and one instant or sorcery")
		}
	}
	d.checkExactSize(f, r)
	d.checkNoSideboard(r)
	d.checkCopies(f, r, 1, d.main, d.command)
	if identity != nil {
		// Covers the signature spell too, which sits in the command zone.
		d.checkIdentity(r, "oathbreaker", identity)
	}
}

// checkExactSize checks a commander-style deck, which counts the command zone
// but not a companion in the sideboard.
func (d deck) checkExactSize(f format, r *reasons) {
	if n := d.main.total() + d.command.total(); n != f.size {
		r.add("Deck has %s (needs exactly %d)", cardCount(n), f.size)
	}
}

// checkNoSideboard allows only a single companion outside a commander-style
// deck (CR 903.5f, 702.139).
func (d deck) checkNoSideboard(r *reasons) {
	if d.side.total() == 0 {
		return
	}
	if d.side.total() == 1 {
		for oracleID := range d.side {
			if c, ok := d.cards[oracleID]; !ok || hasKeyword(c, "Companion") {
				return
			}
		}
	}
	if names := d.names(d.side); len(names) > 0 {
		r.add("Sideboard not allowed (except one companion): %s", list(names))
	}
}

// checkCopies reports cards over their copy limit, counting the given zones
// together. Restricted cards are limited to one copy and reported separately.
func (d deck) checkCopies(f format, r *reasons, defaultLimit int, zones ...zoneCounts) {
	counts := make(map[string]int)
	for _, z := range zones {
		for oracleID, n := range z {
			counts[oracleID] += n
		}
	}

	var restricted []string
	over := make(map[int][]string)
	for oracleID, n := range counts {
		c, ok := d.cards[oracleID]
		if !ok {
			continue
		}
		if f.legality(c.Legalities) == scryfall.LegalityRestricted {
			if n > 1 {
				restricted = append(restricted, fmt.Sprintf("%s (%d)", c.Name, n))
			}
			continue
		}
		if limit := copyLimit(c, defaultLimit); n > limit {
			over[limit] = append(over[limit], fmt.Sprintf("%s (%d)", c.Name, n))
		}
	}

	if len(restricted) > 0 {
		r.add("Restricted (max 1): %s", list(restricted))
	}
	limits := make([]int, 0, len(over))
	for limit := range over {
		limits = append(limits, limit)
	}
	slices.Sort(limits)
	for _, limit := range limits {
		copies := "copies"
		if limit == 1 {
			copies = "copy"
		}
		r.add("More than %d %s: %s", limit, copies, list(over[limit]))
	}
}

// checkIdentity reports cards whose color identity is not within the leader's
// (CR 903.5c).
func (d deck) checkIdentity(r *reasons, leader string, identity map[scryfall.Color]bool) {
	var outside []string
	for _, z := range []zoneCounts{d.main, d.side, d.command} {
		for oracleID := range z {
			c, ok := d.cards[oracleID]
			if !ok {
				continue
			}
			if slices.ContainsFunc(c.ColorIdentity, func(col scryfall.Color) bool { return !identity[col] }) {
				outside = append(outside, c.Name)
			}
		}
	}
	if len(outside) > 0 {
		r.add("Outside the %s's color identity: %s", leader, list(outside))
	}
}

// expand returns the zone's cards, one entry per copy, sorted by name. known
// is false when any of them has no card data.
func (d deck) expand(z zoneCounts) (cards []scryfall.Card, known bool) {
	known = true
	for oracleID, n := range z {
		c, ok := d.cards[oracleID]
		if !ok {
			known = false
			continue
		}
		for range n {
			cards = append(cards, c)
		}
	}
	slices.SortFunc(cards, func(a, b scryfall.Card) int { return cmp.Compare(a.Name, b.Name) })
	return cards, known
}

// names returns the names of the zone's cards that have card data.
func (d deck) names(z zoneCounts) []string {
	var names []string
	for oracleID := range z {
		if c, ok := d.cards[oracleID]; ok {
			names = append(names, c.Name)
		}
	}
	return names
}

func identityOf(cards []scryfall.Card) map[scryfall.Color]bool {
	identity := make(map[scryfall.Color]bool)
	for _, c := range cards {
		for _, col := range c.ColorIdentity {
			identity[col] = true
		}
	}
	return identity
}

// list joins names sorted and de-duplicated, capped at maxNames.
func list(names []string) string {
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) > maxNames {
		return fmt.Sprintf("%s and %d more", strings.Join(names[:maxNames], ", "), len(names)-maxNames)
	}
	return strings.Join(names, ", ")
}

func cardCount(n int) string {
	if n == 1 {
		return "1 card"
	}
	return fmt.Sprintf("%d cards", n)
}
