import { BACKEND_URL, type DeckCardUsage, type EnhancedCardResult, type InventoryCard } from '#lib';
import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

async function loadOtherPrintings(
	fetch: typeof globalThis.fetch,
	card: EnhancedCardResult
): Promise<InventoryCard[]> {
	if (!card.oracle_id) return [];
	try {
		const params = new URLSearchParams();
		if (card.language) params.set('lang', card.language);
		const query = params.toString();
		const response = await fetch(
			`${BACKEND_URL}/api/cards/printings/${encodeURIComponent(card.oracle_id)}${query ? `?${query}` : ''}`
		);
		if (!response.ok) return [];
		const printings: InventoryCard[] = await response.json();
		return (printings ?? []).filter((c) => c.id !== card.id);
	} catch {
		return [];
	}
}

async function loadDeckUsage(fetch: typeof globalThis.fetch, id: string): Promise<DeckCardUsage[]> {
	try {
		const response = await fetch(`${BACKEND_URL}/api/decks/for-card/${id}`);
		return response.ok ? await response.json() : [];
	} catch {
		return [];
	}
}

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { id } = params;

	try {
		// Fetch the card details from the backend
		const cardResponse = await fetch(`${BACKEND_URL}/api/cards/${id}`);

		if (!cardResponse.ok) {
			if (cardResponse.status === 404) {
				throw error(404, 'Card not found');
			}
			throw error(cardResponse.status, 'Failed to fetch card');
		}

		const card: EnhancedCardResult = await cardResponse.json();
		const [otherPrintings, deckUsage] = await Promise.all([
			loadOtherPrintings(fetch, card),
			loadDeckUsage(fetch, id)
		]);

		return {
			card,
			otherPrintings,
			deckUsage
		};
	} catch (e) {
		if (e && typeof e === 'object' && 'status' in e) {
			throw e; // Re-throw SvelteKit errors
		}
		throw error(500, 'Failed to load card details');
	}
};
