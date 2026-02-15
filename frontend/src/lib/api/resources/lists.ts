import { apiClient } from '../client';
import type {
	List,
	ListSummary,
	EnrichedListItem,
	CreateListRequest,
	UpdateListRequest,
	CreateListItemRequest,
	UpdateListItemRequest,
	CreateItemsBatchRequest
} from '$lib';

/**
 * Response type for getting a list with enriched items
 */
export interface ListWithItems {
	id: number;
	name: string;
	description: string;
	created_at: string;
	updated_at: string;
	items: EnrichedListItem[];
}

/**
 * Lists API methods
 *
 * Provides type-safe access to list endpoints.
 *
 * @example
 * ```ts
 * import { listsApi } from '$lib/api';
 *
 * // List all lists
 * const lists = await listsApi.list();
 *
 * // Get a list with items
 * const list = await listsApi.get(1);
 *
 * // Create a list
 * const newList = await listsApi.create({
 *   name: 'My Deck',
 *   description: 'A powerful combo deck'
 * });
 * ```
 */
export const listsApi = {
	/**
	 * List all lists
	 */
	list: () => apiClient.get<ListSummary[]>('/lists'),

	/**
	 * Get a single list with items by ID
	 */
	get: (id: number) => apiClient.get<ListWithItems>(`/lists/${id}`),

	/**
	 * Create a new list
	 */
	create: (data: CreateListRequest) => apiClient.post<List>('/lists', data),

	/**
	 * Update a list
	 */
	update: (id: number, data: UpdateListRequest) => apiClient.put<List>(`/lists/${id}`, data),

	/**
	 * Delete a list
	 */
	delete: (id: number) => apiClient.delete<void>(`/lists/${id}`),

	/**
	 * Add a single item to a list
	 */
	addItem: (listId: number, data: CreateListItemRequest) =>
		apiClient.post<void>(`/lists/${listId}/items`, data),

	/**
	 * Add multiple items to a list
	 */
	addItems: (listId: number, data: CreateItemsBatchRequest) =>
		apiClient.post<void>(`/lists/${listId}/items/batch`, data),

	/**
	 * Update a list item
	 */
	updateItem: (listId: number, itemId: number, data: UpdateListItemRequest) =>
		apiClient.put<void>(`/lists/${listId}/items/${itemId}`, data),

	/**
	 * Remove an item from a list
	 */
	removeItem: (listId: number, itemId: number) =>
		apiClient.delete<void>(`/lists/${listId}/items/${itemId}`)
};
