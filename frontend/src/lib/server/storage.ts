/**
 * Shared server-side loaders for the full set of storage locations.
 * Used by every dropdown/list that needs all locations (search, import, storage,
 * inventory/recent, inventory/[id], inventory/unassigned, rules).
 *
 * The backend `GET /api/storage` endpoint is paginated (default page_size 20,
 * max 100), so a plain fetch silently truncates the list. These helpers page
 * through the endpoint to return every location.
 */
import { BACKEND_URL, type StorageLocation, type StorageLocationWithCount } from '$lib';

// The backend caps page_size at MaxPageSize (100); request the max per page.
const PAGE_SIZE = 100;

/**
 * Fetch ALL storage locations, transparently paging past the backend's default
 * and maximum page size. Returns an empty array on failure (never throws) so
 * callers can treat a missing list the same as an empty one.
 */
export async function loadStorageLocations(
	fetch: typeof globalThis.fetch
): Promise<StorageLocation[]> {
	try {
		const first = await fetch(`${BACKEND_URL}/api/storage?page=1&page_size=${PAGE_SIZE}`);
		if (!first.ok) return [];

		const firstData = await first.json();
		const locations: StorageLocation[] = firstData.data || [];
		const totalPages = firstData.total_pages || 1;

		if (totalPages > 1) {
			const rest = await Promise.all(
				Array.from({ length: totalPages - 1 }, (_, i) =>
					fetch(`${BACKEND_URL}/api/storage?page=${i + 2}&page_size=${PAGE_SIZE}`).then((r) =>
						r.ok ? r.json() : { data: [] }
					)
				)
			);
			for (const pageData of rest) {
				locations.push(...(pageData.data || []));
			}
		}

		return locations;
	} catch {
		return [];
	}
}

/**
 * Fetch ALL storage locations with their card counts, item counts, and total
 * values. Backed by the already-unbounded `GET /api/storage/with-counts`
 * endpoint (returns a bare array, not a paginated envelope). Returns an empty
 * array on failure (never throws).
 */
export async function loadStorageLocationsWithCounts(
	fetch: typeof globalThis.fetch
): Promise<StorageLocationWithCount[]> {
	try {
		const response = await fetch(`${BACKEND_URL}/api/storage/with-counts`);
		if (!response.ok) return [];
		return (await response.json()) as StorageLocationWithCount[];
	} catch {
		return [];
	}
}
