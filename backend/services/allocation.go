package services

import (
	"backend/models"
	"cmp"
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"
)

// AllocationService is the single source of truth for "decked" availability.
//
// It matches deck demand (deck_items in demand zones) against owned inventory
// and reports, per Oracle ID, how many copies are owned, demanded, free, and
// whether the Oracle is over-committed across all decks. It is deliberately
// general: the same map feeds both the deck-detail enrichment (this issue) and
// the inventory "available for deck building" filter (a later issue).
//
// Allocation is global across every deck and surfaces conflicts without picking
// a winner — no deck has priority. See FR98/IMPLEMENTATION_PLAN.md §2.
//
// Treatment scope (Milestone 1): when a deck item pins a finish, treatment is
// NOT factored into the global over-commitment math here — the math runs purely
// on Oracle-ID and pinned-Scryfall-ID quantities. Finish-aware allocation is a
// documented Milestone 2 follow-up (see IMPLEMENTATION_PLAN.md §2 "Treatment —
// v1 scope"). Treatment is only consulted at the deck-detail enrichment level.
type AllocationService struct {
	db *gorm.DB
}

// NewAllocationService creates a new allocation service.
func NewAllocationService(db *gorm.DB) *AllocationService {
	return &AllocationService{db: db}
}

// OracleAvailability is the per-Oracle availability computed from inventory and
// all decks' demand-zone items.
type OracleAvailability struct {
	OracleID      string `json:"oracle_id"`
	Owned         int    `json:"owned"`
	Demand        int    `json:"demand"`
	Free          int    `json:"free"`
	OverCommitted bool   `json:"over_committed"`
}

// AvailabilityMap holds the full per-Oracle availability plus the per-printing
// quantities needed for the pinned-printing over-commitment check. It is the
// reusable result both deck-detail enrichment and the inventory filter consume.
type AvailabilityMap struct {
	// Oracle maps oracle_id -> availability.
	Oracle map[string]OracleAvailability
	// ownedByScryfall maps scryfall_id -> owned copies of that printing.
	ownedByScryfall map[string]int
	// pinnedByScryfall maps scryfall_id -> demand for that specific printing.
	pinnedByScryfall map[string]int
	// demandRows maps oracle_id -> demand-zone quantities per deck and pinned
	// printing, so Conflicts can name the decks behind an over-commitment.
	demandRows map[string][]demandRow
}

// demandRow is one deck's demand-zone quantity for one Oracle and pinned
// printing ("" = any printing).
type demandRow struct {
	OracleID   string
	ScryfallID string
	DeckID     uint
	DeckName   string
	Quantity   int
}

// Conflict explains one cause of an Oracle's over-commitment.
type Conflict struct {
	// ScryfallID is the over-subscribed pinned printing, or "" when the decks
	// together want more copies of the card than are owned.
	ScryfallID string
	Owned      int
	Wanted     int
	// Decks lists the decks that want the card (or pin the printing), by name.
	Decks []DeckDemand
}

// DeckDemand is one deck's share of a Conflict's wanted copies.
type DeckDemand struct {
	DeckID   uint
	DeckName string
	Quantity int
}

// Conflicts explains why an Oracle is over-committed: one entry when the decks
// together want more copies than are owned, then one entry per pinned printing
// that decks want more often than it is owned (by scryfall_id). It returns nil
// when the Oracle is not over-committed.
func (m AvailabilityMap) Conflicts(oracleID string) []Conflict {
	a := m.OracleAvailabilityFor(oracleID)
	rows := m.demandRows[oracleID]
	var conflicts []Conflict

	if a.Demand > a.Owned {
		conflicts = append(conflicts, Conflict{
			Owned:  a.Owned,
			Wanted: a.Demand,
			Decks:  demandByDeck(rows, func(demandRow) bool { return true }),
		})
	}

	var pinned []string
	for _, row := range rows {
		if row.ScryfallID != "" && !slices.Contains(pinned, row.ScryfallID) &&
			m.pinnedByScryfall[row.ScryfallID] > m.ownedByScryfall[row.ScryfallID] {
			pinned = append(pinned, row.ScryfallID)
		}
	}
	slices.Sort(pinned)
	for _, scryfallID := range pinned {
		conflicts = append(conflicts, Conflict{
			ScryfallID: scryfallID,
			Owned:      m.ownedByScryfall[scryfallID],
			Wanted:     m.pinnedByScryfall[scryfallID],
			Decks:      demandByDeck(rows, func(r demandRow) bool { return r.ScryfallID == scryfallID }),
		})
	}

	return conflicts
}

// demandByDeck sums the matching rows per deck, sorted by deck name, then ID.
func demandByDeck(rows []demandRow, match func(demandRow) bool) []DeckDemand {
	var decks []DeckDemand
	for _, row := range rows {
		if !match(row) {
			continue
		}
		i := slices.IndexFunc(decks, func(d DeckDemand) bool { return d.DeckID == row.DeckID })
		if i < 0 {
			decks = append(decks, DeckDemand{DeckID: row.DeckID, DeckName: row.DeckName})
			i = len(decks) - 1
		}
		decks[i].Quantity += row.Quantity
	}
	slices.SortFunc(decks, func(a, b DeckDemand) int {
		return cmp.Or(cmp.Compare(a.DeckName, b.DeckName), cmp.Compare(a.DeckID, b.DeckID))
	})
	return decks
}

