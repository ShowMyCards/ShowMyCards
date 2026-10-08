import { BACKEND_URL } from '$lib/config';
import type { EnhancedCardResult, InventoryCardsResponse } from '$lib';

/** Load the complete collection, merging printings split across inventory-record pages. */
export async function fetchAllInventoryCards(
	fetch: typeof globalThis.fetch
): Promise<EnhancedCardResult[]> {
	async function fetchPage(page: number): Promise<InventoryCardsResponse> {
		const response = await fetch(`${BACKEND_URL}/api/inventory/cards?page=${page}&page_size=50`);
		if (!response.ok) {
			throw new Error(`Failed to load all cards (HTTP ${response.status})`);
		}
		return response.json();
	}

	const firstPage = await fetchPage(1);
	const cards = new Map<string, EnhancedCardResult>();
	const inventoryIds = new Set<number>();

	function mergePage(page: InventoryCardsResponse) {
		for (const card of page.data ?? []) {
			let merged = cards.get(card.id);
			if (!merged) {
				merged = {
					...card,
					inventory: { this_printing: [], other_printings: [], total_quantity: 0 }
				};
				cards.set(card.id, merged);
			}
			for (const item of card.inventory.this_printing) {
				if (inventoryIds.has(item.id)) continue;
				inventoryIds.add(item.id);
				merged.inventory.this_printing.push(item);
				merged.inventory.total_quantity += item.quantity;
			}
		}
	}

	mergePage(firstPage);
	// Limit concurrent requests for large collections, while preserving page order.
	for (let page = 2; page <= firstPage.total_pages; page += 4) {
		const pages = await Promise.all(
			Array.from({ length: Math.min(4, firstPage.total_pages - page + 1) }, (_, i) =>
				fetchPage(page + i)
			)
		);
		for (const data of pages) mergePage(data);
	}

	return [...cards.values()];
}
