import { BACKEND_URL, type SearchResponse } from '#lib';
import {
	handleAddInventory,
	handleDeleteInventory,
	handleUpdateInventory
} from '#lib/server/inventory-actions.ts';
import { loadStorageLocations } from '#lib/server/storage.ts';
import type { Actions, PageServerLoad } from './$types';

export interface SearchResult {
	query: string;
	data?: SearchResponse;
	error?: string;
}

// The search is URL-driven (`/search?q=...&lang=...`) so back-navigation,
// reloads and bookmarks all reproduce the results.
export const load: PageServerLoad = async ({ fetch, url }) => {
	const query = url.searchParams.get('q')?.trim() ?? '';
	const lang = url.searchParams.get('lang')?.trim() || null;

	const [storageLocations, defaultLanguage, search] = await Promise.all([
		loadStorageLocations(fetch),
		loadDefaultLanguage(fetch),
		query ? runSearch(fetch, query, lang) : Promise.resolve(null)
	]);
	return { storageLocations, defaultLanguage, search, lang };
};

async function runSearch(
	fetch: typeof globalThis.fetch,
	query: string,
	lang: string | null
): Promise<SearchResult> {
	try {
		const url = new URL(`${BACKEND_URL}/api/search`);
		url.searchParams.set('q', query);
		if (lang) {
			url.searchParams.set('lang', lang);
		}

		const response = await fetch(url.toString());
		if (!response.ok) {
			return { query, error: `Search failed: ${response.statusText}` };
		}

		const data: SearchResponse = await response.json();
		return { query, data };
	} catch (error) {
		return {
			query,
			error: error instanceof Error ? error.message : 'An unknown error occurred'
		};
	}
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
	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch),
	updateInventory: async ({ request, fetch }) => handleUpdateInventory(request, fetch),

	// Get autocomplete suggestions via backend proxy
	autocomplete: async ({ request, fetch }) => {
		const formData = await request.formData();
		const query = formData.get('q');

		if (!query || typeof query !== 'string' || query.length < 2) {
			return { suggestions: [] };
		}

		try {
			const url = new URL(`${BACKEND_URL}/api/search/autocomplete`);
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
