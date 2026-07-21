import { BACKEND_URL, type EnhancedCardResult, type DeckCardUsage } from '$lib';
import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

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

		// Fetch other printings by oracle_id (search for all cards with same oracle_id)
		let otherPrintings: EnhancedCardResult[] = [];
		if (card.oracle_id) {
			try {
				const searchResponse = await fetch(
					`${BACKEND_URL}/api/search?q=oracle_id:${card.oracle_id}&unique=prints`
				);
				if (searchResponse.ok) {
					const searchData = await searchResponse.json();
					// Filter out the current card
					otherPrintings = (searchData.data || []).filter(
						(c: EnhancedCardResult) => c.id !== card.id
					);
				}
			} catch {
				// Ignore errors fetching other printings
			}
		}

		// Fetch which decks use this printing (any-printing or pinned to it). A
		// failure here must not break the card page, so degrade to an empty list.
		let deckUsage: DeckCardUsage[] = [];
		try {
			const decksResponse = await fetch(`${BACKEND_URL}/api/decks/for-card/${id}`);
			if (decksResponse.ok) {
				deckUsage = await decksResponse.json();
			}
		} catch {
			// Ignore errors fetching deck usage.
		}

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
