import { BACKEND_URL, type SearchResponse, type DeckSummary } from '$lib';
import { handleAddInventory } from '$lib/server/inventory-actions';
import { loadStorageLocations } from '$lib/server/storage';
import { upsertDeckItems } from '$lib/server/deck-items';
import type { Actions, PageServerLoad } from './$types';
import { fail } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ fetch }) => {
	const storageLocations = await loadStorageLocations(fetch);

	// Decks are the alternate import destination. A failure here must not break
	// the (default) inventory import, so degrade to an empty list.
	let decks: DeckSummary[] = [];
	try {
		const response = await fetch(`${BACKEND_URL}/api/decks`);
		if (response.ok) {
			decks = await response.json();
		}
	} catch {
		decks = [];
	}

	return { storageLocations, decks };
};

export const actions = {
	// Search for a single card by name (and optionally set)
	searchCard: async ({ request, fetch }) => {
		const formData = await request.formData();
		const query = formData.get('query');

		if (!query || typeof query !== 'string') {
			return fail(400, { error: 'Search query is required' });
		}

		try {
			const url = new URL(`${BACKEND_URL}/api/search`);
			url.searchParams.set('q', query);

			const response = await fetch(url.toString());

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, {
					error: errorData.error || `Search failed: ${response.statusText}`
				});
			}

			const data: SearchResponse = await response.json();

			return {
				success: true,
				data
			};
		} catch (error) {
			return fail(500, {
				error: error instanceof Error ? error.message : 'Search failed'
			});
		}
	},

	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),

	// Resolve a batch of decklist lines against the local card DB (bulk-ingested
	// Scryfall data). Pinned lines match on set+collector, name-only on exact name;
	// unmatched lines come back Found=false so the client falls back to Scryfall.
	resolveLocal: async ({ request, fetch }) => {
		const formData = await request.formData();
		const itemsJson = formData.get('items');
		const language = (formData.get('language') as string) || '';

		if (!itemsJson || typeof itemsJson !== 'string') {
			return fail(400, { error: 'No items provided' });
		}

		try {
			const items = JSON.parse(itemsJson);
			const response = await fetch(`${BACKEND_URL}/api/cards/resolve`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ items, language })
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to resolve cards' });
			}

			const data = await response.json();
			return { success: true, action: 'resolveLocal', data };
		} catch (error) {
			return fail(500, {
				error: error instanceof Error ? error.message : 'Failed to resolve cards'
			});
		}
	},

	// Create a new deck inline (the import page's "create new deck" destination).
	createDeck: async ({ request, fetch }) => {
		const formData = await request.formData();
		const name = formData.get('name');
		const description = (formData.get('description') as string) || '';

		if (!name || typeof name !== 'string') {
			return fail(400, { error: 'Deck name is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name, description })
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to create deck' });
			}

			const deck = await response.json();
			return { success: true, action: 'createDeck', deck };
		} catch (error) {
			return fail(500, {
				error: error instanceof Error ? error.message : 'Failed to create deck'
			});
		}
	},

	// Add a batch of resolved cards to a deck (upsert on re-import).
	addDeckItems: async ({ request, fetch }) => {
		const formData = await request.formData();
		const deckId = formData.get('deck_id');
		const itemsJson = formData.get('items');

		if (!deckId || typeof deckId !== 'string') {
			return fail(400, { error: 'Deck is required' });
		}
		if (!itemsJson || typeof itemsJson !== 'string') {
			return fail(400, { error: 'No items provided' });
		}

		try {
			const items = JSON.parse(itemsJson);
			const result = await upsertDeckItems(fetch, deckId, items);
			if (!result.ok) {
				return fail(result.status, { error: result.error });
			}
			return { success: true, action: 'addDeckItems' };
		} catch (error) {
			return fail(500, {
				error: error instanceof Error ? error.message : 'Failed to add cards to deck'
			});
		}
	}
} satisfies Actions;
