/**
 * API module barrel export
 *
 * Re-exports the API client and all resource modules.
 *
 * @example
 * ```ts
 * import { storageApi, listsApi, rulesApi } from '#lib/api/index.ts';
 *
 * const locations = await storageApi.list();
 * const lists = await listsApi.list();
 * ```
 */

export { apiClient, ApiError } from './client';
export * from './resources';
