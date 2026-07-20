/**
 * ManaBox text export support for bulk import.
 *
 * ManaBox exports cards in the format:
 *   1 Lightning Bolt (2ED) 161
 *   2 Counterspell (EMA) 43 *F*
 *   1 Sol Ring (LTC) 284 *E*
 *
 * The import page only understands Scryfall search syntax
 * (see card-list-parser.ts), so ManaBox lines are converted to the equivalent
 * Scryfall syntax before being parsed:
 *   1 Lightning Bolt (2ED) 161   -> 1 e:2ED cn:161
 *   2 Counterspell (EMA) 43 *F*  -> 2! e:EMA cn:43
 *   1 Sol Ring (LTC) 284 *E*     -> 1!! e:LTC cn:284
 *
 * A trailing `*F*` marks a foil printing (`!` marker); `*E*` marks etched
 * (`!!` marker). Both map to card-list-parser's treatment markers.
 */

// Captures: quantity, set code, collector number, optional finish marker.
//
// The pattern is deliberately strict so it does NOT swallow valid Scryfall
// queries that happen to contain parentheses (e.g. `1 (t:creature) e:dom` or
// `4 (o:draw or o:scry) e:dom`) — those must pass through untouched:
//   - the set code is alphanumeric only, so `(o:draw or o:scry)` cannot match;
//   - the `.*` before the set code is greedy, so a card name that itself
//     contains parentheses (e.g. `Erase (Not the One) (WTH) 20`) still binds
//     the set code to the LAST `(XXX)` group, not the first;
//   - the line is anchored end-to-end (allowing only a trailing finish
//     marker), so anything with extra Scryfall clauses fails and passes through.
const MANABOX_LINE_PATTERN =
	/^(\d+)\s+.*\(([0-9A-Za-z]+)\)\s+([0-9A-Za-z★-]+)(?:\s+\*([FE])\*)?\s*$/;

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

	const [, quantity, setCode, collectorNumber, finishMarker] = match;

	// *F* -> foil (!), *E* -> etched (!!), none -> nonfoil.
	const treatmentMarker = finishMarker === 'F' ? '!' : finishMarker === 'E' ? '!!' : '';

	return `${quantity}${treatmentMarker} e:${setCode} cn:${collectorNumber}`;
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
