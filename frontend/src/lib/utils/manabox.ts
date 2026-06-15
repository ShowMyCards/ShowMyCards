/**
 * ManaBox text export support for bulk import.
 *
 * ManaBox exports cards in the format:
 *   1 Lightning Bolt (2ED) 161
 *   2 Counterspell (EMA) 43 *F*
 *
 * The import page only understands Scryfall search syntax
 * (see card-list-parser.ts), so ManaBox lines are converted to the equivalent
 * Scryfall syntax before being parsed:
 *   1 Lightning Bolt (2ED) 161   -> 1 e:2ED cn:161
 *   2 Counterspell (EMA) 43 *F*  -> 2! e:EMA cn:43
 *
 * A trailing `*F*` marks a foil printing and maps to the `!` foil marker.
 */

// Captures: quantity, set code (inside parentheses), collector number.
const MANABOX_LINE_PATTERN = /^(\d+)\s+.*?\((.*?)\)\s+(\S+)/;

// Detects an existing Scryfall language clause (l: or lang:), mirroring the
// backend's languageClauseRegExp in backend/api/search.go so we never
// double-append a language.
const LANGUAGE_CLAUSE_PATTERN = /(^|\s)[-!]?(l|lang):/i;

/**
 * Convert a single ManaBox export line to Scryfall import syntax.
 *
 * @param line - A raw line from a ManaBox text export
 * @returns The equivalent Scryfall syntax line, or null if the line is not in
 *          ManaBox format (the caller should leave such lines untouched).
 */
export function convertManaboxLine(line: string): string | null {
	const trimmed = line.trim();
	const match = trimmed.match(MANABOX_LINE_PATTERN);
	if (!match) {
		return null;
	}

	const [, quantity, setCode, collectorNumber] = match;
	const isFoil = trimmed.endsWith('*F*');

	return `${quantity}${isFoil ? '!' : ''} e:${setCode} cn:${collectorNumber}`;
}

/**
 * Preprocess bulk-import text before it is fed into parseCardList().
 *
 * Each non-empty, non-comment line is converted from ManaBox format when it
 * matches (otherwise left as-is, i.e. treated as Scryfall syntax). The given
 * language code is then appended as an `l:<code>` clause unless the line already
 * specifies a language.
 *
 * @param input - The raw textarea contents
 * @param languageCode - Scryfall language code (e.g. "en", "de") or "" to skip
 * @returns The preprocessed multi-line string
 */
export function preprocessImportText(input: string, languageCode: string): string {
	const lines = input.split(/\r?\n/);

	return lines
		.map((line) => {
			const trimmed = line.trim();

			// Pass blank lines and comments through unchanged.
			if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('#')) {
				return line;
			}

			const converted = convertManaboxLine(line) ?? line;

			if (languageCode && !LANGUAGE_CLAUSE_PATTERN.test(converted)) {
				return `${converted} l:${languageCode}`;
			}

			return converted;
		})
		.join('\n');
}
