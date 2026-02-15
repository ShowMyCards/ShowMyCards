import { BACKEND_URL, type SearchResponse, type StorageLocation } from '$lib';
import { handleAddInventory, handleDeleteInventory } from '$lib/server/inventory-actions';
import type { Actions, PageServerLoad } from './$types';
import { fail } from '@sveltejs/kit';

// Load storage locations for the dropdown
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
	// Search for cards
	search: async ({ request, fetch }) => {
		const formData = await request.formData();
		const query = formData.get('q');

		if (!query || typeof query !== 'string') {
			return fail(400, { error: 'Search query is required' });
		}

		try {
			const url = new URL(`${BACKEND_URL}/search`);
			url.searchParams.set('q', query);

			const response = await fetch(url.toString());

			if (!response.ok) {
				return {
					success: false,
					error: `Search failed: ${response.statusText}`
				};
			}

			const data: SearchResponse = await response.json();

			return {
				success: true,
				data,
				query
			};
		} catch (error) {
			return {
				success: false,
				error: error instanceof Error ? error.message : 'An unknown error occurred'
			};
		}
	},

	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch),

	// Get autocomplete suggestions via backend proxy
	autocomplete: async ({ request, fetch }) => {
		const formData = await request.formData();
		const query = formData.get('q');

		if (!query || typeof query !== 'string' || query.length < 2) {
			return { suggestions: [] };
		}

		try {
			const url = new URL(`${BACKEND_URL}/search/autocomplete`);
			url.searchParams.set('q', query);

			const response = await fetch(url.toString());

			if (!response.ok) {
				return { suggestions: [] };
			}

			const data: { suggestions: string[] } = await response.json();
			return { suggestions: data.suggestions };
		} catch {
			return { suggestions: [] };
		}
	}
} satisfies Actions;
