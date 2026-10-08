import { describe, expect, it } from 'vitest';
import { inventoryCard, inventoryItem } from '$lib/test/fixtures/inventory';
import { getInventoryUnitPrice, sortInventoryCardsByPrice } from './card-prices';

describe('inventory prices', () => {
	it('uses the highest owned treatment price rather than total stack value or an unowned finish', () => {
		const card = inventoryCard(
			'one',
			[inventoryItem(1, { quantity: 100 }), inventoryItem(2, { treatment: 'foil' })],
			{ prices: { usd: '2.00', usd_foil: '9.00', usd_etched: '50.00' } }
		);

		expect(getInventoryUnitPrice(card, 'usd')).toBe(9);
	});

	it('uses the selected currency and existing etched price mapping', () => {
		const card = inventoryCard('one', [inventoryItem(1, { treatment: 'etched' })], {
			prices: { usd: '1.00', usd_etched: '12.00', eur: '0.50', eur_foil: '7.00' }
		});
		expect(getInventoryUnitPrice(card, 'usd')).toBe(12);
		expect(getInventoryUnitPrice(card, 'eur')).toBe(7);
	});

	it('keeps a missing currency price unavailable rather than mixing currencies', () => {
		const card = inventoryCard('one', [inventoryItem(1)], { prices: { usd: '10.00' } });
		expect(getInventoryUnitPrice(card, 'eur')).toBeUndefined();
	});

	it.each([undefined, '', ' ', 'invalid', '-1.00', 'Infinity'])(
		'treats %s as unavailable pricing',
		(usd) => {
			const card = inventoryCard('one', [inventoryItem(1)], { prices: { usd } });
			expect(getInventoryUnitPrice(card, 'usd')).toBeUndefined();
		}
	);

	it('ignores rows without owned copies and accepts a zero price', () => {
		const card = inventoryCard(
			'one',
			[
				inventoryItem(1, { treatment: '', quantity: 1 }),
				inventoryItem(2, { treatment: 'foil', quantity: 0 })
			],
			{ prices: { usd: '0.00', usd_foil: '50.00' } }
		);
		expect(getInventoryUnitPrice(card, 'usd')).toBe(0);
	});

	it('sorts numerically in both directions, keeps ties stable and unknown prices last', () => {
		const cards = [
			inventoryCard('unknown', [inventoryItem(1)], { prices: {} }),
			inventoryCard('two', [inventoryItem(2)], { prices: { usd: '2.00' } }),
			inventoryCard('ten', [inventoryItem(3)], { prices: { usd: '10.00' } }),
			inventoryCard('two-again', [inventoryItem(4)], { prices: { usd: '2.00' } }),
			inventoryCard('zero', [inventoryItem(5)], { prices: { usd: '0.00' } })
		];

		expect(sortInventoryCardsByPrice(cards, 'usd', 'desc').map((card) => card.id)).toEqual([
			'ten',
			'two',
			'two-again',
			'zero',
			'unknown'
		]);
		expect(sortInventoryCardsByPrice(cards, 'usd', 'asc').map((card) => card.id)).toEqual([
			'zero',
			'two',
			'two-again',
			'ten',
			'unknown'
		]);
		expect(cards.map((card) => card.id)).toEqual(['unknown', 'two', 'ten', 'two-again', 'zero']);
	});
});
