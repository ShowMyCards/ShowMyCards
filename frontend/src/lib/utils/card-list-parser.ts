/**
 * Card list parser for bulk import.
 *
 * Supports two overlapping input styles on a single line:
 *
 * 1. Scryfall search syntax (power users, inventory import):
 *      [quantity][treatment] [scryfall query]
 *      - treatment marker is LEADING: none = nonfoil, ! = foil, !! = etched
 *      - e.g. "4 e:who cn:1056", "2! !\"Lightning Bolt\"", "sol ring"
 *
 * 2. Decklist-export lines (Moxfield / MTGGoldfish / MTGA / MTGO / ManaBox):
 *      [quantity] Name [(SET) [COLLECTOR]] [*F*|*E*]
 *      - treatment marker is TRAILING: *F* = foil, *E* = etched
 *      - the set may be in (round) or [square] brackets; they mean the same
 *      - "1 Ashnod's Altar"              -> any printing (name search)
 *      - "1 Ashnod's Altar (BRR) 67 *F*" -> pinned printing (e:BRR cn:67), foil
 *      - "Cat [tmh3]"                    -> pinned to the set (!"Cat" e:tmh3); Moxfield
 *                                           wishlists list tokens this way
 *
 * Collector numbers that aren't purely digits are quoted in the resolved query
 * (cn:"2XM-169", cn:"112p", cn:"65†"): Scryfall does not match them in an unquoted
 * cn:. Plain numbers stay unquoted. This applies to both input styles, so a typed
 * "e:plst cn:2XM-169" is rewritten too.
 *
 * Zones (deck import): section headers (Commander / Deck / Sideboard / Maybeboard
 * and common variants, plus an "About" metadata block that is skipped) switch the
 * active zone. Exports with no headers use blank-line-separated blocks: the first
 * block is Main, subsequent blocks are Side (matches MTGGoldfish's main/sideboard
 * split; a trailing commander block, e.g. Moxfield's MTGO export, lands in Side and
 * can be re-zoned in the deck UI). Inventory import ignores zones.
 */

export type TreatmentPreference = 'nonfoil' | 'foil' | 'etched';

/** Where a parsed line belongs within a deck. Assignable to the DeckZone model type. */
export type ImportZone = 'command' | 'main' | 'side' | 'maybe';

export interface ParsedCard {
	/** Original line from input */
	line: string;
	/** Line number (1-based) */
	lineNumber: number;
	/** Desired quantity */
	quantity: number;
	/** Treatment preference */
	treatment: TreatmentPreference;
	/** Scryfall search query used to resolve the line */
	query: string;
	/** Deck zone this line belongs to (defaults to 'main'; only meaningful for deck import) */
	zone: ImportZone;
	/** Parsed card name, when the line is a decklist entry (not a raw Scryfall query) */
	name?: string;
	/** Set code, when the line pins a printing */
	set?: string;
	/** Collector number, when the line pins a printing and gives one */
	collectorNumber?: string;
	/**
	 * True when the line names a printing (a set, with or without a collector
	 * number). Such lines resolve to a pinned Scryfall ID; non-pinned lines keep
	 * only the Oracle ID.
	 */
	pinned: boolean;
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

export interface ParseOptions {
	/** Scryfall language code (e.g. "en", "de") appended as l:<code>, or "" to skip. */
	language?: string;
}

// Leading quantity + optional LEADING treatment marker (Scryfall-syntax style),
// then the remainder. Examples: "4!! e:who", "3! bolt", "2 counterspell", "sol ring".
const LEADING_PATTERN = /^(\d+)?\s*(!!?)?\s*(.+)$/;

// A set code in matching round or square brackets: "(MUL)" or "[tmh3]". Moxfield
// uses both (decklists round, wishlists square). Two capture groups: one per
// bracket style, exactly one of which is set.
const SET_GROUP = String.raw`(?:\(([0-9A-Za-z]+)\)|\[([0-9A-Za-z]+)\])`;

// A decklist line pinning a specific printing: "Name (SET) COLLECTOR [*F*|*E*]".
// The set code is alphanumeric only and the line is anchored end-to-end (bar the
// finish marker), so raw Scryfall queries that contain parentheses — e.g.
// "(o:draw or o:scry) e:dom" — cannot match and pass through as queries. The name
// group is greedy so a name that itself contains parentheses binds the set to the
// LAST "(XXX)" group (e.g. "Erase (Not the One) (WTH) 20").
//
// The collector number is any run of characters except whitespace and those that
// would let it swallow Scryfall syntax or the finish marker: ":" (so "e:dom" isn't
// a number), '"' (keeps quoting intact), "(" ")" (Scryfall grouping) and "*" (the
// *F* / *E* marker). This admits real numbers such as "65†", "2XM-169" and "118★".
// The lookahead requires at least one letter or digit, so a lone "-" isn't a number.
const PINNED_PATTERN = new RegExp(
	String.raw`^(.+)\s+${SET_GROUP}\s+((?=[^\s:"()*]*[\p{L}\p{N}])[^\s:"()*]+)(?:\s+\*([FE])\*)?$`,
	'u'
);

// A decklist line naming a set but no collector number: "Name (SET)" or "Name [SET]"
// (e.g. Moxfield wishlist tokens, "Cat [tmh3]"). Same anchoring and greedy name as
// PINNED_PATTERN. The name must not contain ':' or '"', so a raw query ending in a
// bracketed word (e.g. "t:goblin (red)") is still treated as a query.
const SET_ONLY_PATTERN = new RegExp(String.raw`^([^:"]+)\s+${SET_GROUP}(?:\s+\*([FE])\*)?$`, 'u');

function finishTreatment(marker: string | undefined): TreatmentPreference {
	return marker === 'F' ? 'foil' : marker === 'E' ? 'etched' : 'nonfoil';
}

// An unquoted cn:/number: clause in a Scryfall query (":" or "=" only; range
// comparisons like cn>=100 are numeric and left alone). The value stops at
// whitespace, quotes and parentheses so grouped queries keep their syntax. The
// first alternative consumes quoted strings whole so text inside them is untouched.
const CN_CLAUSE_PATTERN = /"[^"]*"|(^|[\s(])(-?)(cn|number)([:=])([^\s"()]+)/gi;

