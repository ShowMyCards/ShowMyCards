package legality_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"backend/legality"
	"backend/models"

	scryfall "github.com/BlueMonday/go-scryfall"
)

// --- card builders ---------------------------------------------------------

type option func(*scryfall.Card)

// card returns a colorless instant that is legal in every format.
func card(name string, opts ...option) scryfall.Card {
	var l scryfall.Legalities
	v := reflect.ValueOf(&l).Elem()
	for i := range v.NumField() {
		v.Field(i).SetString(string(scryfall.LegalityLegal))
	}
	c := scryfall.Card{OracleID: "oracle-" + name, Name: name, TypeLine: "Instant", Legalities: l}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func typeLine(t string) option    { return func(c *scryfall.Card) { c.TypeLine = t } }
func printedName(n string) option { return func(c *scryfall.Card) { c.PrintedName = &n } }
func text(t string) option        { return func(c *scryfall.Card) { c.OracleText = t } }
func keywords(k ...string) option { return func(c *scryfall.Card) { c.Keywords = k } }
func power(p string) option       { return func(c *scryfall.Card) { c.Power = &p } }
func layout(l scryfall.Layout) option {
	return func(c *scryfall.Card) { c.Layout = l }
}
func colors(cs ...scryfall.Color) option {
	return func(c *scryfall.Card) { c.ColorIdentity = cs }
}
func legal(set func(*scryfall.Legalities)) option {
	return func(c *scryfall.Card) { set(&c.Legalities) }
}

func faces(front, back scryfall.CardFace) option {
	return func(c *scryfall.Card) {
		c.TypeLine = front.TypeLine + " // " + back.TypeLine
		c.CardFaces = []scryfall.CardFace{front, back}
	}
}

func legend(name string, cs ...scryfall.Color) scryfall.Card {
	return card(name, typeLine("Legendary Creature — Human"), colors(cs...))
}

const (
	W = scryfall.ColorWhite
	U = scryfall.ColorBlue
	B = scryfall.ColorBlack
	R = scryfall.ColorRed
	G = scryfall.ColorGreen
)

// --- deck builder ----------------------------------------------------------

type deck struct {
	cards   map[string]scryfall.Card
	items   []models.DeckItem
	fillers int
}

func newDeck() *deck { return &deck{cards: map[string]scryfall.Card{}} }

func (d *deck) add(c scryfall.Card, zone models.DeckZone, qty int) *deck {
	d.cards[c.OracleID] = c
	d.items = append(d.items, models.DeckItem{OracleID: c.OracleID, Zone: zone, DesiredQuantity: qty})
	return d
}

// fill adds n distinct single copies, so they never trip a copy limit.
func (d *deck) fill(zone models.DeckZone, n int, opts ...option) *deck {
	for range n {
		d.fillers++
		d.add(card(fmt.Sprintf("Filler %03d", d.fillers), opts...), zone, 1)
	}
	return d
}

func (d *deck) check() map[string]legality.Result {
	results := make(map[string]legality.Result)
	for _, r := range legality.Check(d.items, d.cards) {
		results[r.Format] = r
	}
	return results
}

// expect asserts a format's exact reasons; no reasons means legal.
func expect(t *testing.T, results map[string]legality.Result, format string, want ...string) {
	t.Helper()
	got, ok := results[format]
	if !ok {
		t.Fatalf("no result for format %q", format)
	}
	if want == nil {
		want = []string{}
	}
	if got.Reasons == nil {
		t.Errorf("%s: reasons is nil, want non-nil slice", format)
	}
	if !slices.Equal(got.Reasons, want) {
		t.Errorf("%s reasons:\n got  %q\n want %q", format, got.Reasons, want)
	}
	if got.Legal != (len(want) == 0) {
		t.Errorf("%s: legal = %v with reasons %q", format, got.Legal, got.Reasons)
	}
}

// --- tests -----------------------------------------------------------------

func TestCheck_FormatsInOrder(t *testing.T) {
	results := legality.Check(nil, nil)
	want := []struct{ key, name string }{
		{"standard", "Standard"},
		{"pioneer", "Pioneer"},
		{"modern", "Modern"},
		{"legacy", "Legacy"},
		{"vintage", "Vintage"},
		{"pauper", "Pauper"},
		{"future", "Future Standard"},
		{"commander", "Commander"},
		{"oathbreaker", "Oathbreaker"},
	}
	if len(results) != len(want) {
		t.Fatalf("got %d formats, want %d", len(results), len(want))
	}
	for i, w := range want {
		r := results[i]
		if r.Format != w.key || r.Name != w.name {
			t.Errorf("format %d = %s/%s, want %s/%s", i, r.Format, r.Name, w.key, w.name)
		}
	}
}

func TestCheck_Constructed(t *testing.T) {
	t.Run("legal 60-card deck", func(t *testing.T) {
		results := newDeck().fill(models.ZoneMain, 60).check()
		for _, f := range []string{"standard", "pioneer", "modern", "legacy", "vintage", "pauper", "future"} {
			expect(t, results, f)
		}
		expect(t, results, "commander", "No commander", "Deck has 60 cards (needs exactly 100)")
	})

	t.Run("banned and not legal cards, sorted", func(t *testing.T) {
		results := newDeck().
			fill(models.ZoneMain, 58).
			add(card("Zeta", legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityNotLegal })), models.ZoneMain, 1).
			add(card("Alpha", legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityNotLegal })), models.ZoneSide, 1).
			add(card("Black Lotus", legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityBanned })), models.ZoneMain, 1).
			check()
		expect(t, results, "modern", "Banned: Black Lotus", "Not legal: Alpha, Zeta")
		expect(t, results, "legacy")
	})

	t.Run("missing legality value counts as not legal", func(t *testing.T) {
		results := newDeck().
			fill(models.ZoneMain, 59).
			add(card("Stale", legal(func(l *scryfall.Legalities) { l.Pioneer = "" })), models.ZoneMain, 1).
			check()
		expect(t, results, "pioneer", "Not legal: Stale")
	})

	t.Run("restricted card limited to one copy across main and side", func(t *testing.T) {
		lotus := card("Black Lotus", legal(func(l *scryfall.Legalities) {
			l.Vintage = scryfall.LegalityRestricted
			l.Legacy = scryfall.LegalityBanned
		}))
		results := newDeck().
			fill(models.ZoneMain, 57).
			add(lotus, models.ZoneMain, 3).
			add(lotus, models.ZoneSide, 1).
			check()
		expect(t, results, "vintage", "Restricted (max 1): Black Lotus (4)")
		expect(t, results, "legacy", "Banned: Black Lotus")
	})

	t.Run("one restricted copy is legal", func(t *testing.T) {
		lotus := card("Black Lotus", legal(func(l *scryfall.Legalities) { l.Vintage = scryfall.LegalityRestricted }))
		results := newDeck().fill(models.ZoneMain, 59).add(lotus, models.ZoneMain, 1).check()
		expect(t, results, "vintage")
	})

	t.Run("too few cards; maybe board does not count", func(t *testing.T) {
		results := newDeck().fill(models.ZoneMain, 59).fill(models.ZoneMaybe, 5).check()
		expect(t, results, "modern", "Deck has 59 cards (needs at least 60)")
	})

	t.Run("sideboard over 15", func(t *testing.T) {
		results := newDeck().fill(models.ZoneMain, 60).fill(models.ZoneSide, 16).check()
		expect(t, results, "modern", "Sideboard has 16 cards (max 15)")
	})

	t.Run("command zone not allowed", func(t *testing.T) {
		results := newDeck().fill(models.ZoneMain, 60).add(legend("Leader"), models.ZoneCommand, 1).check()
		expect(t, results, "modern", "Modern has no command zone")
	})

	t.Run("copies counted across zones and printings", func(t *testing.T) {
		bolt := card("Lightning Bolt")
		d := newDeck().fill(models.ZoneMain, 56).add(bolt, models.ZoneMain, 3).add(bolt, models.ZoneSide, 1)
		// Same Oracle, different pinned printing: a separate row that still counts.
		d.items = append(d.items, models.DeckItem{OracleID: bolt.OracleID, ScryfallID: "other-printing", Zone: models.ZoneMain, DesiredQuantity: 1})
		expect(t, d.check(), "modern", "More than 4 copies: Lightning Bolt (5)")
	})

	t.Run("basic lands are exempt", func(t *testing.T) {
		results := newDeck().
			add(card("Island", typeLine("Basic Land — Island"), colors()), models.ZoneMain, 20).
			add(card("Snow-Covered Island", typeLine("Basic Snow Land — Island")), models.ZoneMain, 20).
			add(card("Wastes", typeLine("Basic Land")), models.ZoneMain, 20).
			check()
		expect(t, results, "modern")
	})

	t.Run("rules-text copy limits", func(t *testing.T) {
		results := newDeck().
			add(card("Relentless Rats", text("A deck can have any number of cards named Relentless Rats.")), models.ZoneMain, 30).
			add(card("Nazgûl", text("A deck can have up to nine cards named Nazgûl.")), models.ZoneMain, 9).
			add(card("Seven Dwarves", text("A deck can have up to seven cards named Seven Dwarves.")), models.ZoneMain, 8).
			add(card("Vazal", text("Megalegendary (Your deck can have only one copy of this card.)")), models.ZoneMain, 2).
			add(card("Opt"), models.ZoneMain, 11).
			check()
		expect(t, results, "modern",
			"More than 1 copy: Vazal (2)",
			"More than 4 copies: Opt (11)",
			"More than 7 copies: Seven Dwarves (8)",
		)
	})

	t.Run("missing card data fails every format but counts toward size", func(t *testing.T) {
		d := newDeck().fill(models.ZoneMain, 59)
		d.items = append(d.items, models.DeckItem{OracleID: "unknown", Zone: models.ZoneMain, DesiredQuantity: 1})
		results := d.check()
		expect(t, results, "modern", "No card data yet: 1 card")
		for _, r := range results {
			if r.Legal || r.Reasons[0] != "No card data yet: 1 card" {
				t.Errorf("%s: want not legal with missing-data reason first, got %q", r.Format, r.Reasons)
			}
		}
	})

	t.Run("long name lists are capped", func(t *testing.T) {
		d := newDeck().fill(models.ZoneMain, 48)
		for i := range 12 {
			d.add(card(fmt.Sprintf("Banned %02d", i+1), legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityBanned })), models.ZoneMain, 1)
		}
		expect(t, d.check(), "modern",
			"Banned: Banned 01, Banned 02, Banned 03, Banned 04, Banned 05, Banned 06, Banned 07, Banned 08, Banned 09, Banned 10 and 2 more")
	})
}

