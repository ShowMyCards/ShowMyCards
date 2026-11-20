/**
 * Card list export in Cardmarket's Wants-list syntax:
 *
 *   <quantity> <Card Name> [(<Expansion name>)]
 *
 * e.g. "2 Dark Confidant (Modern Masters)". Pinned printings carry their Scryfall
 * set name as the expansion. Cardmarket pins the want to that expansion when it
 * knows the name, and quietly falls back to any expansion when it doesn't (some
 * sets are named differently there, e.g. Scryfall's "Limited Edition Alpha").
 * Cards that may be any printing are written by name only.
 *
 * Deliberately left out, per paste tests against Cardmarket: set codes and
 * collector numbers ("(M10) 146" fails the line), finish markers ("*F*" is
 * ignored), zone headers (reported as unmatched) and Cardmarket's "(V.1)" version
 * index (not derivable from Scryfall data). Names are Scryfall's English names, with
 * double-faced and split cards written in full ("Fire // Ice"), which Cardmarket
 * matches; a front-face-only name does not.
 */
import type { DeckItemsResponse, EnrichedDeckItem, EnrichedListItem } from '../types/api';

/** 'all' exports every wanted copy; 'missing' only the copies not yet owned/collected. */
export type CardListExportMode = 'all' | 'missing';

export interface CardListExportLine {
	quantity: number;
	name: string;
	/** Expansion (set) name of a pinned printing; absent when any printing will do. */
	expansion?: string;
}

export interface CardListExport {
	/**
	 * One line per card and expansion, sorted by name (name-only line first, then
	 * expansions A→Z). Lines with nothing to export are omitted.
	 */
	lines: CardListExportLine[];
	/** Rows left out because their card data (name) could not be loaded. */
	skipped: number;
}

export type DeckExportItems = Pick<
	DeckItemsResponse,
	'command' | 'main' | 'side' | 'maybe' | 'other'
>;

function hasName<T extends { name?: string }>(row: T): row is T & { name: string } {
	return !!row.name;
}

/** Sums quantities per (name, expansion) line. */
function createLines() {
	const lines = new Map<string, CardListExportLine>();

	return {
		add(name: string, expansion: string | undefined, quantity: number) {
			if (quantity <= 0) return;
			const key = `${name}\u0000${expansion ?? ''}`;
			const line = lines.get(key);
			if (line) {
				line.quantity += quantity;
			} else {
				lines.set(key, expansion ? { quantity, name, expansion } : { quantity, name });
			}
		},
		sorted(): CardListExportLine[] {
			return [...lines.values()].sort(
				(a, b) =>
					a.name.localeCompare(b.name) || (a.expansion ?? '').localeCompare(b.expansion ?? '')
			);
		}
	};
}

/**
 * Only a pinned row's set says which printing is wanted. Any-printing rows still
 * carry a representative printing's set data, which must not leak into the export.
 */
function deckExpansion(row: EnrichedDeckItem): string | undefined {
	return row.scryfall_id ? row.set_name || undefined : undefined;
}

/**
 * Export a deck across all five zones (Command, Main, Side, Maybe, and Other
 * for tokens and other extras).
 *
 * 'missing' spends each card's owned copies (an Oracle-level, finish-agnostic
 * count) on its rows and exports what is left. Owned copies don't record their
 * printing, so they cover any-printing rows first and pinned rows after (each in
 * zone order), keeping specifically requested printings on the buy list. In total
 * that is max(0, Σ desired − owned) per card. Copies other decks also claim are
 * not considered, matching the deck's shortfall banner (which, unlike this
 * export, excludes the maybe-board).
 */
export function deckExportLines(items: DeckExportItems, mode: CardListExportMode): CardListExport {
	const rows = [...items.command, ...items.main, ...items.side, ...items.maybe, ...items.other];
	const named = rows.filter(hasName);
	const lines = createLines();

	if (mode === 'all') {
		for (const row of named) lines.add(row.name, deckExpansion(row), row.desired_quantity);
		return { lines: lines.sorted(), skipped: rows.length - named.length };
	}

	const byOracle = new Map<string, typeof named>();
	for (const row of named) {
		const group = byOracle.get(row.oracle_id);
		if (group) group.push(row);
		else byOracle.set(row.oracle_id, [row]);
	}

	for (const group of byOracle.values()) {
		let owned = Math.max(0, group[0].owned);
		const anyPrintingFirst = [
			...group.filter((row) => !row.scryfall_id),
			...group.filter((row) => row.scryfall_id)
		];
		for (const row of anyPrintingFirst) {
			const covered = Math.min(owned, row.desired_quantity);
			owned -= covered;
			lines.add(row.name, deckExpansion(row), row.desired_quantity - covered);
		}
	}

	return { lines: lines.sorted(), skipped: rows.length - named.length };
}

/**
 * Export a list. Every list item is a specific printing, so each line names its
 * expansion. 'missing' is desired − collected, clamped per item before merging, so
 * an over-collected foil doesn't cancel out a missing nonfoil.
 */
export function listExportLines(
	items: EnrichedListItem[],
	mode: CardListExportMode
): CardListExport {
	const named = items.filter(hasName);
	const lines = createLines();

	for (const item of named) {
		const quantity =
			mode === 'all'
				? item.desired_quantity
				: Math.max(0, item.desired_quantity - item.collected_quantity);
		lines.add(item.name, item.set_name || undefined, quantity);
	}

	return { lines: lines.sorted(), skipped: items.length - named.length };
}

/** One option of the export dialog's switch, e.g. "Only missing" or "All cards". */
export interface CardListExportChoice {
	label: string;
	export: CardListExport;
	/** Shown when the choice has no lines. Defaults to "No cards to export." */
	empty?: string;
}

/** Render export lines as newline-separated "<quantity> <name> [(<expansion>)]" text. */
export function formatExportLines(lines: CardListExportLine[]): string {
	return lines
		.map(({ quantity, name, expansion }) =>
			expansion ? `${quantity} ${name} (${expansion})` : `${quantity} ${name}`
		)
		.join('\n');
}
