import { apiClient } from '../client';
import type { StorageLocation } from '$lib';

/**
 * Request type for creating a storage location
 */
export interface CreateStorageRequest {
	name: string;
	storage_type: 'Box' | 'Binder';
	[key: string]: unknown;
}

/**
 * Request type for updating a storage location
 */
export interface UpdateStorageRequest {
	name?: string;
	storage_type?: 'Box' | 'Binder';
	[key: string]: unknown;
}

/**
 * Paginated response from the storage API
 */
export interface PaginatedStorageResponse {
	data: StorageLocation[];
	page: number;
	page_size: number;
	total_items: number;
	total_pages: number;
}

/**
 * Storage location API methods
 *
 * Provides type-safe access to storage location endpoints.
 *
 * @example
 * ```ts
 * import { storageApi } from '$lib/api';
 *
 * // List all storage locations
 * const response = await storageApi.list();
 * const locations = response.data;
 *
 * // Create a new storage location
 * const newLocation = await storageApi.create({
 *   name: 'Box 1',
 *   storage_type: 'Box'
 * });
 * ```
 */
export const storageApi = {
	/**
	 * List all storage locations (paginated response)
	 */
	list: () => apiClient.get<PaginatedStorageResponse>('/storage'),

	/**
	 * Get a single storage location by ID
	 */
	get: (id: number) => apiClient.get<StorageLocation>(`/storage/${id}`),

	/**
	 * Create a new storage location
	 */
	create: (data: CreateStorageRequest) => apiClient.post<StorageLocation>('/storage', data),

	/**
	 * Update a storage location
	 */
	update: (id: number, data: UpdateStorageRequest) =>
		apiClient.put<StorageLocation>(`/storage/${id}`, data),

	/**
	 * Delete a storage location
	 */
	delete: (id: number) => apiClient.delete<void>(`/storage/${id}`)
};
