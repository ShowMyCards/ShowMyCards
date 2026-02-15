import {
	BACKEND_URL,
	type InventoryCardsResponse,
	type StorageLocationWithCount,
	type EnhancedCardResult
} from '$lib';
import { handleAddInventory, handleDeleteInventory } from '$lib/server/inventory-actions';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		// Fetch ALL unassigned cards and all locations in parallel
		// First, get page 1 to know total pages
		const [firstPageResponse, locationsResponse] = await Promise.all([
			fetch(`${BACKEND_URL}/inventory/cards?storage_location_id=null&page=1&page_size=100`),
			fetch(`${BACKEND_URL}/storage`)
		]);

		if (!firstPageResponse.ok) {
			throw new Error('Failed to fetch unassigned cards');
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
						`${BACKEND_URL}/inventory/cards?storage_location_id=null&page=${page}&page_size=100`
					).then((res) => (res.ok ? res.json() : { data: [] }))
				);
			}

			const remainingPages = await Promise.all(pagePromises);
			for (const pageData of remainingPages) {
				allCards.push(...(pageData.data || []));
			}
		}

		// Parse locations for bulk move dropdown
		let allLocations: StorageLocationWithCount[] = [];
		if (locationsResponse.ok) {
			const locationsData = await locationsResponse.json();
			allLocations = locationsData.data || [];
		}

		return {
			cards: allCards,
			allLocations,
			totalCards: firstPageData.total_cards
		};
	} catch (e) {
		return {
			error: e instanceof Error ? e.message : 'Failed to load unassigned cards',
			cards: [],
			allLocations: [],
			totalCards: 0
		};
	}
};

export const actions = {
	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch)
} satisfies Actions;
