import { apiClient } from '../client';
import type { DeckSummary, CreateDeckRequest } from '$lib';

/** A deck as returned by create (BaseModel fields flattened at runtime). */
export interface CreatedDeck {
	id: number;
	name: string;
	description?: string;
}

/**
 * Decks API methods (client-side reads and deck creation via the /api proxy).
 *
 * Adding cards to a deck goes through the SvelteKit route
 * `POST /decks/[id]/items` instead, which runs the shared server-side upsert.
 */
export const decksApi = {
	/** List all decks with summary statistics. */
	list: () => apiClient.get<DeckSummary[]>('/decks'),

	/** Create a new deck. */
	create: (data: CreateDeckRequest) => apiClient.post<CreatedDeck>('/decks', data)
};
