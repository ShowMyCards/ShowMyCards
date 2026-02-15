import { BACKEND_URL, type InventoryCardsResponse, type StorageLocation } from '$lib';
import { handleAddInventory, handleDeleteInventory } from '$lib/server/inventory-actions';
import type { PageServerLoad, Actions } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		// Fetch recently added cards (the API already orders by created_at DESC)
		const [cardsResponse, storageResponse] = await Promise.all([
			fetch(`${BACKEND_URL}/inventory/cards?page_size=20`),
			fetch(`${BACKEND_URL}/storage`)
		]);

		let cards: InventoryCardsResponse = {
			data: [],
			total_cards: 0,
			total_pages: 0,
			page: 1,
			page_size: 20
		};
		let storageLocations: StorageLocation[] = [];

		if (cardsResponse.ok) {
			cards = await cardsResponse.json();
		}

		if (storageResponse.ok) {
			const storageData = await storageResponse.json();
			storageLocations = storageData.data || [];
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
	deleteInventory: async ({ request, fetch }) => handleDeleteInventory(request, fetch)
} satisfies Actions;
