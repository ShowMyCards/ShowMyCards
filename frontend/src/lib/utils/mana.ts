/**
 * Convert a Scryfall mana-cost string into Mana-font icon class lists.
 *
 * Each returned string is the class list for a single symbol, e.g. "ms ms-2 ms-cost"
 * (the circular "cost" pip variant). Handles generic/coloured symbols, hybrids
 * ({U/R} → ms-ur), phyrexian ({G/P} → ms-gp), twobrid ({2/W} → ms-2w), and the
 * special {½} / {∞} glyphs.
 *
 * Mana font by Andrew Gioia — https://mana.andrewgioia.com (SIL OFL 1.1).
 */
export function manaSymbolClasses(manaCost: string | undefined | null): string[] {
	if (!manaCost) return [];

	const classes: string[] = [];
	const re = /\{([^}]+)\}/g;
	let match: RegExpExecArray | null;
	while ((match = re.exec(manaCost)) !== null) {
		classes.push(`ms ms-${manaSymbolCode(match[1])} ms-cost`);
	}
	return classes;
}

function manaSymbolCode(token: string): string {
	if (token === '½') return 'half';
	if (token === '∞') return 'infinity';
	// Hybrid / phyrexian / twobrid: "U/R" → "ur", "G/P" → "gp", "2/W" → "2w".
	return token.toLowerCase().replace(/\//g, '');
}
