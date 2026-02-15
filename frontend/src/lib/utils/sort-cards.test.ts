import { describe, it, expect } from 'vitest';
import {
	sortBySetAndCollectorNumber,
	createSetCollectorComparator,
	sortCardsBySetAndCollector,
	type SetSortable
} from './sort-cards';

describe('sortBySetAndCollectorNumber', () => {
	describe('standard numeric collector numbers', () => {
		it('should sort cards by collector number within same set', () => {
			const cardA = { set_name: 'Alpha', collector_number: '10' };
			const cardB = { set_name: 'Alpha', collector_number: '20' };

			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(cardB, cardA)).toBeGreaterThan(0);
		});

		it('should handle larger collector numbers correctly', () => {
			const card1 = { set_name: 'Beta', collector_number: '1' };
			const card50 = { set_name: 'Beta', collector_number: '50' };
			const card100 = { set_name: 'Beta', collector_number: '100' };

			expect(sortBySetAndCollectorNumber(card1, card50)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(card50, card100)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(card1, card100)).toBeLessThan(0);
		});

		it('should return 0 for identical set and collector number', () => {
			const cardA = { set_name: 'Alpha', collector_number: '123' };
			const cardB = { set_name: 'Alpha', collector_number: '123' };

			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBe(0);
		});
	});

	describe('collector numbers with letters', () => {
		it('should sort cards with letter suffixes by base number', () => {
			const card123 = { set_name: 'Set', collector_number: '123' };
			const card123a = { set_name: 'Set', collector_number: '123a' };
			const card123b = { set_name: 'Set', collector_number: '123b' };

			// All have same base number, so should be equal
			expect(sortBySetAndCollectorNumber(card123, card123a)).toBe(0);
			expect(sortBySetAndCollectorNumber(card123a, card123b)).toBe(0);
		});

		it('should sort different base numbers with letters correctly', () => {
			const card10a = { set_name: 'Set', collector_number: '10a' };
			const card20b = { set_name: 'Set', collector_number: '20b' };

			expect(sortBySetAndCollectorNumber(card10a, card20b)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(card20b, card10a)).toBeGreaterThan(0);
		});
	});

	describe('special symbols in collector numbers', () => {
		it('should handle star symbols (★123)', () => {
			const card10 = { set_name: 'Set', collector_number: '10' };
			const cardStar123 = { set_name: 'Set', collector_number: '★123' };

			expect(sortBySetAndCollectorNumber(card10, cardStar123)).toBeLessThan(0);
		});

		it('should handle other non-digit prefixes', () => {
			const card50 = { set_name: 'Set', collector_number: '50' };
			const cardSpecial100 = { set_name: 'Set', collector_number: 'S100' };

			expect(sortBySetAndCollectorNumber(card50, cardSpecial100)).toBeLessThan(0);
		});
	});

	describe('missing or null values', () => {
		it('should handle missing collector numbers', () => {
			const cardWithNum = { set_name: 'Set', collector_number: '50' };
			const cardWithoutNum = { set_name: 'Set' };

			expect(sortBySetAndCollectorNumber(cardWithoutNum, cardWithNum)).toBeLessThan(0);
		});

		it('should handle null collector numbers', () => {
			const cardWithNum: SetSortable = { set_name: 'Set', collector_number: '50' };
			const cardWithNull: SetSortable = { set_name: 'Set', collector_number: null };

			expect(sortBySetAndCollectorNumber(cardWithNull, cardWithNum)).toBeLessThan(0);
		});

		it('should handle missing set names', () => {
			const cardA = { collector_number: '10' };
			const cardB = { set_name: 'Alpha', collector_number: '10' };

			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBeLessThan(0);
		});

		it('should handle null set names', () => {
			const cardA: SetSortable = { set_name: null, collector_number: '10' };
			const cardB: SetSortable = { set_name: 'Beta', collector_number: '10' };

			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBeLessThan(0);
		});

		it('should handle completely empty objects', () => {
			const cardA: SetSortable = {};
			const cardB: SetSortable = {};

			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBe(0);
		});
	});

	describe('set name sorting', () => {
		it('should sort different sets alphabetically', () => {
			const cardAlpha = { set_name: 'Alpha', collector_number: '1' };
			const cardBeta = { set_name: 'Beta', collector_number: '1' };
			const cardGamma = { set_name: 'Gamma', collector_number: '1' };

			expect(sortBySetAndCollectorNumber(cardAlpha, cardBeta)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(cardBeta, cardGamma)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(cardAlpha, cardGamma)).toBeLessThan(0);
		});

		it('should be case-sensitive in set name sorting', () => {
			const cardLower = { set_name: 'alpha', collector_number: '1' };
			const cardUpper = { set_name: 'Alpha', collector_number: '1' };

			// localeCompare is case-sensitive
			const result = sortBySetAndCollectorNumber(cardUpper, cardLower);
			expect(result).not.toBe(0);
		});

		it('should prioritize set name over collector number', () => {
			const cardAlpha999 = { set_name: 'Alpha', collector_number: '999' };
			const cardBeta1 = { set_name: 'Beta', collector_number: '1' };

			// Even though 999 > 1, Alpha comes before Beta
			expect(sortBySetAndCollectorNumber(cardAlpha999, cardBeta1)).toBeLessThan(0);
		});
	});

	describe('edge cases from Scryfall data', () => {
		it('should handle promo numbers (P1, P2, etc.)', () => {
			const card1 = { set_name: 'Set', collector_number: '1' };
			const cardP10 = { set_name: 'Set', collector_number: 'P10' };

			// P10 should sort after 1 due to numeric extraction
			expect(sortBySetAndCollectorNumber(card1, cardP10)).toBeLessThan(0);
		});

		it('should handle tokens (T1, T2, etc.)', () => {
			const card5 = { set_name: 'Set', collector_number: '5' };
			const cardT100 = { set_name: 'Set', collector_number: 'T100' };

			expect(sortBySetAndCollectorNumber(card5, cardT100)).toBeLessThan(0);
		});

		it('should handle non-numeric collector numbers', () => {
			const cardA = { set_name: 'Set', collector_number: 'ABC' };
			const cardB = { set_name: 'Set', collector_number: 'XYZ' };

			// Both extract to 0, so should be equal
			expect(sortBySetAndCollectorNumber(cardA, cardB)).toBe(0);
		});

		it('should handle mixed alphanumeric collector numbers', () => {
			const card10 = { set_name: 'Set', collector_number: '10' };
			const card20a = { set_name: 'Set', collector_number: '20a' };
			const card5x = { set_name: 'Set', collector_number: '5x' };

			expect(sortBySetAndCollectorNumber(card5x, card10)).toBeLessThan(0);
			expect(sortBySetAndCollectorNumber(card10, card20a)).toBeLessThan(0);
		});
	});
});