/** Build a cn: clause, quoting the collector number unless it is purely digits. */
function cnClause(collector: string): string {
	return /^\d+$/.test(collector) ? `cn:${collector}` : `cn:"${collector}"`;
}

/** Quote any non-numeric unquoted cn:/number: value in a Scryfall query. */
function quoteCollectorNumbers(query: string): string {
	return query.replace(CN_CLAUSE_PATTERN, (clause, lead, neg, key, op, value?: string) =>
		value === undefined || /^\d+$/.test(value) ? clause : `${lead}${neg}${key}${op}"${value}"`
	);
}

// Existing Scryfall language clause (l: or lang:), mirroring the backend's
// languageClauseRegExp in backend/api/search.go so we never double-append.
const LANGUAGE_CLAUSE_PATTERN = /(^|\s)[-!]?(l|lang):/i;

// Section headers → zone, plus "about" for the metadata block MTGA exports emit.
const ZONE_HEADERS: Record<string, ImportZone | 'ignore'> = {
	about: 'ignore',
	commander: 'command',
	commanders: 'command',
	'command zone': 'command',
	deck: 'main',
	mainboard: 'main',
	'main deck': 'main',
	main: 'main',
	sideboard: 'side',
	side: 'side',
	maybeboard: 'maybe',
	maybe: 'maybe',
	considering: 'maybe',
	companion: 'side'
};

type LineKind =
	{ kind: 'card' } | { kind: 'header'; zone: ImportZone | 'ignore' } | { kind: 'skip' };

/**
 * Classify a non-empty, non-comment line as a card, a section header, or a
 * skippable non-card line (e.g. an Archidekt category header like "Creatures {30}").
 */
function classifyLine(trimmed: string): LineKind {
	// A leading quantity unambiguously marks a card line.
	if (/^\d/.test(trimmed)) {
		return { kind: 'card' };
	}

	// No leading quantity: it may be a section header. Strip a trailing count
	// (e.g. "Sideboard (15)", "Commander {1}") and a trailing colon before matching.
	const normalized = trimmed
		.replace(/[\s]*[{(]\d+[)}]\s*$/, '')
		.replace(/:\s*$/, '')
		.trim()
		.toLowerCase();

	if (normalized in ZONE_HEADERS) {
		return { kind: 'header', zone: ZONE_HEADERS[normalized] };
	}

	// A non-zone header carrying a count (e.g. Archidekt category "Lands {38}") is
	// structural noise — skip it without changing the active zone or emitting a card.
	if (/[{(]\d+[)}]\s*$/.test(trimmed)) {
		return { kind: 'skip' };
	}

	// Otherwise treat it as a name-only card with an implicit quantity of 1.
	return { kind: 'card' };
}

/** Append the language clause to a query unless one is already present. */
function withLanguage(query: string, language?: string): string {
	if (language && !LANGUAGE_CLAUSE_PATTERN.test(query)) {
		return `${query} l:${language}`;
	}
	return query;
}

/**
 * Parse a single card line into a ParsedCard. Assumes the line is not empty, a
 * comment, or a header (the caller handles those while tracking zone).
 */
