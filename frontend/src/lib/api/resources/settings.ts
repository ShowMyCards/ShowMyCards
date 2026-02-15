import { apiClient } from '../client';

/**
 * Application settings structure
 */
export interface Settings {
	[key: string]: string;
}

/**
 * Request type for updating settings
 */
export interface UpdateSettingsRequest {
	[key: string]: string;
}

/**
 * Settings API methods
 *
 * Provides type-safe access to settings endpoints.
 *
 * @example
 * ```ts
 * import { settingsApi } from '$lib/api';
 *
 * // Get all settings
 * const settings = await settingsApi.get();
 *
 * // Update settings
 * await settingsApi.update({
 *   import_time: '02:00'
 * });
 * ```
 */
export const settingsApi = {
	/**
	 * Get all application settings
	 */
	get: () => apiClient.get<Settings>('/settings'),

	/**
	 * Update application settings
	 */
	update: (data: UpdateSettingsRequest) => apiClient.put<Settings>('/settings', data)
};