func TestCheck_Commander(t *testing.T) {
	commanderDeck := func(leaders ...scryfall.Card) *deck {
		d := newDeck()
		for _, l := range leaders {
			d.add(l, models.ZoneCommand, 1)
		}
		return d.fill(models.ZoneMain, 100-len(leaders))
	}

	t.Run("legal 99 plus commander", func(t *testing.T) {
		results := commanderDeck(legend("Leader", G)).check()
		expect(t, results, "commander")
		expect(t, results, "oathbreaker",
			"Command zone needs one planeswalker and one instant or sorcery", "Deck has 100 cards (needs exactly 60)")
	})

	t.Run("no commander", func(t *testing.T) {
		expect(t, newDeck().fill(models.ZoneMain, 100).check(), "commander", "No commander")
	})

	t.Run("too many commanders", func(t *testing.T) {
		results := commanderDeck(legend("A"), legend("B"), legend("C")).check()
		expect(t, results, "commander", "Too many commanders: 3 (max 2)")
	})

	t.Run("wrong size and duplicates", func(t *testing.T) {
		sol := card("Sol Ring")
		results := newDeck().
			add(legend("Leader"), models.ZoneCommand, 1).
			add(sol, models.ZoneMain, 2).
			fill(models.ZoneMain, 80).
			check()
		expect(t, results, "commander", "Deck has 83 cards (needs exactly 100)", "More than 1 copy: Sol Ring (2)")
	})

	t.Run("commander also in the main deck", func(t *testing.T) {
		leader := legend("Leader")
		results := commanderDeck(leader).add(leader, models.ZoneMain, 1).check()
		expect(t, results, "commander", "Deck has 101 cards (needs exactly 100)", "More than 1 copy: Leader (2)")
	})

	t.Run("eligibility", func(t *testing.T) {
		tests := []struct {
			name   string
			leader scryfall.Card
			want   []string
		}{
			{"non-legendary creature", card("Grizzly Bears", typeLine("Creature — Bear")),
				[]string{"Not a valid commander: Grizzly Bears"}},
			{"plain planeswalker cannot lead", card("Jace", typeLine("Legendary Planeswalker — Jace")),
				[]string{"Not a valid commander: Jace"}},
			{"can be your commander", card("Daretti", typeLine("Legendary Planeswalker — Daretti"),
				text("Daretti can be your commander.")), nil},
			{"legendary vehicle", card("Parhelion II", typeLine("Legendary Artifact — Vehicle")), nil},
			{"legendary spacecraft with power", card("Ship", typeLine("Legendary Artifact — Spacecraft"), power("5")), nil},
			{"legendary spacecraft without power", card("Station", typeLine("Legendary Artifact — Spacecraft")),
				[]string{"Not a valid commander: Station"}},
			{"double-faced card with legendary front", card("Front // Back", faces(
				scryfall.CardFace{TypeLine: "Legendary Creature — Human"},
				scryfall.CardFace{TypeLine: "Legendary Planeswalker — Human"})), nil},
			{"double-faced card with legendary back only", card("Plain // Legend", faces(
				scryfall.CardFace{TypeLine: "Creature — Human"},
				scryfall.CardFace{TypeLine: "Legendary Creature — Human"})),
				[]string{"Not a valid commander: Plain // Legend"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expect(t, commanderDeck(tt.leader).check(), "commander", tt.want...)
			})
		}
	})
}