function parseCardLine(
	line: string,
	lineNumber: number,
	zone: ImportZone,
	options: ParseOptions
): ParsedCard {
	const trimmed = line.trim();

	const match = trimmed.match(LEADING_PATTERN);
	if (!match) {
		return {
			line,
			lineNumber,
			quantity: 0,
			treatment: 'nonfoil',
			query: '',
			zone,
			pinned: false,
			error: 'Could not parse line'
		};
	}

	const [, quantityStr, leadingMarker, rest] = match;
	const quantity = quantityStr ? parseInt(quantityStr, 10) : 1;

	if (!rest || !rest.trim()) {
		return {
			line,
			lineNumber,
			quantity,
			treatment: 'nonfoil',
			query: '',
			zone,
			pinned: false,
			error: 'No card name or query provided'
		};
	}

	// A pinned decklist printing: "Name (SET) COLLECTOR [*F*|*E*]".
	const pinnedMatch = rest.match(PINNED_PATTERN);
	if (pinnedMatch) {
		const [, name, parenSet, bracketSet, collector, finishMarker] = pinnedMatch;
		const set = parenSet ?? bracketSet;

		return {
			line,
			lineNumber,
			quantity,
			treatment: finishTreatment(finishMarker),
			query: withLanguage(`e:${set} ${cnClause(collector)}`, options.language),
			zone,
			name: name.trim(),
			set,
			collectorNumber: collector,
			pinned: true
		};
	}

	// A set but no collector number: "Name (SET)" / "Name [SET]". Pinned to whichever
	// printing Scryfall returns for that exact name in that set. With no collector
	// number the local resolver can't pin it, so it always goes to Scryfall.
	const setOnlyMatch = rest.match(SET_ONLY_PATTERN);
	if (setOnlyMatch) {
		const [, rawName, parenSet, bracketSet, finishMarker] = setOnlyMatch;
		const name = rawName.trim();
		const set = parenSet ?? bracketSet;

		return {
			line,
			lineNumber,
			quantity,
			treatment: finishTreatment(finishMarker),
			query: withLanguage(`!"${name}" e:${set}`, options.language),
			zone,
			name,
			set,
			pinned: true
		};
	}

	// Otherwise a raw Scryfall query or a plain card name. Treatment (if any) comes
	// from the leading !/!! marker, matching the existing inventory-import syntax.
	const treatment: TreatmentPreference =
		leadingMarker === '!!' ? 'etched' : leadingMarker === '!' ? 'foil' : 'nonfoil';

	// Expose a display name only for plain names (no Scryfall operators present).
	const looksLikeQuery = /[:!"]/.test(rest);

	return {
		line,
		lineNumber,
		quantity,
		treatment,
		query: withLanguage(
			looksLikeQuery ? quoteCollectorNumbers(rest.trim()) : rest.trim(),
			options.language
		),
		zone,
		name: looksLikeQuery ? undefined : rest.trim(),
		pinned: false
	};
}

/**
 * Parse a card list from text input.
 *
 * @param input - Multi-line string containing a card list or decklist export
 * @param options - Optional language clause to append to every resolved query
 * @returns Parsed cards (each tagged with a zone) and any errors
 */
export function parseCardList(input: string, options: ParseOptions = {}): ParseResult {
	const lines = input.split(/\r?\n/);
	const cards: ParsedCard[] = [];
	const errors: ParsedCard[] = [];

	// Zone tracking. `hasHeaders` disables blank-line block inference once any
	// section header is seen (the file is explicitly structured). `ignore` skips
	// lines inside an "About" metadata block until the next header.
	let zone: ImportZone = 'main';
	let ignore = false;
	let hasHeaders = false;
	let blockIndex = 0;
	let sawContentInBlock = false;

	for (let i = 0; i < lines.length; i++) {
		const line = lines[i];
		const trimmed = line.trim();

		// Blank line: a block boundary for header-less block inference.
		if (!trimmed) {
			if (sawContentInBlock) {
				blockIndex++;
				sawContentInBlock = false;
			}
			continue;
		}

		// Comments.
		if (trimmed.startsWith('//') || trimmed.startsWith('#')) {
			continue;
		}

		const classified = classifyLine(trimmed);

		if (classified.kind === 'header') {
			hasHeaders = true;
			if (classified.zone === 'ignore') {
				ignore = true;
			} else {
				ignore = false;
				zone = classified.zone;
			}
			continue;
		}

		if (classified.kind === 'skip' || ignore) {
			continue;
		}

		// A card line. Header-driven zone when headers exist; otherwise the first
		// block is Main and later blocks are Side.
		sawContentInBlock = true;
		const effectiveZone: ImportZone = hasHeaders ? zone : blockIndex === 0 ? 'main' : 'side';

		const parsed = parseCardLine(line, i + 1, effectiveZone, options);

		if (parsed.error) {
			errors.push(parsed);
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
 * Map treatment preference to actual finish string.
 * Returns the preferred treatment if available, otherwise first available.
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
