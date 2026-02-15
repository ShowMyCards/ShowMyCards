import { apiClient } from '../client';
import type { Inventory, InventoryCardsResponse } from '$lib';
import type {
	BatchMoveRequest,
	BatchMoveResponse,
	BatchDeleteRequest,
	BatchDeleteResponse,
	ResortRequest,
	ResortResponse
} from '$lib/types/api';

/**
 * Request type for creating inventory
 */
export interface CreateInventoryRequest {
	scryfall_id: string;
	treatment: string;
	quantity: number;
	storage_location_id?: number;
}

/**
 * Request type for updating inventory
 */
export interface UpdateInventoryRequest {
	treatment?: string;
	quantity?: number;
	storage_location_id?: number;
	clear_storage?: boolean;
}

/**
 * Query parameters for listing inventory
 */
export interface ListInventoryParams {
	scryfall_id?: string;
	storage_location_id?: string | number; // Can be "null" for unassigned
}

/**
 * Inventory API methods
 *
 * Provides type-safe access to inventory endpoints.
 *
 * @example
 * ```ts
 * import { inventoryApi } from '$lib/api';
 *
 * // List all inventory items
 * const items = await inventoryApi.list();
 *
 * // Get unassigned items
 * const unassigned = await inventoryApi.list({ storage_location_id: 'null' });
 *
 * // Create inventory
 * const item = await inventoryApi.create({
 *   scryfall_id: 'abc123',
 *   treatment: 'Foil',
 *   quantity: 1
 * });
 * ```
 */
export const inventoryApi = {
	/**
	 * List inventory items with optional filters
	 */
	list: (params?: ListInventoryParams) => {
		const searchParams = new URLSearchParams();
		if (params?.scryfall_id) searchParams.set('scryfall_id', params.scryfall_id);
		if (params?.storage_location_id !== undefined)
			searchParams.set('storage_location_id', String(params.storage_location_id));

		const query = searchParams.toString();
		return apiClient.get<InventoryCardsResponse>(`/inventory${query ? `?${query}` : ''}`);
	},

	/**
	 * Get a single inventory item by ID
	 */
	get: (id: number) => apiClient.get<Inventory>(`/inventory/${id}`),

	/**
	 * Create a new inventory item
	 */
	create: (data: CreateInventoryRequest) => apiClient.post<Inventory>('/inventory', data),

	/**
	 * Update an inventory item
	 */
	update: (id: number, data: UpdateInventoryRequest) =>
		apiClient.put<Inventory>(`/inventory/${id}`, data),

	/**
	 * Delete an inventory item
	 */
	delete: (id: number) => apiClient.delete<void>(`/inventory/${id}`),

	/**
	 * Move multiple inventory items to a new storage location
	 */
	batchMove: (ids: number[], storageLocationId?: number) =>
		apiClient.post<BatchMoveResponse>('/inventory/batch/move', {
			ids,
			storage_location_id: storageLocationId
		} as BatchMoveRequest),

	/**
	 * Delete multiple inventory items
	 */
	batchDelete: (ids: number[]) =>
		apiClient.delete<BatchDeleteResponse>('/inventory/batch', { ids } as BatchDeleteRequest),

	/**
	 * Re-evaluate inventory items against sorting rules
	 * If no ids provided, re-evaluates all items
	 */
	resort: (ids?: number[]) =>
		apiClient.post<ResortResponse>('/inventory/resort', { ids } as ResortRequest)
};
