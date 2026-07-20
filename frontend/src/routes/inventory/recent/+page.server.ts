import { BACKEND_URL, type InventoryCardsResponse } from '$lib';
import {
	handleAddInventory,
	handleDeleteInventory,
	handleUpdateInventory
} from '$lib/server/inventory-actions';
import { loadStorageLocations } from '$lib/server/storage';
import type { PageServerLoad, Actions } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		// Fetch recently added cards (the API already orders by created_at DESC)
		const [cardsResponse, storageLocations] = await Promise.all([
			fetch(`${BACKEND_URL}/api/inventory/cards?page_size=20`),
			loadStorageLocations(fetch)
		]);

		let cards: InventoryCardsResponse = {
			data: [],
			total_cards: 0,
			total_pages: 0,
			page: 1,
			page_size: 20
		};

		if (cardsResponse.ok) {
			cards = await cardsResponse.json();
		}

		return {
			cards: cards.data,
			total: cards.total_cards,
			storageLocations
		};
	} catch {
		return {
			cards: [],
			total: 0,
			storageLocations: []
		};
	}
};

export const actions = {
	addInventory: async ({ request, fetch }) => handleAddInventory(request, fetch),
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch),
	updateInventory: async ({ request, fetch }) => handleUpdateInventory(request, fetch)
} satisfies Actions;