func TestCheck_CommanderPairs(t *testing.T) {
	partner := func(name string, cs ...scryfall.Color) scryfall.Card {
		return card(name, typeLine("Legendary Creature — Human"), keywords("Partner"),
			text("Partner (You can have two commanders if both have partner.)"), colors(cs...))
	}
	partnerWith := func(name, other string) scryfall.Card {
		return card(name, typeLine("Legendary Creature — Human"), keywords("Partner with", "Partner"),
			text("Partner with "+other+" (When this creature enters, target player may put "+other+" into their hand.)"))
	}
	variant := func(name, v string) scryfall.Card {
		return card(name, typeLine("Legendary Creature — Human"), keywords("Partner"),
			text("Partner—"+v+" (You can have two commanders if both have this ability.)"))
	}
	background := card("Raised by Giants", typeLine("Legendary Enchantment — Background"), colors(G))
	chooser := card("Wilson", typeLine("Legendary Creature — Bear Warrior"), keywords("Choose a background"), colors(G))
	companion := card("Clara Oswald", typeLine("Legendary Creature — Human Advisor"), keywords("Doctor's companion"))
	doctor := card("The Tenth Doctor", typeLine("Legendary Creature — Time Lord Doctor"), colors(R))

	pair := func(a, b scryfall.Card) map[string]legality.Result {
		return newDeck().add(a, models.ZoneCommand, 1).add(b, models.ZoneCommand, 1).fill(models.ZoneMain, 98).check()
	}

	valid := []struct {
		name string
		a, b scryfall.Card
	}{
		{"plain partners", partner("Thrasios"), partner("Tymna")},
		{"partner with each other", partnerWith("Pir, Imaginative Rascal", "Toothy, Imaginary Friend"),
			partnerWith("Toothy, Imaginary Friend", "Pir, Imaginative Rascal")},
		{"friends forever", variant("Hargilde", "Friends forever"), variant("Cecily", "Friends forever")},
		{"choose a background", chooser, background},
		{"background listed first", background, chooser},
		{"doctor's companion", companion, doctor},
	}
	for _, tt := range valid {
		t.Run("valid "+tt.name, func(t *testing.T) {
			expect(t, pair(tt.a, tt.b), "commander")
		})
	}

	invalid := []struct {
		name string
		a, b scryfall.Card
		want []string
	}{
		{"partner with someone else", partnerWith("Pir, Imaginative Rascal", "Toothy, Imaginary Friend"), partner("Thrasios"),
			[]string{"Not a valid commander pair: Pir, Imaginative Rascal + Thrasios"}},
		{"different partner variants", variant("Hargilde", "Friends forever"), variant("Leon", "Survivors"),
			[]string{"Not a valid commander pair: Hargilde + Leon"}},
		{"two legends without partner", legend("A"), legend("B"),
			[]string{"Not a valid commander pair: A + B"}},
		{"background without a chooser", legend("A"), background,
			[]string{"Not a valid commander: Raised by Giants"}},
	}
	for _, tt := range invalid {
		t.Run("invalid "+tt.name, func(t *testing.T) {
			expect(t, pair(tt.a, tt.b), "commander", tt.want...)
		})
	}

	t.Run("two copies of one partner", func(t *testing.T) {
		thrasios := partner("Thrasios")
		results := newDeck().add(thrasios, models.ZoneCommand, 2).fill(models.ZoneMain, 98).check()
		expect(t, results, "commander",
			"Not a valid commander pair: Thrasios + Thrasios", "More than 1 copy: Thrasios (2)")
	})

	t.Run("partners combine color identity", func(t *testing.T) {
		results := newDeck().
			add(partner("Thrasios", G, U), models.ZoneCommand, 1).
			add(partner("Tymna", W, B), models.ZoneCommand, 1).
			add(card("Four Color", colors(W, U, B, G)), models.ZoneMain, 1).
			add(card("Red Card", colors(R)), models.ZoneMain, 1).
			fill(models.ZoneMain, 96).
			check()
		expect(t, results, "commander", "Outside the commander's color identity: Red Card")
	})
}

