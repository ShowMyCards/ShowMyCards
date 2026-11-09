import { describe, expect, it } from 'vitest';
import { buildDeckItems, type AddToDeckCard } from './deck-add';

function card(oracle: string, printing: string, copies = 1): AddToDeckCard {
	return { oracle_id: oracle, scryfall_id: printing, name: oracle, copies };
}

describe('buildDeckItems', () => {
	it('merges printings of the same card into one any-printing item when unpinned', () => {
		const items = buildDeckItems([card('sol', 'cmm', 2), card('sol', 'c21', 1)], {
			zone: 'main',
			pin: false,
			quantity: 'copies'
		});
		expect(items).toEqual([
			{ oracle_id: 'sol', scryfall_id: '', treatment: '', desired_quantity: 3, zone: 'main' }
		]);
	});

	it('keeps pinned printings as separate items', () => {
		const items = buildDeckItems([card('sol', 'cmm', 2), card('sol', 'c21', 1)], {
			zone: 'side',
			pin: true,
			quantity: 'copies'
		});
		expect(items).toEqual([
			{ oracle_id: 'sol', scryfall_id: 'cmm', treatment: '', desired_quantity: 2, zone: 'side' },
			{ oracle_id: 'sol', scryfall_id: 'c21', treatment: '', desired_quantity: 1, zone: 'side' }
		]);
	});

	it('adds a fixed quantity per group regardless of copies', () => {
		const items = buildDeckItems(
			[card('sol', 'cmm', 4), card('sol', 'c21', 2), card('bolt', 'm10', 3)],
			{ zone: 'main', pin: false, quantity: 1 }
		);
		expect(items.map((i) => [i.oracle_id, i.desired_quantity])).toEqual([
			['sol', 1],
			['bolt', 1]
		]);
	});

	it('sums copies per distinct card', () => {
		const items = buildDeckItems([card('sol', 'cmm', 2), card('bolt', 'm10', 4)], {
			zone: 'main',
			pin: false,
			quantity: 'copies'
		});
		expect(items.map((i) => [i.oracle_id, i.desired_quantity])).toEqual([
			['sol', 2],
			['bolt', 4]
		]);
	});

	it('drops groups with nothing to add', () => {
		const items = buildDeckItems([card('sol', 'cmm', 0)], {
			zone: 'main',
			pin: false,
			quantity: 'copies'
		});
		expect(items).toEqual([]);
	});
});
