import { describe, expect, it } from 'vitest';
import type { EnrichedDeckItem, EnrichedListItem } from '#lib';
import { parseCardList } from './card-list-parser';
import {
	deckExportLines,
	formatExportLines,
	listExportLines,
	type DeckExportItems
} from './card-list-export';

/**
 * An any-printing deck row. Like the backend, it still carries a representative
 * printing's set data, which the export must ignore.
 */
function deckItem(overrides: Partial<EnrichedDeckItem> = {}): EnrichedDeckItem {
	return {
		id: 1,
		created_at: '',
		updated_at: '',
		deck_id: 1,
		oracle_id: 'bolt',
		scryfall_id: '',
		treatment: 'nonfoil',
		zone: 'main',
		desired_quantity: 1,
		name: 'Lightning Bolt',
		set_name: 'Magic 2010',
		set_code: 'm10',
		collector_number: '146',
		owned: 0,
		under_owned: false,
		over_committed: false,
		...overrides
	};
}

/** A deck row pinned to a specific printing (Magic 2010 unless overridden). */
function pinnedItem(overrides: Partial<EnrichedDeckItem> = {}): EnrichedDeckItem {
	return deckItem({ scryfall_id: 'bolt-m10', ...overrides });
}

function deck(items: EnrichedDeckItem[]): DeckExportItems {
	return {
		command: items.filter((i) => i.zone === 'command'),
		main: items.filter((i) => i.zone === 'main'),
		side: items.filter((i) => i.zone === 'side'),
		maybe: items.filter((i) => i.zone === 'maybe')
	};
}

function listItem(overrides: Partial<EnrichedListItem> = {}): EnrichedListItem {
	return {
		id: 1,
		created_at: '',
		updated_at: '',
		list_id: 1,
		scryfall_id: 'bolt-m10',
		oracle_id: 'bolt',
		treatment: 'nonfoil',
		desired_quantity: 1,
		collected_quantity: 0,
		name: 'Lightning Bolt',
		set_name: 'Magic 2010',
		set_code: 'm10',
		collector_number: '146',
		...overrides
	};
}

describe('deckExportLines', () => {
	it('sums every zone, including the maybe-board, into one line per card', () => {
		const items = deck([
			deckItem({ zone: 'command', oracle_id: 'atraxa', name: 'Atraxa, Grand Unifier' }),
			deckItem({ zone: 'main', desired_quantity: 3 }),
			deckItem({ zone: 'side', desired_quantity: 1 }),
			deckItem({ zone: 'maybe', oracle_id: 'ice', name: 'Fire // Ice', desired_quantity: 2 })
		]);

		expect(deckExportLines(items, 'all')).toEqual({
			lines: [
				{ quantity: 1, name: 'Atraxa, Grand Unifier' },
				{ quantity: 2, name: 'Fire // Ice' },
				{ quantity: 4, name: 'Lightning Bolt' }
			],
			skipped: 0
		});
	});

	it('names the expansion of pinned printings only', () => {
		const items = deck([
			pinnedItem({ desired_quantity: 1, treatment: 'foil' }),
			pinnedItem({
				desired_quantity: 1,
				scryfall_id: 'bolt-2xm',
				set_name: 'Double Masters'
			}),
			deckItem({ desired_quantity: 2 })
		]);

		expect(deckExportLines(items, 'all').lines).toEqual([
			{ quantity: 2, name: 'Lightning Bolt' },
			{ quantity: 1, name: 'Lightning Bolt', expansion: 'Double Masters' },
			{ quantity: 1, name: 'Lightning Bolt', expansion: 'Magic 2010' }
		]);
	});

	it('merges the same pinned printing across zones', () => {
		const items = deck([
			pinnedItem({ zone: 'main', desired_quantity: 3 }),
			pinnedItem({ zone: 'side', desired_quantity: 1, treatment: 'foil' })
		]);

		expect(deckExportLines(items, 'all').lines).toEqual([
			{ quantity: 4, name: 'Lightning Bolt', expansion: 'Magic 2010' }
		]);
	});

	it('falls back to the bare name for a pinned row without set data', () => {
		const items = deck([pinnedItem({ set_name: undefined })]);

		expect(deckExportLines(items, 'all').lines).toEqual([{ quantity: 1, name: 'Lightning Bolt' }]);
	});

	it('exports wanted minus owned per card, counted once across zones', () => {
		// 3 main + 1 side, 2 owned. Owned is an Oracle-level count stamped on every row.
		const items = deck([
			deckItem({ zone: 'main', desired_quantity: 3, owned: 2 }),
			deckItem({ zone: 'side', desired_quantity: 1, owned: 2 })
		]);

		expect(deckExportLines(items, 'missing').lines).toEqual([
			{ quantity: 2, name: 'Lightning Bolt' }
		]);
	});

	it('counts owned copies toward any-printing rows first, keeping pinned printings to buy', () => {
		// Owned copies don't record their printing, so they satisfy "any printing"
		// before a specific one — even when the pinned row sits in an earlier zone.
		const items = deck([
			pinnedItem({ zone: 'command', desired_quantity: 1, owned: 1 }),
			deckItem({ zone: 'main', desired_quantity: 2, owned: 1 })
		]);

		expect(deckExportLines(items, 'missing').lines).toEqual([
			{ quantity: 1, name: 'Lightning Bolt' },
			{ quantity: 1, name: 'Lightning Bolt', expansion: 'Magic 2010' }
		]);
	});

	it('fills pinned rows in zone order once any-printing rows are covered', () => {
		const items = deck([
			pinnedItem({ zone: 'main', desired_quantity: 1, owned: 1 }),
			pinnedItem({
				zone: 'side',
				desired_quantity: 1,
				owned: 1,
				scryfall_id: 'bolt-2xm',
				set_name: 'Double Masters'
			})
		]);

		expect(deckExportLines(items, 'missing').lines).toEqual([
			{ quantity: 1, name: 'Lightning Bolt', expansion: 'Double Masters' }
		]);
	});

	it('counts maybe-board demand as missing once the real deck has used up owned copies', () => {
		const items = deck([
			deckItem({ zone: 'main', desired_quantity: 1, owned: 1 }),
			deckItem({ zone: 'maybe', desired_quantity: 1, owned: 1 })
		]);

		expect(deckExportLines(items, 'missing').lines).toEqual([
			{ quantity: 1, name: 'Lightning Bolt' }
		]);
	});

	it('leaves out cards that are fully owned', () => {
		const items = deck([
			pinnedItem({ desired_quantity: 4, owned: 7 }),
			deckItem({ oracle_id: 'ice', name: 'Fire // Ice', desired_quantity: 1, owned: 0 })
		]);

		expect(deckExportLines(items, 'missing').lines).toEqual([{ quantity: 1, name: 'Fire // Ice' }]);
	});

	it('skips and counts rows without card data', () => {
		const items = deck([
			deckItem(),
			deckItem({ oracle_id: 'unknown', name: undefined }),
			deckItem({ oracle_id: 'blank', name: '' })
		]);

		expect(deckExportLines(items, 'all')).toEqual({
			lines: [{ quantity: 1, name: 'Lightning Bolt' }],
			skipped: 2
		});
	});

	it('returns nothing for an empty deck', () => {
		expect(deckExportLines(deck([]), 'missing')).toEqual({ lines: [], skipped: 0 });
	});
});

