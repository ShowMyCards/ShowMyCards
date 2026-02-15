import {
	BACKEND_URL,
	type InventoryCardsResponse,
	type StorageLocationWithCount,
	type EnhancedCardResult
} from '$lib';
import { handleAddInventory, handleDeleteInventory } from '$lib/server/inventory-actions';
import type { PageServerLoad, Actions } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { id } = params;

	try {
		// Fetch storage location info and all locations in parallel
		const [locationResponse, allLocationsResponse] = await Promise.all([
			fetch(`${BACKEND_URL}/storage/${id}`),
			fetch(`${BACKEND_URL}/storage`)
		]);

		if (!locationResponse.ok) {
			if (locationResponse.status === 404) {
				throw error(404, 'Storage location not found');
			}
			throw new Error('Failed to fetch storage location');
		}
		const location: StorageLocationWithCount = await locationResponse.json();

		// Parse all locations (for bulk move dropdown)
		let allLocations: StorageLocationWithCount[] = [];
		if (allLocationsResponse.ok) {
			const allLocationsData = await allLocationsResponse.json();
			allLocations = allLocationsData.data || [];
		}

		// Fetch ALL cards for this location (for client-side filtering)
		// First, get page 1 to know total pages
		const firstPageResponse = await fetch(
			`${BACKEND_URL}/inventory/cards?storage_location_id=${id}&page=1&page_size=100`
		);
		if (!firstPageResponse.ok) {
			throw new Error('Failed to fetch cards');
		}
		const firstPageData: InventoryCardsResponse = await firstPageResponse.json();
		const allCards: EnhancedCardResult[] = firstPageData.data || [];
		const totalPages = firstPageData.total_pages || 1;

		// Fetch remaining pages in parallel
		if (totalPages > 1) {
			const pagePromises = [];
			for (let page = 2; page <= totalPages; page++) {
				pagePromises.push(
					fetch(
						`${BACKEND_URL}/inventory/cards?storage_location_id=${id}&page=${page}&page_size=100`
					).then((res) => (res.ok ? res.json() : { data: [] }))
				);
			}

			const remainingPages = await Promise.all(pagePromises);
			for (const pageData of remainingPages) {
				allCards.push(...(pageData.data || []));
			}
		}

		return {
			location,
			allLocations,
			cards: allCards,
			totalCards: firstPageData.total_cards
		};
	} catch (e) {
		if (e && typeof e === 'object' && 'status' in e) {
			throw e; // Re-throw SvelteKit errors
		}
		return {
			error: e instanceof Error ? e.message : 'Failed to load inventory',
			location: null,
			allLocations: [],
			cards: [],
			totalCards: 0
		};
	}
};

export const actions = {
	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch)
} satisfies Actions;
