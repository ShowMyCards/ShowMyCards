import { BACKEND_URL, type StorageLocation, type SearchResponse } from '$lib';
import { handleAddInventory } from '$lib/server/inventory-actions';
import type { Actions, PageServerLoad } from './$types';
import { fail } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/storage`);
		if (response.ok) {
			const data = await response.json();
			return {
				storageLocations: data.data as StorageLocation[]
			};
		}
	} catch {
		// Ignore errors, dropdown will just be empty
	}
	return {
		storageLocations: [] as StorageLocation[]
	};
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
			const url = new URL(`${BACKEND_URL}/search`);
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

	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch)
} satisfies Actions;