func TestCheck_CommanderSideboardAndIdentity(t *testing.T) {
	lurrus := card("Lurrus", typeLine("Legendary Creature — Cat Nightmare"), keywords("Lifelink", "Companion"), colors(W, B))
	base := func() *deck {
		return newDeck().add(legend("Leader", W, B), models.ZoneCommand, 1).fill(models.ZoneMain, 99)
	}

	t.Run("one companion is allowed and not counted", func(t *testing.T) {
		expect(t, base().add(lurrus, models.ZoneSide, 1).check(), "commander")
	})

	t.Run("other sideboard cards are not allowed", func(t *testing.T) {
		results := base().add(lurrus, models.ZoneSide, 1).add(card("Extra"), models.ZoneSide, 1).check()
		expect(t, results, "commander", "Sideboard not allowed (except one companion): Extra, Lurrus")
	})

	t.Run("a single non-companion is not allowed", func(t *testing.T) {
		results := base().add(card("Extra"), models.ZoneSide, 1).check()
		expect(t, results, "commander", "Sideboard not allowed (except one companion): Extra")
	})

	t.Run("cards outside the color identity", func(t *testing.T) {
		results := newDeck().
			add(legend("Leader", G), models.ZoneCommand, 1).
			add(lurrus, models.ZoneSide, 1).
			add(card("Bolt", colors(R)), models.ZoneMain, 1).
			add(card("Plains", typeLine("Basic Land — Plains"), colors(W)), models.ZoneMain, 1).
			fill(models.ZoneMain, 97).
			check()
		expect(t, results, "commander", "Outside the commander's color identity: Bolt, Lurrus, Plains")
	})

	t.Run("color identity is skipped when the commander is invalid", func(t *testing.T) {
		results := newDeck().
			add(card("Grizzly Bears", typeLine("Creature — Bear"), colors(G)), models.ZoneCommand, 1).
			add(card("Bolt", colors(R)), models.ZoneMain, 1).
			fill(models.ZoneMain, 98).
			check()
		expect(t, results, "commander", "Not a valid commander: Grizzly Bears")
	})
}

