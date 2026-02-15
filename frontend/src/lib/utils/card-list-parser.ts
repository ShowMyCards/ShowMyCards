/**
 * Card list parser for bulk import using Scryfall queries
 *
 * Format: [quantity][treatment] [scryfall query]
 * - quantity: number (default 1)
 * - treatment: none (nonfoil/first available), ! (foil), !! (etched)
 *
 * Examples:
 * - "4 e:who cn:1056" - 4 copies, nonfoil or first available
 * - "4! e:who cn:1056" - 4 copies, foil treatment
 * - "4!! e:who cn:1056" - 4 copies, etched treatment
 * - "lightning bolt" - 1 copy, nonfoil or first available
 */

export type TreatmentPreference = 'nonfoil' | 'foil' | 'etched';

export interface ParsedCard {
	/** Original line from input */
	line: string;
	/** Line number (1-based) */
	lineNumber: number;
	/** Desired quantity */
	quantity: number;
	/** Treatment preference */
	treatment: TreatmentPreference;
	/** Scryfall search query */
	query: string;
	/** Parse error if line couldn't be parsed */
	error?: string;
}

export interface ParseResult {
	/** Successfully parsed cards */
	cards: ParsedCard[];
	/** Lines that couldn't be parsed */
	errors: ParsedCard[];
	/** Total line count */
	totalLines: number;
}

// Pattern: optional quantity, optional treatment markers (! or !!), then the query
// Examples: "4!! e:who", "3! bolt", "2 counterspell", "sol ring"
const LINE_PATTERN = /^(\d+)?(!!?)?(?:\s+)?(.+)$/;

/**
 * Parse a single line of card list input
 */
function parseLine(line: string, lineNumber: number): ParsedCard {
	const trimmed = line.trim();

	// Skip empty lines and comments
	if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('#')) {
		return {
			line,
			lineNumber,
			quantity: 0,
			treatment: 'nonfoil',
			query: '',
			error: 'Empty or comment line'
		};
	}

	const match = trimmed.match(LINE_PATTERN);
	if (!match) {
		return {
			line,
			lineNumber,
			quantity: 0,
			treatment: 'nonfoil',
			query: '',
			error: 'Could not parse line'
		};
	}

	const [, quantityStr, treatmentMarker, query] = match;

	// Parse quantity (default to 1)
	const quantity = quantityStr ? parseInt(quantityStr) : 1;

	// Parse treatment preference
	let treatment: TreatmentPreference = 'nonfoil';
	if (treatmentMarker === '!!') {
		treatment = 'etched';
	} else if (treatmentMarker === '!') {
		treatment = 'foil';
	}

	// Validate we have a query
	if (!query || !query.trim()) {
		return {
			line,
			lineNumber,
			quantity,
			treatment,
			query: '',
			error: 'No search query provided'
		};
	}

	return {
		line,
		lineNumber,
		quantity,
		treatment,
		query: query.trim()
	};
}

/**
 * Parse a card list from text input
 *
 * @param input - Multi-line string containing card list
 * @returns Parsed cards and any errors
 */
export function parseCardList(input: string): ParseResult {
	const lines = input.split(/\r?\n/);
	const cards: ParsedCard[] = [];
	const errors: ParsedCard[] = [];

	for (let i = 0; i < lines.length; i++) {
		const parsed = parseLine(lines[i], i + 1);

		if (parsed.error) {
			// Only track actual errors, not empty lines
			if (parsed.line.trim() && !parsed.error.includes('Empty')) {
				errors.push(parsed);
			}
		} else if (parsed.query) {
			cards.push(parsed);
		}
	}

	return {
		cards,
		errors,
		totalLines: lines.length
	};
}

/**
 * Map treatment preference to actual finish string
 * Returns the preferred treatment if available, otherwise first available
 *
 * @param preference - Requested treatment preference
 * @param availableFinishes - Finishes available on the card
 * @returns The treatment to use, or null if preference is required but unavailable
 */
export function resolveTreatment(
	preference: TreatmentPreference,
	availableFinishes: string[]
): string | null {
	// Normalize available finishes
	const finishes = availableFinishes.length > 0 ? availableFinishes : ['nonfoil'];

	if (preference === 'nonfoil') {
		// For nonfoil preference, use nonfoil if available, otherwise first available
		if (finishes.includes('nonfoil')) {
			return 'nonfoil';
		}
		return finishes[0];
	}

	if (preference === 'foil') {
		// For foil, check for any foil variant
		const foilFinish = finishes.find((f) => f.includes('foil') && f !== 'nonfoil');
		if (foilFinish) {
			return foilFinish;
		}
		// Foil not available
		return null;
	}

	if (preference === 'etched') {
		// For etched, must have etched finish
		if (finishes.includes('etched')) {
			return 'etched';
		}
		// Etched not available
		return null;
	}

	return finishes[0];
}

/**
 * Get display name for treatment preference
 */
export function getTreatmentDisplayName(preference: TreatmentPreference): string {
	switch (preference) {
		case 'foil':
			return 'Foil';
		case 'etched':
			return 'Etched';
		default:
			return 'Regular';
	}
}

/**
 * Get the treatment marker for display
 */
export function getTreatmentMarker(preference: TreatmentPreference): string {
	switch (preference) {
		case 'foil':
			return '!';
		case 'etched':
			return '!!';
		default:
			return '';
	}
}
