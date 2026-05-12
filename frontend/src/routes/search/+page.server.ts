import { BACKEND_URL, type SearchResponse, type StorageLocation } from '$lib';
import { handleAddInventory, handleDeleteInventory } from '$lib/server/inventory-actions';
import type { Actions, PageServerLoad } from './$types';
import { fail } from '@sveltejs/kit';

// Load storage locations for the dropdown and the saved default language
export const load: PageServerLoad = async ({ fetch }) => {
	const [storageLocations, defaultLanguage] = await Promise.all([
		loadStorageLocations(fetch),
		loadDefaultLanguage(fetch)
	]);
	return { storageLocations, defaultLanguage };
};

async function loadStorageLocations(fetch: typeof globalThis.fetch): Promise<StorageLocation[]> {
	try {
		const response = await fetch(`${BACKEND_URL}/storage`);
		if (response.ok) {
			const data = await response.json();
			return data.data as StorageLocation[];
		}
	} catch {
		// Ignore errors, dropdown will just be empty
	}
	return [];
}

async function loadDefaultLanguage(fetch: typeof globalThis.fetch): Promise<string> {
	try {
		const response = await fetch(`${BACKEND_URL}/api/settings/scryfall_default_language`);
		if (response.ok) {
			const data = (await response.json()) as { value?: string };
			if (data.value) return data.value;
		}
	} catch {
		// Fall through to default
	}
	return 'en';
}

export const actions = {
	// Search for cards
	search: async ({ request, fetch }) => {
		const formData = await request.formData();
		const query = formData.get('q');
		const lang = formData.get('lang');

		if (!query || typeof query !== 'string') {
			return fail(400, { error: 'Search query is required' });
		}

		try {
			const url = new URL(`${BACKEND_URL}/search`);
			url.searchParams.set('q', query);
			if (typeof lang === 'string' && lang.trim() !== '') {
				url.searchParams.set('lang', lang.trim());
			}

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
