package legality

import scryfall "github.com/BlueMonday/go-scryfall"

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
	// oathbreaker is exactly `size` singleton cards led by a planeswalker plus
	// an instant or sorcery signature spell (oathbreakermtg.org/rules 906).
	oathbreaker
)

// format is one row of the format table.
type format struct {
	key       string // Scryfall legalities key
	name      string // display name
	legality  func(scryfall.Legalities) scryfall.Legality
	size      int // minimum (noCommander) or exact (all others) deck size
	commander commanderRule
}

// formats lists every checked format in display order.
//
// Only paper formats with an official rules source are listed. The MTG Arena
// formats (Alchemy, Historic, Timeless, Brawl, Standard Brawl) are left out on
// purpose: the app tracks physical cards, which cannot be played on Arena.
var formats = []format{
	{"standard", "Standard", func(l scryfall.Legalities) scryfall.Legality { return l.Standard }, 60, noCommander},
	{"pioneer", "Pioneer", func(l scryfall.Legalities) scryfall.Legality { return l.Pioneer }, 60, noCommander},
	{"modern", "Modern", func(l scryfall.Legalities) scryfall.Legality { return l.Modern }, 60, noCommander},
	{"legacy", "Legacy", func(l scryfall.Legalities) scryfall.Legality { return l.Legacy }, 60, noCommander},
	{"vintage", "Vintage", func(l scryfall.Legalities) scryfall.Legality { return l.Vintage }, 60, noCommander},
	{"pauper", "Pauper", func(l scryfall.Legalities) scryfall.Legality { return l.Pauper }, 60, noCommander},
	{"future", "Future Standard", func(l scryfall.Legalities) scryfall.Legality { return l.Future }, 60, noCommander},
	{"commander", "Commander", func(l scryfall.Legalities) scryfall.Legality { return l.Commander }, 100, legendary},
	{"oathbreaker", "Oathbreaker", func(l scryfall.Legalities) scryfall.Legality { return l.Oathbreaker }, 60, oathbreaker},
}

const (
	// maxSideboard is the constructed sideboard limit (CR 100.4a).
	maxSideboard = 15
	// constructedCopies is the constructed copy limit (CR 100.2a).
	constructedCopies = 4
)
