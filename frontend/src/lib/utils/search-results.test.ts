import { describe, it, expect } from 'vitest';
import { getUniqueTreatments, groupCardsByTreatment } from './search-results';
import type { ScryfallCard } from '../types/runtime';

// Helper to create minimal test cards
function createCard(
	id: string,
	name: string,
	finishes?: string[]
): Partial<ScryfallCard> & { id: string; name: string; oracle_id: string } {
	return {
		id,
		name,
		oracle_id: `oracle-${id}`,
		finishes
	};
}

describe('getUniqueTreatments', () => {
	it('should return unique treatments from cards with single treatment', () => {
		const cards = [
			createCard('1', 'Card A', ['nonfoil']),
			createCard('2', 'Card B', ['nonfoil']),
			createCard('3', 'Card C', ['nonfoil'])
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['nonfoil']);
	});

	it('should return unique treatments from cards with multiple treatments', () => {
		const cards = [
			createCard('1', 'Card A', ['nonfoil', 'foil']),
			createCard('2', 'Card B', ['nonfoil', 'foil']),
			createCard('3', 'Card C', ['etched'])
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['etched', 'foil', 'nonfoil']);
	});

	it('should handle cards with special finishes', () => {
		const cards = [
			createCard('1', 'Card A', ['nonfoil', 'foil', 'etched', 'glossy'])
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['etched', 'foil', 'glossy', 'nonfoil']);
	});

	it('should default to nonfoil for cards with no finishes array', () => {
		const cards = [createCard('1', 'Card A')] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['nonfoil']);
	});

	it('should default to nonfoil for cards with empty finishes array', () => {
		const cards = [createCard('1', 'Card A', [])] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['nonfoil']);
	});

	it('should return empty array for empty card list', () => {
		const cards: ScryfallCard[] = [];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual([]);
	});

	it('should sort treatments alphabetically', () => {
		const cards = [
			createCard('1', 'Card A', ['foil']),
			createCard('2', 'Card B', ['etched']),
			createCard('3', 'Card C', ['nonfoil']),
			createCard('4', 'Card D', ['glossy'])
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['etched', 'foil', 'glossy', 'nonfoil']);
	});

	it('should deduplicate treatments across multiple cards', () => {
		const cards = [
			createCard('1', 'Card A', ['foil', 'nonfoil']),
			createCard('2', 'Card B', ['foil', 'nonfoil']),
			createCard('3', 'Card C', ['foil', 'nonfoil'])
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['foil', 'nonfoil']);
	});

	it('should handle mix of cards with and without finishes', () => {
		const cards = [
			createCard('1', 'Card A', ['foil']),
			createCard('2', 'Card B'), // no finishes
			createCard('3', 'Card C', []) // empty finishes
		] as ScryfallCard[];

		const treatments = getUniqueTreatments(cards);
		expect(treatments).toEqual(['foil', 'nonfoil']);
	});
});

