import type { ScryfallCard } from '../types/runtime';

/**
 * Card with an associated treatment/finish
 */
export interface CardWithTreatment extends ScryfallCard {
	selectedTreatment: string;
}

/**
 * Result of grouping cards by treatment
 */
export interface GroupedCardsResult {
	/** Sorted array of unique treatments found across all cards */
	treatments: string[];
	/** Map of treatment -> cards that have that treatment */
	cardsByTreatment: Map<string, CardWithTreatment[]>;
}

/**
 * Get array of unique treatments (finishes) from a list of cards
 *
 * Extracts all unique finishes from the provided cards, defaulting to 'nonfoil'
 * for cards with no finishes array. Results are sorted alphabetically.
 *
 * @param cards - Array of Scryfall cards
 * @returns Sorted array of unique treatment names
 *
 * @example
 * ```typescript
 * const treatments = getUniqueTreatments([
 *   { finishes: ['nonfoil', 'foil'], ... },
 *   { finishes: ['etched'], ... }
 * ]);
 * // Returns: ['etched', 'foil', 'nonfoil']
 * ```
 */
export function getUniqueTreatments(cards: ScryfallCard[]): string[] {
	const treatments = new Set<string>();

	for (const card of cards) {
		// Default to ['nonfoil'] if no finishes array
		const finishes = card.finishes && card.finishes.length > 0 ? card.finishes : ['nonfoil'];
		for (const finish of finishes) {
			treatments.add(finish);
		}
	}

	return Array.from(treatments).sort();
}

/**
 * Group cards by their available treatments (finishes)
 *
 * Creates a map where each treatment maps to all cards that have that treatment.
 * Each card in the resulting arrays includes a `selectedTreatment` property indicating
 * which treatment it represents in that array.
 *
 * Cards with multiple finishes will appear in multiple treatment arrays.
 * Cards with no finishes array default to 'nonfoil'.
 *
 * @param cards - Array of Scryfall cards
 * @returns Object with sorted treatments array and map of treatment -> cards
 *
 * @example
 * ```typescript
 * const result = groupCardsByTreatment([
 *   { id: '1', name: 'Card A', finishes: ['nonfoil', 'foil'], ... },
 *   { id: '2', name: 'Card B', finishes: ['nonfoil'], ... }
 * ]);
 *
 * // result.treatments: ['foil', 'nonfoil']
 * // result.cardsByTreatment.get('nonfoil'): [
 * //   { id: '1', name: 'Card A', finishes: ['nonfoil', 'foil'], selectedTreatment: 'nonfoil', ... },
 * //   { id: '2', name: 'Card B', finishes: ['nonfoil'], selectedTreatment: 'nonfoil', ... }
 * // ]
 * // result.cardsByTreatment.get('foil'): [
 * //   { id: '1', name: 'Card A', finishes: ['nonfoil', 'foil'], selectedTreatment: 'foil', ... }
 * // ]
 * ```
 */
export function groupCardsByTreatment(cards: ScryfallCard[]): GroupedCardsResult {
	// Get unique treatments first
	const treatments = getUniqueTreatments(cards);

	// Initialize map with empty arrays for each treatment
	const cardsByTreatment = new Map<string, CardWithTreatment[]>();
	for (const treatment of treatments) {
		cardsByTreatment.set(treatment, []);
	}

	// Group cards by their treatments
	for (const card of cards) {
		// Default to ['nonfoil'] if no finishes array
		const finishes = card.finishes && card.finishes.length > 0 ? card.finishes : ['nonfoil'];

		for (const finish of finishes) {
			if (cardsByTreatment.has(finish)) {
				// Add card with selectedTreatment property
				cardsByTreatment.get(finish)!.push({
					...card,
					selectedTreatment: finish
				});
			}
		}
	}

	return {
		treatments,
		cardsByTreatment
	};
}
