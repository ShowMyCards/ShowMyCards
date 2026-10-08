import { fetchAllInventoryCards } from '$lib/server/inventory-cards';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, depends, setHeaders }) => {
	depends('inventory:cards');
	setHeaders({ 'cache-control': 'no-store' });

	try {
		const cards = await fetchAllInventoryCards(fetch);
		return {
			cards,
			totalCopies: cards.reduce((total, card) => total + card.inventory.total_quantity, 0)
		};
	} catch (e) {
		return {
			error: e instanceof Error ? e.message : 'Failed to load all cards',
			cards: [],
			totalCopies: 0
		};
	}
};