describe('groupCardsByTreatment', () => {
	it('should group cards by single treatment', () => {
		const cards = [
			createCard('1', 'Card A', ['nonfoil']),
			createCard('2', 'Card B', ['nonfoil'])
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['nonfoil']);
		expect(result.cardsByTreatment.size).toBe(1);
		expect(result.cardsByTreatment.get('nonfoil')).toHaveLength(2);
	});

	it('should group cards by multiple treatments', () => {
		const cards = [
			createCard('1', 'Card A', ['nonfoil', 'foil']),
			createCard('2', 'Card B', ['nonfoil']),
			createCard('3', 'Card C', ['foil'])
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['foil', 'nonfoil']);
		expect(result.cardsByTreatment.size).toBe(2);
		expect(result.cardsByTreatment.get('nonfoil')).toHaveLength(2); // Cards A, B
		expect(result.cardsByTreatment.get('foil')).toHaveLength(2); // Cards A, C
	});

	it('should add selectedTreatment property to grouped cards', () => {
		const cards = [createCard('1', 'Card A', ['nonfoil', 'foil'])] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		const nonfoilCards = result.cardsByTreatment.get('nonfoil')!;
		expect(nonfoilCards[0].selectedTreatment).toBe('nonfoil');

		const foilCards = result.cardsByTreatment.get('foil')!;
		expect(foilCards[0].selectedTreatment).toBe('foil');
	});

	it('should handle cards with no finishes (default to nonfoil)', () => {
		const cards = [createCard('1', 'Card A')] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['nonfoil']);
		expect(result.cardsByTreatment.get('nonfoil')).toHaveLength(1);
		expect(result.cardsByTreatment.get('nonfoil')![0].selectedTreatment).toBe('nonfoil');
	});

	it('should handle cards with empty finishes array', () => {
		const cards = [createCard('1', 'Card A', [])] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['nonfoil']);
		expect(result.cardsByTreatment.get('nonfoil')).toHaveLength(1);
	});

	it('should handle empty card list', () => {
		const cards: ScryfallCard[] = [];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual([]);
		expect(result.cardsByTreatment.size).toBe(0);
	});

	it('should preserve all card properties', () => {
		const cards = [
			{
				id: '1',
				name: 'Card A',
				oracle_id: 'oracle-1',
				finishes: ['nonfoil'],
				set_name: 'Test Set',
				collector_number: '123'
			}
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		const nonfoilCard = result.cardsByTreatment.get('nonfoil')![0];
		expect(nonfoilCard.id).toBe('1');
		expect(nonfoilCard.name).toBe('Card A');
		expect(nonfoilCard.oracle_id).toBe('oracle-1');
		expect(nonfoilCard.set_name).toBe('Test Set');
		expect(nonfoilCard.collector_number).toBe('123');
		expect(nonfoilCard.selectedTreatment).toBe('nonfoil');
	});

	it('should handle special finishes', () => {
		const cards = [
			createCard('1', 'Card A', ['etched', 'glossy']),
			createCard('2', 'Card B', ['etched'])
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['etched', 'glossy']);
		expect(result.cardsByTreatment.get('etched')).toHaveLength(2);
		expect(result.cardsByTreatment.get('glossy')).toHaveLength(1);
	});

	it('should create separate card instances for each treatment', () => {
		const cards = [createCard('1', 'Card A', ['nonfoil', 'foil'])] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		const nonfoilCard = result.cardsByTreatment.get('nonfoil')![0];
		const foilCard = result.cardsByTreatment.get('foil')![0];

		// Should be different objects
		expect(nonfoilCard).not.toBe(foilCard);

		// But have same base properties
		expect(nonfoilCard.id).toBe(foilCard.id);
		expect(nonfoilCard.name).toBe(foilCard.name);

		// With different selectedTreatment
		expect(nonfoilCard.selectedTreatment).toBe('nonfoil');
		expect(foilCard.selectedTreatment).toBe('foil');
	});

	it('should sort treatments alphabetically', () => {
		const cards = [
			createCard('1', 'Card A', ['foil', 'nonfoil', 'etched', 'glossy'])
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['etched', 'foil', 'glossy', 'nonfoil']);
	});

	it('should handle complex real-world scenario', () => {
		const cards = [
			createCard('1', 'Lightning Bolt', ['nonfoil', 'foil']),
			createCard('2', 'Black Lotus', ['nonfoil']),
			createCard('3', 'Mox Ruby', ['foil']),
			createCard('4', 'Ancestral Recall', ['nonfoil', 'foil', 'etched']),
			createCard('5', 'Time Walk') // no finishes
		] as ScryfallCard[];

		const result = groupCardsByTreatment(cards);

		expect(result.treatments).toEqual(['etched', 'foil', 'nonfoil']);

		// Check nonfoil group (Cards 1, 2, 4, 5)
		const nonfoilCards = result.cardsByTreatment.get('nonfoil')!;
		expect(nonfoilCards).toHaveLength(4);
		expect(nonfoilCards.map((c) => c.name)).toEqual([
			'Lightning Bolt',
			'Black Lotus',
			'Ancestral Recall',
			'Time Walk'
		]);

		// Check foil group (Cards 1, 3, 4)
		const foilCards = result.cardsByTreatment.get('foil')!;
		expect(foilCards).toHaveLength(3);
		expect(foilCards.map((c) => c.name)).toEqual([
			'Lightning Bolt',
			'Mox Ruby',
			'Ancestral Recall'
		]);

		// Check etched group (Card 4)
		const etchedCards = result.cardsByTreatment.get('etched')!;
		expect(etchedCards).toHaveLength(1);
		expect(etchedCards[0].name).toBe('Ancestral Recall');
	});
});