describe('createSetCollectorComparator', () => {
	it('should return a comparator function', () => {
		const comparator = createSetCollectorComparator();
		expect(typeof comparator).toBe('function');
	});

	it('should create a function that sorts correctly', () => {
		const cards = [
			{ set_name: 'Beta', collector_number: '50' },
			{ set_name: 'Alpha', collector_number: '10' },
			{ set_name: 'Alpha', collector_number: '5' }
		];

		const comparator = createSetCollectorComparator();
		const sorted = [...cards].sort(comparator);

		expect(sorted[0].set_name).toBe('Alpha');
		expect(sorted[0].collector_number).toBe('5');
		expect(sorted[1].set_name).toBe('Alpha');
		expect(sorted[1].collector_number).toBe('10');
		expect(sorted[2].set_name).toBe('Beta');
		expect(sorted[2].collector_number).toBe('50');
	});

	it('should work with array.sort()', () => {
		const cards = [
			{ set_name: 'Gamma', collector_number: '1' },
			{ set_name: 'Alpha', collector_number: '1' },
			{ set_name: 'Beta', collector_number: '1' }
		];

		cards.sort(createSetCollectorComparator());

		expect(cards[0].set_name).toBe('Alpha');
		expect(cards[1].set_name).toBe('Beta');
		expect(cards[2].set_name).toBe('Gamma');
	});
});

describe('sortCardsBySetAndCollector', () => {
	it('should return a new sorted array', () => {
		const cards = [
			{ set_name: 'Beta', collector_number: '50' },
			{ set_name: 'Alpha', collector_number: '10' }
		];

		const sorted = sortCardsBySetAndCollector(cards);

		// Should be a new array
		expect(sorted).not.toBe(cards);

		// Should be sorted correctly
		expect(sorted[0].set_name).toBe('Alpha');
		expect(sorted[1].set_name).toBe('Beta');
	});

	it('should not modify the original array', () => {
		const cards = [
			{ set_name: 'Beta', collector_number: '50' },
			{ set_name: 'Alpha', collector_number: '10' }
		];

		const originalOrder = [...cards];
		sortCardsBySetAndCollector(cards);

		// Original array should be unchanged
		expect(cards).toEqual(originalOrder);
		expect(cards[0].set_name).toBe('Beta');
	});

	it('should handle empty arrays', () => {
		const cards: SetSortable[] = [];
		const sorted = sortCardsBySetAndCollector(cards);

		expect(sorted).toEqual([]);
		expect(sorted).not.toBe(cards); // Still returns new array
	});

	it('should handle single-item arrays', () => {
		const cards = [{ set_name: 'Alpha', collector_number: '1' }];
		const sorted = sortCardsBySetAndCollector(cards);

		expect(sorted).toEqual(cards);
		expect(sorted).not.toBe(cards); // Still returns new array
	});

	it('should sort complex real-world example', () => {
		const cards = [
			{ set_name: 'Innistrad', collector_number: '250' },
			{ set_name: 'Innistrad', collector_number: '10' },
			{ set_name: 'Innistrad', collector_number: '100a' },
			{ set_name: 'Alpha', collector_number: '50' },
			{ set_name: 'Zendikar', collector_number: '1' },
			{ set_name: 'Alpha', collector_number: '5' }
		];

		const sorted = sortCardsBySetAndCollector(cards);

		// Should be sorted by set name first
		expect(sorted[0].set_name).toBe('Alpha');
		expect(sorted[1].set_name).toBe('Alpha');
		expect(sorted[2].set_name).toBe('Innistrad');
		expect(sorted[3].set_name).toBe('Innistrad');
		expect(sorted[4].set_name).toBe('Innistrad');
		expect(sorted[5].set_name).toBe('Zendikar');

		// Within Alpha, should be sorted by number
		expect(sorted[0].collector_number).toBe('5');
		expect(sorted[1].collector_number).toBe('50');

		// Within Innistrad, should be sorted by number
		expect(sorted[2].collector_number).toBe('10');
		expect(sorted[3].collector_number).toBe('100a');
		expect(sorted[4].collector_number).toBe('250');
	});

	it('should preserve object properties', () => {
		interface Card extends SetSortable {
			id: string;
			name: string;
		}

		const cards: Card[] = [
			{ id: '1', name: 'Card B', set_name: 'Beta', collector_number: '1' },
			{ id: '2', name: 'Card A', set_name: 'Alpha', collector_number: '1' }
		];

		const sorted = sortCardsBySetAndCollector(cards);

		expect(sorted[0].id).toBe('2');
		expect(sorted[0].name).toBe('Card A');
		expect(sorted[1].id).toBe('1');
		expect(sorted[1].name).toBe('Card B');
	});
});