// OracleAvailabilityFor returns the availability for an Oracle ID. Oracle IDs
// with no inventory and no demand are absent from the map; this returns a
// zero-valued (all-zero, not over-committed) entry for them so callers can
// treat "unknown" as "nothing owned, nothing decked".
func (m AvailabilityMap) OracleAvailabilityFor(oracleID string) OracleAvailability {
	if a, ok := m.Oracle[oracleID]; ok {
		return a
	}
	return OracleAvailability{OracleID: oracleID}
}

// ComputeAvailability builds the full availability map from inventory and every
// deck's demand-zone items, using two GROUP BY aggregations (one over inventory,
// one over deck_items). Data is small for a personal collection, so no caching
// or pagination is needed — see IMPLEMENTATION_PLAN.md §"Performance".
func (s *AllocationService) ComputeAvailability(ctx context.Context) (AvailabilityMap, error) {
	db := s.db.WithContext(ctx)

	result := AvailabilityMap{
		Oracle:           make(map[string]OracleAvailability),
		ownedByScryfall:  make(map[string]int),
		pinnedByScryfall: make(map[string]int),
		demandRows:       make(map[string][]demandRow),
	}

	// Owned: aggregate inventory quantities by oracle_id and by scryfall_id.
	type ownedRow struct {
		OracleID   string
		ScryfallID string
		Quantity   int
	}
	var ownedRows []ownedRow
	if err := db.Model(&models.Inventory{}).
		Select("oracle_id, scryfall_id, SUM(quantity) AS quantity").
		Group("oracle_id, scryfall_id").
		Scan(&ownedRows).Error; err != nil {
		return AvailabilityMap{}, fmt.Errorf("aggregating inventory: %w", err)
	}

	for _, row := range ownedRows {
		a := result.Oracle[row.OracleID]
		a.OracleID = row.OracleID
		a.Owned += row.Quantity
		result.Oracle[row.OracleID] = a

		if row.ScryfallID != "" {
			result.ownedByScryfall[row.ScryfallID] += row.Quantity
		}
	}

	// Demand: aggregate deck_items in demand zones (maybe excluded) by oracle_id,
	// pinned scryfall_id and deck. The per-deck rows only feed Conflicts; the
	// totals below sum them back up.
	var demandRows []demandRow
	if err := db.Model(&models.DeckItem{}).
		Select("deck_items.oracle_id, deck_items.scryfall_id, deck_items.deck_id, "+
			"COALESCE(decks.name, '') AS deck_name, SUM(deck_items.desired_quantity) AS quantity").
		Joins("LEFT JOIN decks ON decks.id = deck_items.deck_id").
		Where("deck_items.zone IN ?", demandZones()).
		Group("deck_items.oracle_id, deck_items.scryfall_id, deck_items.deck_id, decks.name").
		Scan(&demandRows).Error; err != nil {
		return AvailabilityMap{}, fmt.Errorf("aggregating deck demand: %w", err)
	}

	// scryfallToOracle maps each pinned scryfall_id back to its Oracle so a
	// printing conflict can flag the whole Oracle as over-committed. A pinned
	// deck item always carries its own oracle_id, so this is built directly from
	// the demand rows already in hand — no extra query needed.
	scryfallToOracle := make(map[string]string)
	for _, row := range demandRows {
		a := result.Oracle[row.OracleID]
		a.OracleID = row.OracleID
		a.Demand += row.Quantity
		result.Oracle[row.OracleID] = a
		result.demandRows[row.OracleID] = append(result.demandRows[row.OracleID], row)

		// Pinned printings compete for a specific printing's pool.
		if row.ScryfallID != "" {
			result.pinnedByScryfall[row.ScryfallID] += row.Quantity
			scryfallToOracle[row.ScryfallID] = row.OracleID
		}
	}

	// Track which Oracles have an over-subscribed pinned printing.
	oraclePinnedConflict := make(map[string]bool)
	for scryfallID, pinned := range result.pinnedByScryfall {
		if pinned > result.ownedByScryfall[scryfallID] {
			if oracleID, ok := scryfallToOracle[scryfallID]; ok {
				oraclePinnedConflict[oracleID] = true
			}
		}
	}

	// Finalise free and over_committed per Oracle.
	for oracleID, a := range result.Oracle {
		a.Free = max(0, a.Owned-a.Demand)
		// over_committed = total demand exceeds supply OR any pinned printing of
		// this Oracle is individually over-subscribed (IMPLEMENTATION_PLAN.md §2).
		a.OverCommitted = a.Demand > a.Owned || oraclePinnedConflict[oracleID]
		result.Oracle[oracleID] = a
	}

	return result, nil
}

// demandZones returns the zones that lock inventory (everything except maybe),
// matching models.DeckZone.CountsAsDemand. Tokens in the other zone lock the
// token copies you own, so they are tracked across decks like cards.
func demandZones() []models.DeckZone {
	return []models.DeckZone{models.ZoneMain, models.ZoneSide, models.ZoneCommand, models.ZoneOther}
}
