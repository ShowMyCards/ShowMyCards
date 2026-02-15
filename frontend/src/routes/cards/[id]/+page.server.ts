import { BACKEND_URL, type EnhancedCardResult } from '$lib';
import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { id } = params;

	try {
		// Fetch the card details from the backend
		const cardResponse = await fetch(`${BACKEND_URL}/cards/${id}`);

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
					`${BACKEND_URL}/search?q=oracle_id:${card.oracle_id}&unique=prints`
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

		return {
			card,
			otherPrintings
		};
	} catch (e) {
		if (e && typeof e === 'object' && 'status' in e) {
			throw e; // Re-throw SvelteKit errors
		}
		throw error(500, 'Failed to load card details');
	}
};
