package legality

import scryfall "github.com/BlueMonday/go-scryfall"

// Display groups, in the order the frontend shows them.
const (
	groupConstructed = "Constructed"
	groupCommander   = "Commander"
	groupArena       = "Arena"
)

// commanderRule says whether a format has a command zone and what may lead it.
type commanderRule int

const (
	// noCommander formats (constructed) require at least `size` main-deck
	// cards, allow a 15-card sideboard and 4 copies of a card, and have no
	// command zone (CR 100.2a, 100.4a, 100.5).
	noCommander commanderRule = iota
	// legendary formats (Commander) are exactly `size` singleton cards led by
	// a legendary creature, Vehicle or Spacecraft (CR 903.3, 903.5).
	legendary
	// legendaryOrWalker formats (Brawl, Standard Brawl) also allow a
	// planeswalker as commander (CR 903.12c).
	legendaryOrWalker
	// oathbreaker is exactly `size` singleton cards led by a planeswalker plus
	// an instant or sorcery signature spell (oathbreakermtg.org/rules 906).
	oathbreaker
)

// format is one row of the format table.
type format struct {
	key       string // Scryfall legalities key
	name      string // display name
	group     string
	legality  func(scryfall.Legalities) scryfall.Legality
	size      int // minimum (noCommander) or exact (all others) deck size
	commander commanderRule
}

// formats lists every checked format in display order.
//
// Only formats with an official rules source are listed. Dropped formats can
// be added back as rows: Premodern (PreModern), Penny Dreadful (Penny) and Old
// School (OldSchool) as noCommander rows of size 60; Duel Commander (Duel) and
// PreDH (PreDH) as legendary rows of size 100; Tiny Leaders
// (TinyLeadersReborn) as a legendary row of size 50. Pauper Commander and
// Gladiator need rules of their own (uncommon commander; 100-card singleton
// without a commander).
var formats = []format{
	{"standard", "Standard", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Standard }, 60, noCommander},
	{"pioneer", "Pioneer", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Pioneer }, 60, noCommander},
	{"modern", "Modern", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Modern }, 60, noCommander},
	{"legacy", "Legacy", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Legacy }, 60, noCommander},
	{"vintage", "Vintage", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Vintage }, 60, noCommander},
	{"pauper", "Pauper", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Pauper }, 60, noCommander},
	{"future", "Future Standard", groupConstructed, func(l scryfall.Legalities) scryfall.Legality { return l.Future }, 60, noCommander},
	{"commander", "Commander", groupCommander, func(l scryfall.Legalities) scryfall.Legality { return l.Commander }, 100, legendary},
	{"oathbreaker", "Oathbreaker", groupCommander, func(l scryfall.Legalities) scryfall.Legality { return l.Oathbreaker }, 60, oathbreaker},
	{"alchemy", "Alchemy", groupArena, func(l scryfall.Legalities) scryfall.Legality { return l.Alchemy }, 60, noCommander},
	{"historic", "Historic", groupArena, func(l scryfall.Legalities) scryfall.Legality { return l.Historic }, 60, noCommander},
	{"timeless", "Timeless", groupArena, func(l scryfall.Legalities) scryfall.Legality { return l.Timeless }, 60, noCommander},
	{"brawl", "Brawl", groupArena, func(l scryfall.Legalities) scryfall.Legality { return l.Brawl }, 100, legendaryOrWalker},
	{"standardbrawl", "Standard Brawl", groupArena, func(l scryfall.Legalities) scryfall.Legality { return l.StandardBrawl }, 60, legendaryOrWalker},
}

const (
	// maxSideboard is the constructed sideboard limit (CR 100.4a).
	maxSideboard = 15
	// constructedCopies is the constructed copy limit (CR 100.2a).
	constructedCopies = 4
)