func TestCheck_Oathbreaker(t *testing.T) {
	walker := card("Jace", typeLine("Legendary Planeswalker — Jace"), colors(U))
	spell := card("Brainstorm", typeLine("Instant"), colors(U))

	t.Run("legal deck", func(t *testing.T) {
		results := newDeck().
			add(spell, models.ZoneCommand, 1).
			add(walker, models.ZoneCommand, 1).
			fill(models.ZoneMain, 58).
			check()
		expect(t, results, "oathbreaker")
	})

	t.Run("missing signature spell", func(t *testing.T) {
		results := newDeck().add(walker, models.ZoneCommand, 1).fill(models.ZoneMain, 58).check()
		expect(t, results, "oathbreaker",
			"Command zone needs one planeswalker and one instant or sorcery", "Deck has 59 cards (needs exactly 60)")
	})

	t.Run("signature spell outside the oathbreaker's identity", func(t *testing.T) {
		results := newDeck().
			add(walker, models.ZoneCommand, 1).
			add(card("Lightning Bolt", colors(R)), models.ZoneCommand, 1).
			fill(models.ZoneMain, 58).
			check()
		expect(t, results, "oathbreaker", "Outside the oathbreaker's color identity: Lightning Bolt")
	})
}

func TestCheck_IgnoresNonDeckCards(t *testing.T) {
	notLegalAnywhere := legal(func(l *scryfall.Legalities) { *l = scryfall.Legalities{} })
	fish := card("Fish", layout(scryfall.LayoutToken), typeLine("Token Creature — Fish"), colors(U), notLegalAnywhere)
	role := card("Monster Role", layout(scryfall.LayoutFlip), typeLine("Token Enchantment — Aura Role"), notLegalAnywhere)
	emblem := card("Emblem", layout(scryfall.LayoutEmblem), typeLine("Emblem — Jace"), notLegalAnywhere)
	poison := card("Poison Counter", layout(scryfall.LayoutNormal), typeLine("Card"), notLegalAnywhere)
	artCard := card("Sol Ring // Sol Ring", layout(scryfall.LayoutArtSeries), typeLine("Card // Card"), notLegalAnywhere)
	themeCard := card("Dragons", layout(scryfall.Layout("front_card")), typeLine("Card"), notLegalAnywhere)

	t.Run("constructed", func(t *testing.T) {
		results := newDeck().
			fill(models.ZoneMain, 60).
			add(fish, models.ZoneMain, 5).
			add(role, models.ZoneSide, 16).
			check()
		expect(t, results, "modern")
	})

	t.Run("commander", func(t *testing.T) {
		results := newDeck().
			add(legend("Leader", G), models.ZoneCommand, 1).
			fill(models.ZoneMain, 99).
			add(fish, models.ZoneMain, 3).
			add(poison, models.ZoneMain, 1).
			add(artCard, models.ZoneMain, 1).
			add(themeCard, models.ZoneMain, 1).
			add(emblem, models.ZoneSide, 1).
			check()
		expect(t, results, "commander")
	})
}

