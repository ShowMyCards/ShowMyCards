/**
 * Grouping and sorting helpers for the deck detail views.
 *
 * All inputs come from the local card database (type line, mana value), so this is
 * pure client-side computation over the already-loaded deck items.
 */

export type DeckGroupMode = 'none' | 'type' | 'manaValue';
export type DeckSortMode = 'name' | 'manaValue';

/** Minimal shape needed to group/sort a deck item. */
export interface GroupableCard {
	name?: string;
	cmc?: number;
	type_line?: string;
}

// Placement priority: a card falls into the first bucket whose type it contains.
// Creature beats everything (Artifact Creature → Creatures); Land beats the
// remaining non-creature permanents (Enchantment Land → Lands).
const TYPE_PLACEMENT: { bucket: string; match: string }[] = [
	{ bucket: 'Creatures', match: 'Creature' },
	{ bucket: 'Lands', match: 'Land' },
	{ bucket: 'Planeswalkers', match: 'Planeswalker' },
	{ bucket: 'Battles', match: 'Battle' },
	{ bucket: 'Instants', match: 'Instant' },
	{ bucket: 'Sorceries', match: 'Sorcery' },
	{ bucket: 'Artifacts', match: 'Artifact' },
	{ bucket: 'Enchantments', match: 'Enchantment' }
];

// The order the type buckets are displayed in (Lands near the end, à la Moxfield).
export const TYPE_DISPLAY_ORDER = [
	'Creatures',
	'Planeswalkers',
	'Battles',
	'Instants',
	'Sorceries',
	'Artifacts',
	'Enchantments',
	'Lands',
	'Other'
];

// Mana-value buckets: 0–6, then 7+, with lands pulled into their own group.
export const MANA_VALUE_DISPLAY_ORDER = ['0', '1', '2', '3', '4', '5', '6', '7+', 'Lands'];

/** Return the type bucket for a card by Moxfield-style placement priority. */
export function typeBucket(typeLine: string | undefined): string {
	const line = typeLine ?? '';
	for (const { bucket, match } of TYPE_PLACEMENT) {
		if (line.includes(match)) return bucket;
	}
	return 'Other';
}

/** Return the mana-value bucket for a card (lands are grouped separately). */
export function manaValueBucket(card: GroupableCard): string {
	if ((card.type_line ?? '').includes('Land')) return 'Lands';
	const cmc = Math.floor(card.cmc ?? 0);
	return cmc >= 7 ? '7+' : String(cmc);
}

/** Compare two cards for the given sort mode; name is always the tiebreaker. */
export function compareCards(a: GroupableCard, b: GroupableCard, sort: DeckSortMode): number {
	if (sort === 'manaValue') {
		const delta = (a.cmc ?? 0) - (b.cmc ?? 0);
		if (delta !== 0) return delta;
	}
	return (a.name ?? '').localeCompare(b.name ?? '');
}

export interface CardGroup<T> {
	label: string;
	items: T[];
}

/**
 * Group and sort a list of cards. With group 'none' a single unlabelled group is
 * returned. Groups are emitted in their canonical display order; items within each
 * group are sorted by the chosen sort mode.
 */
export function groupCards<T extends GroupableCard>(
	cards: T[],
	group: DeckGroupMode,
	sort: DeckSortMode
): CardGroup<T>[] {
	const sorted = [...cards].sort((a, b) => compareCards(a, b, sort));

	if (group === 'none') {
		return sorted.length > 0 ? [{ label: '', items: sorted }] : [];
	}

	const bucketOf =
		group === 'type' ? (c: T) => typeBucket(c.type_line) : (c: T) => manaValueBucket(c);
	const order = group === 'type' ? TYPE_DISPLAY_ORDER : MANA_VALUE_DISPLAY_ORDER;

	const buckets = new Map<string, T[]>();
	for (const card of sorted) {
		const label = bucketOf(card);
		const list = buckets.get(label);
		if (list) {
			list.push(card);
		} else {
			buckets.set(label, [card]);
		}
	}

	const result: CardGroup<T>[] = [];
	for (const label of order) {
		const items = buckets.get(label);
		if (items && items.length > 0) {
			result.push({ label, items });
		}
	}
	// Safety net for any label outside the canonical order.
	for (const [label, items] of buckets) {
		if (!order.includes(label) && items.length > 0) {
			result.push({ label, items });
		}
	}
	return result;
}
