import { json } from '@sveltejs/kit';
import { upsertDeckItems } from '$lib/server/deck-items';
import type { RequestHandler } from './$types';

/**
 * Add cards to a deck from client components (card tiles, inventory bulk bar).
 *
 * Body: `{ items: CreateDeckItemRequest[] }`. Uses the shared upsert so a card
 * that is already in the deck has its quantity increased instead of colliding
 * with the deck-item unique index. The browser cannot reach the backend
 * directly, so this route runs the upsert server-side.
 */
export const POST: RequestHandler = async ({ params, request, fetch }) => {
	const body = await request.json().catch(() => null);
	const items = body?.items;

	if (!Array.isArray(items) || items.length === 0) {
		return json({ error: 'No items provided' }, { status: 400 });
	}

	try {
		const result = await upsertDeckItems(fetch, params.id, items);
		if (!result.ok) {
			return json({ error: result.error }, { status: result.status });
		}
		return json({ success: true });
	} catch {
		return json({ error: 'Failed to add cards to deck' }, { status: 500 });
	}
};
