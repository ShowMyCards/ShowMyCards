/**
 * Utility functions for sorting card data
 * Centralized sorting logic to ensure consistency across the application
 */

/**
 * Interface for objects that have set information
 */
export interface SetSortable {
	set_name?: string | null;
	collector_number?: string | null;
}

/**
 * Sorts cards by set name (alphabetically), then by collector number (numerically)
 * This is the standard sort order used throughout the application
 *
 * @param a First card to compare
 * @param b Second card to compare
 * @returns Negative if a < b, positive if a > b, 0 if equal
 */
export function sortBySetAndCollectorNumber<T extends SetSortable>(a: T, b: T): number {
	const setA = a.set_name || '';
	const setB = b.set_name || '';

	// First, sort by set name alphabetically
	if (setA !== setB) {
		return setA.localeCompare(setB);
	}

	// Then, sort by collector number numerically
	const numA = parseCollectorNumber(a.collector_number);
	const numB = parseCollectorNumber(b.collector_number);

	return numA - numB;
}

/**
 * Parses a collector number into a numeric value for sorting
 * Handles cases like "123", "123a", "★123", etc.
 *
 * @param collectorNumber The collector number string
 * @returns A numeric value for sorting
 */
function parseCollectorNumber(collectorNumber?: string | null): number {
	if (!collectorNumber) return 0;

	// Extract leading digits from the collector number
	const match = collectorNumber.match(/^\D*(\d+)/);
	if (match) {
		return parseInt(match[1], 10);
	}

	// If no digits found, return 0 (will sort to beginning)
	return 0;
}

/**
 * Creates a comparator function that sorts by set and collector number
 * Useful for array.sort() calls
 *
 * @example
 * const sorted = cards.sort(createSetCollectorComparator());
 */
export function createSetCollectorComparator<T extends SetSortable>() {
	return (a: T, b: T) => sortBySetAndCollectorNumber(a, b);
}

/**
 * Sorts an array of cards by set and collector number (returns new array)
 *
 * @param cards Array of cards to sort
 * @returns New sorted array
 */
export function sortCardsBySetAndCollector<T extends SetSortable>(cards: T[]): T[] {
	return [...cards].sort(sortBySetAndCollectorNumber);
}
