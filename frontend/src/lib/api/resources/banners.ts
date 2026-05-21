import { apiClient } from '../client';
import type { Banner } from '$lib';

/**
 * Banners API methods.
 *
 * Provides type-safe access to the banner endpoint, which returns the
 * server-derived banners that should currently be displayed.
 *
 * @example
 * ```ts
 * import { bannersApi } from '$lib';
 *
 * const banners = await bannersApi.list();
 * ```
 */
export const bannersApi = {
	/**
	 * List the banners that should currently be displayed.
	 */
	list: () => apiClient.get<Banner[]>('/banners')
};
