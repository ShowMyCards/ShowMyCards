import { describe, expect, it } from 'vitest';
import {
	typeBucket,
	manaValueBucket,
	compareCards,
	groupCards,
	type GroupableCard
} from './deck-grouping';

describe('typeBucket', () => {
	it('places by Moxfield priority', () => {
		expect(typeBucket('Artifact Creature — Golem')).toBe('Creatures');
		expect(typeBucket('Legendary Artifact')).toBe('Artifacts');
		expect(typeBucket('Enchantment Land')).toBe('Lands');
		expect(typeBucket('Basic Land — Forest')).toBe('Lands');
		expect(typeBucket('Instant')).toBe('Instants');
		expect(typeBucket('Legendary Planeswalker — Jace')).toBe('Planeswalkers');
		expect(typeBucket('Enchantment — Aura')).toBe('Enchantments');
		expect(typeBucket('')).toBe('Other');
	});
});

describe('manaValueBucket', () => {
	it('buckets 0-6, caps at 7+, and separates lands', () => {
		expect(manaValueBucket({ cmc: 0 })).toBe('0');
		expect(manaValueBucket({ cmc: 3 })).toBe('3');
		expect(manaValueBucket({ cmc: 6 })).toBe('6');
		expect(manaValueBucket({ cmc: 7 })).toBe('7+');
		expect(manaValueBucket({ cmc: 12 })).toBe('7+');
		expect(manaValueBucket({ cmc: 0, type_line: 'Basic Land — Island' })).toBe('Lands');
	});
});

describe('compareCards', () => {
	it('sorts by mana value with a name tiebreak', () => {
		expect(compareCards({ cmc: 1, name: 'B' }, { cmc: 3, name: 'A' }, 'manaValue')).toBeLessThan(0);
		expect(
			compareCards({ cmc: 2, name: 'Bolt' }, { cmc: 2, name: 'Ancestral' }, 'manaValue')
		).toBeGreaterThan(0);
	});

	it('sorts by name', () => {
		expect(compareCards({ name: 'Alpha' }, { name: 'Beta' }, 'name')).toBeLessThan(0);
	});
});

describe('groupCards', () => {
	const cards: GroupableCard[] = [
		{ name: 'Llanowar Elves', cmc: 1, type_line: 'Creature — Elf Druid' },
		{ name: 'Forest', cmc: 0, type_line: 'Basic Land — Forest' },
		{ name: 'Sol Ring', cmc: 1, type_line: 'Artifact' },
		{ name: 'Ghalta', cmc: 12, type_line: 'Legendary Creature — Dinosaur' }
	];

	it('returns one unlabelled group for group=none, sorted', () => {
		const groups = groupCards(cards, 'none', 'name');
		expect(groups).toHaveLength(1);
		expect(groups[0].items.map((c) => c.name)).toEqual([
			'Forest',
			'Ghalta',
			'Llanowar Elves',
			'Sol Ring'
		]);
	});

	it('groups by type in display order (Lands last)', () => {
		const groups = groupCards(cards, 'type', 'manaValue');
		expect(groups.map((g) => g.label)).toEqual(['Creatures', 'Artifacts', 'Lands']);
		// Creatures sorted by mana value: Llanowar (1) before Ghalta (12).
		expect(groups[0].items.map((c) => c.name)).toEqual(['Llanowar Elves', 'Ghalta']);
	});

	it('groups by mana value with lands separated', () => {
		const groups = groupCards(cards, 'manaValue', 'name');
		expect(groups.map((g) => g.label)).toEqual(['1', '7+', 'Lands']);
	});
});
