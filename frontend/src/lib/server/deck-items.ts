import { BACKEND_URL } from '$lib';

/** A single deck item to add, matching the backend CreateDeckItemRequest shape. */
export interface DeckItemInput {
	oracle_id: string;
	scryfall_id: string;
	treatment: string;
	desired_quantity: number;
	zone: string;
}

type FetchFn = typeof fetch;

export type UpsertResult = { ok: true } | { ok: false; status: number; error: string };

/**
 * Add items to a deck with upsert semantics.
 *
 * A deck item is unique on (deck_id, oracle_id, scryfall_id, treatment, zone), so
 * re-adding an identical card would collide on that index (the backend batch-create
 * returns a raw 500 on such a conflict). Instead we fetch the deck's current items,
 * increment the desired_quantity of any that already match, and batch-create only
 * the genuinely new rows. This delivers the FR98 "append/upsert on re-import"
 * behaviour and is shared by the deck detail page and the bulk import flow.
 */
export async function upsertDeckItems(
	fetch: FetchFn,
	deckId: string | number,
	items: DeckItemInput[]
): Promise<UpsertResult> {
	if (items.length === 0) {
		return { ok: true };
	}

	const existingRes = await fetch(`${BACKEND_URL}/api/decks/${deckId}/items`);
	const existing = existingRes.ok
		? await existingRes.json()
		: { command: [], main: [], side: [], maybe: [] };
	const allExisting: Array<DeckItemInput & { id: number }> = [
		...(existing.command ?? []),
		...(existing.main ?? []),
		...(existing.side ?? []),
		...(existing.maybe ?? [])
	];

	const toCreate: DeckItemInput[] = [];
	for (const item of items) {
		const match = allExisting.find(
			(e) =>
				e.oracle_id === item.oracle_id &&
				e.scryfall_id === item.scryfall_id &&
				e.treatment === item.treatment &&
				e.zone === item.zone
		);

		if (match) {
			const putRes = await fetch(`${BACKEND_URL}/api/decks/${deckId}/items/${match.id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ desired_quantity: match.desired_quantity + item.desired_quantity })
			});
			if (!putRes.ok) {
				const errorData = await putRes.json().catch(() => ({}));
				return { ok: false, status: putRes.status, error: errorData.error || 'Failed to add card' };
			}
		} else {
			toCreate.push(item);
		}
	}

	if (toCreate.length > 0) {
		const res = await fetch(`${BACKEND_URL}/api/decks/${deckId}/items/batch`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ items: toCreate })
		});
		if (!res.ok) {
			const errorData = await res.json().catch(() => ({}));
			const error =
				res.status === 409
					? 'Some cards are already in this deck and zone'
					: errorData.error || 'Failed to add cards';
			return { ok: false, status: res.status, error };
		}
	}

	return { ok: true };
}