func TestCheck_LocalizedCardNames(t *testing.T) {
	t.Run("uses printed_name in constructed violations and falls back to english", func(t *testing.T) {
		lotus := card("Black Lotus", printedName("Schwarzer Lotus"), legal(func(l *scryfall.Legalities) {
			l.Modern = scryfall.LegalityBanned
			l.Vintage = scryfall.LegalityRestricted
		}))
		bolt := card("Lightning Bolt", printedName("Blitzschlag"))
		unlocalized := card("Zeta", legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityNotLegal }))
		localizedNotLegal := card("Sol Ring", printedName("Aether-Ring"), legal(func(l *scryfall.Legalities) { l.Modern = scryfall.LegalityNotLegal }))

		results := newDeck().
			fill(models.ZoneMain, 52).
			add(lotus, models.ZoneMain, 2).
			add(bolt, models.ZoneMain, 5).
			add(unlocalized, models.ZoneMain, 1).
			add(localizedNotLegal, models.ZoneMain, 1).
			check()

		expect(t, results, "modern",
			"Banned: Schwarzer Lotus",
			"Not legal: Aether-Ring, Zeta",
			"More than 4 copies: Blitzschlag (5)",
		)
		expect(t, results, "vintage",
			"Restricted (max 1): Schwarzer Lotus (2)",
			"More than 4 copies: Blitzschlag (5)",
		)
	})

	t.Run("uses printed_name in commander violations", func(t *testing.T) {
		bear := card("Grizzly Bears", printedName("Grizzlybären"), typeLine("Creature — Bear"), colors(G))
		expect(t, newDeck().add(bear, models.ZoneCommand, 1).fill(models.ZoneMain, 99).check(),
			"commander", "Not a valid commander: Grizzlybären")

		leaderA := card("Leader A", printedName("Anführer Eins"), typeLine("Legendary Creature — Human"), colors(G))
		leaderB := card("Leader B", printedName("Anführer Zwei"), typeLine("Legendary Creature — Human"), colors(G))
		expect(t, newDeck().add(leaderA, models.ZoneCommand, 1).add(leaderB, models.ZoneCommand, 1).fill(models.ZoneMain, 98).check(),
			"commander", "Not a valid commander pair: Anführer Eins + Anführer Zwei")

		redCard := card("Lightning Bolt", printedName("Blitzschlag"), colors(R))
		sideCard := card("Negate", printedName("Negieren"), colors(G))
		expect(t, newDeck().
			add(legend("Leader", G), models.ZoneCommand, 1).
			add(redCard, models.ZoneMain, 1).
			add(sideCard, models.ZoneSide, 1).
			fill(models.ZoneMain, 98).
			check(),
			"commander",
			"Sideboard not allowed (except one companion): Negieren",
			"Outside the commander's color identity: Blitzschlag",
		)
	})

	t.Run("partner-with matching still uses canonical english name", func(t *testing.T) {
		pir := card("Pir, Imaginative Rascal",
			printedName("Pir, fantasievoller Schlingel"),
			typeLine("Legendary Creature — Human"),
			keywords("Partner with", "Partner"),
			text("Partner with Toothy, Imaginary Friend (When this creature enters, target player may put Toothy, Imaginary Friend into their hand.)"),
		)
		toothy := card("Toothy, Imaginary Friend",
			printedName("Zähnchen, imaginärer Freund"),
			typeLine("Legendary Creature — Illusion"),
			keywords("Partner with", "Partner"),
			text("Partner with Pir, Imaginative Rascal (When this creature enters, target player may put Pir, Imaginative Rascal into their hand.)"),
		)
		results := newDeck().add(pir, models.ZoneCommand, 1).add(toothy, models.ZoneCommand, 1).fill(models.ZoneMain, 98).check()
		expect(t, results, "commander")
	})
}
