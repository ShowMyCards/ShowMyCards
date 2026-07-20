/**
 * Shared server-side loaders for the full set of storage locations.
 *
 * The backend `GET /api/storage` endpoint is paginated (default page_size 20,
 * max 100), so a plain fetch silently truncates the list. The core helpers
 * here page through it to return every location.
 *
 * Two contracts are offered, because callers differ in what a failure means:
 *
 * - `load*` variants SWALLOW failure and return `[]`. Use these where the list
 *   is auxiliary (e.g. a bulk-move dropdown next to the real content): an empty
 *   list and a failed request are interchangeable, so there is nothing to
 *   report.
 * - `fetch*` variants THROW on failure. Use these where the list is the page's
 *   primary content and "failed to load" must be distinguishable from
 *   "genuinely empty" (otherwise a backend outage silently renders as an empty
 *   collection).
 */
import { BACKEND_URL, type StorageLocation, type StorageLocationWithCount } from '$lib';

// The backend caps page_size at MaxPageSize (100); request the max per page.
const PAGE_SIZE = 100;

/**
 * Page through `GET /api/storage` and return every location. Throws if the
 * first page fails, so callers whose primary content is this list can
 * distinguish a failed load from an empty collection; a failed later page
 * degrades to the locations gathered so far rather than losing the whole list.
 */
export async function fetchStorageLocations(
	fetch: typeof globalThis.fetch
): Promise<StorageLocation[]> {
	const first = await fetch(`${BACKEND_URL}/api/storage?page=1&page_size=${PAGE_SIZE}`);
	if (!first.ok) {
		throw new Error(`Failed to load storage locations (HTTP ${first.status})`);
	}

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
}

/**
 * Fetch ALL storage locations, transparently paging past the backend's default
 * and maximum page size. Returns an empty array on failure (never throws) so
 * callers can treat a missing list the same as an empty one.
 */
export async function loadStorageLocations(
	fetch: typeof globalThis.fetch
): Promise<StorageLocation[]> {
	try {
		return await fetchStorageLocations(fetch);
	} catch {
		return [];
	}
}

/**
 * Fetch ALL storage locations with their card counts, item counts, and total
 * values. Backed by the already-unbounded `GET /api/storage/with-counts`
 * endpoint (returns a bare array, not a paginated envelope). Throws on failure
 * so callers whose primary content is this list can distinguish a failed load
 * from an empty collection.
 */
export async function fetchStorageLocationsWithCounts(
	fetch: typeof globalThis.fetch
): Promise<StorageLocationWithCount[]> {
	const response = await fetch(`${BACKEND_URL}/api/storage/with-counts`);
	if (!response.ok) {
		throw new Error(`Failed to load storage locations (HTTP ${response.status})`);
	}
	return (await response.json()) as StorageLocationWithCount[];
}
