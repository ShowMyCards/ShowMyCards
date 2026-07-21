import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail, redirect } from '@sveltejs/kit';
import type { Deck, DeckItemsResponse } from '$lib';
import { upsertDeckItems } from '$lib/server/deck-items';

const emptyItems: DeckItemsResponse = {
	deck_id: 0,
	command: [],
	main: [],
	side: [],
	maybe: [],
	aggregate_shortfall: 0
};

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { id } = params;

	try {
		// Load deck metadata and grouped items in parallel.
		const [deckResponse, itemsResponse] = await Promise.all([
			fetch(`${BACKEND_URL}/api/decks/${id}`),
			fetch(`${BACKEND_URL}/api/decks/${id}/items`)
		]);

		if (!deckResponse.ok) {
			return {
				deck: null,
				items: emptyItems,
				error: 'Failed to load deck'
			};
		}

		const deck: Deck = await deckResponse.json();

		if (!itemsResponse.ok) {
			return {
				deck,
				items: emptyItems,
				error: 'Failed to load deck items'
			};
		}

		const items: DeckItemsResponse = await itemsResponse.json();

		return {
			deck,
			items
		};
	} catch {
		return {
			deck: null,
			items: emptyItems,
			error: 'Failed to load deck'
		};
	}
};

export const actions: Actions = {
	// Search for cards to add (reuses the shared search endpoint).
	search: async ({ request, fetch }) => {
		const data = await request.formData();
		const query = data.get('q') as string;

		if (!query) {
			return fail(400, { error: 'Search query is required', searchResults: [] });
		}

		try {
			let page = 1;
			let hasMore = true;
			const allResults = [];

			while (hasMore) {
				const searchResponse = await fetch(
					`${BACKEND_URL}/api/search?q=${encodeURIComponent(query)}&page=${page}`
				);
				if (searchResponse.ok) {
					const searchData = await searchResponse.json();
					allResults.push(...(searchData.data || []));
					hasMore = searchData.has_more || false;
					page++;

					// Safety limit: don't fetch more than 10 pages.
					if (page > 10) {
						break;
					}
				} else {
					hasMore = false;
				}
			}

			return {
				success: true,
				action: 'search',
				searchResults: allResults,
				query
			};
		} catch {
			return fail(500, { error: 'Search failed', searchResults: [] });
		}
	},

	// Add cards to the deck (batch endpoint; the UI adds one card per submit).
	addItems: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemsJson = data.get('items') as string;

		if (!itemsJson) {
			return fail(400, { error: 'No items provided' });
		}

		try {
			const items = JSON.parse(itemsJson);

			// Upsert (increment matching rows, create the rest) — shared with the
			// bulk import flow. See upsertDeckItems for the rationale.
			const result = await upsertDeckItems(fetch, id, items);
			if (!result.ok) {
				return fail(result.status, { error: result.error });
			}

			return { success: true, action: 'add' };
		} catch {
			return fail(500, { error: 'Failed to add card' });
		}
	},

	// Update a deck item's desired quantity and/or zone.
	updateItem: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemId = data.get('item_id') as string;
		const desiredQuantity = data.get('desired_quantity') as string | null;
		const zone = data.get('zone') as string | null;

		if (!itemId) {
			return fail(400, { error: 'Item ID is required' });
		}

		const body: { desired_quantity?: number; zone?: string } = {};
		if (desiredQuantity !== null && desiredQuantity !== '') {
			body.desired_quantity = parseInt(desiredQuantity, 10);
		}
		if (zone !== null && zone !== '') {
			body.zone = zone;
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}/items/${itemId}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(body)
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				const message =
					response.status === 409
						? 'That change collides with an existing card in the target zone'
						: errorData.error || 'Failed to update item';
				return fail(response.status, { error: message });
			}

			return { success: true, action: 'update' };
		} catch {
			return fail(500, { error: 'Failed to update item' });
		}
	},

	// Remove a card from the deck.
	deleteItem: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemId = data.get('item_id') as string;

		if (!itemId) {
			return fail(400, { error: 'Item ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}/items/${itemId}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to remove item' });
			}

			return { success: true, action: 'delete' };
		} catch {
			return fail(500, { error: 'Failed to remove item' });
		}
	},

	// Rename / re-describe the deck itself.
	updateDeck: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const name = data.get('name') as string;
		const description = data.get('description') as string;

		if (!name) {
			return fail(400, { error: 'Name is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ name, description })
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to update deck' });
			}

			return { success: true, action: 'updateDeck' };
		} catch {
			return fail(500, { error: 'Failed to update deck' });
		}
	},

	// Delete the deck and return to the deck list.
	deleteDeck: async ({ params, fetch }) => {
		const { id } = params;

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to delete deck' });
			}
		} catch {
			return fail(500, { error: 'Failed to delete deck' });
		}

		return redirect(303, '/decks');
	}
};