describe('listExportLines', () => {
	it('merges finishes of the same printing and splits printings by expansion', () => {
		const items = [
			listItem({ desired_quantity: 2 }),
			listItem({ id: 2, treatment: 'foil', desired_quantity: 1 }),
			listItem({ id: 3, scryfall_id: 'bolt-2xm', set_name: 'Double Masters', desired_quantity: 1 })
		];

		expect(listExportLines(items, 'all').lines).toEqual([
			{ quantity: 1, name: 'Lightning Bolt', expansion: 'Double Masters' },
			{ quantity: 3, name: 'Lightning Bolt', expansion: 'Magic 2010' }
		]);
	});

	it('exports desired minus collected, clamped per item before merging', () => {
		// An over-collected foil must not cancel out the still-missing nonfoil.
		const items = [
			listItem({ desired_quantity: 2, collected_quantity: 0 }),
			listItem({ id: 2, treatment: 'foil', desired_quantity: 1, collected_quantity: 3 }),
			listItem({
				id: 3,
				oracle_id: 'delver',
				name: 'Delver of Secrets // Insectile Aberration',
				set_name: 'Innistrad',
				desired_quantity: 1,
				collected_quantity: 1
			})
		];

		expect(listExportLines(items, 'missing').lines).toEqual([
			{ quantity: 2, name: 'Lightning Bolt', expansion: 'Magic 2010' }
		]);
	});

	it('skips and counts items without card data', () => {
		const items = [listItem(), listItem({ id: 2, oracle_id: 'x', name: undefined })];

		expect(listExportLines(items, 'all')).toEqual({
			lines: [{ quantity: 1, name: 'Lightning Bolt', expansion: 'Magic 2010' }],
			skipped: 1
		});
	});
});

describe('formatExportLines', () => {
	it('writes Cardmarket Wants lines, with the expansion in parentheses when known', () => {
		expect(
			formatExportLines([
				{ quantity: 2, name: 'Delver of Secrets // Insectile Aberration' },
				{ quantity: 1, name: 'Dark Confidant', expansion: 'Modern Masters' }
			])
		).toBe('2 Delver of Secrets // Insectile Aberration\n1 Dark Confidant (Modern Masters)');
	});

	it('returns an empty string for no lines', () => {
		expect(formatExportLines([])).toBe('');
	});

	it('round-trips name-only lines through the import parser', () => {
		// Expansion lines are Cardmarket syntax; the app's import only reads set codes.
		const lines = [
			{ quantity: 1, name: 'Atraxa, Grand Unifier' },
			{ quantity: 2, name: 'Fire // Ice' },
			{ quantity: 4, name: "Ashnod's Altar" }
		];

		const { cards, errors } = parseCardList(formatExportLines(lines));
		expect(errors).toEqual([]);
		expect(cards.map(({ quantity, name }) => ({ quantity, name }))).toEqual(lines);
	});
});
