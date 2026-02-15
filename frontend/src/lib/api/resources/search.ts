import { apiClient } from '../client';
import type { SearchResponse } from '$lib';

/**
 * Query parameters for card search
 */
export interface SearchParams {
	q: string;
	page?: number;
}

/**
 * Card search API methods
 *
 * Provides type-safe access to Scryfall card search endpoints (proxied through backend).
 *
 * @example
 * ```ts
 * import { searchApi } from '$lib/api';
 *
 * // Search for cards
 * const results = await searchApi.cards({ q: 'lightning bolt', page: 1 });
 * ```
 */
export const searchApi = {
	/**
	 * Search for cards using Scryfall syntax
	 *
	 * @param params - Search query and optional page number
	 */
	cards: (params: SearchParams) => {
		const searchParams = new URLSearchParams();
		searchParams.set('q', params.q);
		if (params.page) searchParams.set('page', String(params.page));

		return apiClient.get<SearchResponse>(`/search?${searchParams.toString()}`);
	}
};
