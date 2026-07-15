import { BACKEND_URL } from '$lib';
import { fetchStorageLocationsWithCounts } from '$lib/server/storage';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, depends, setHeaders }) => {
	// Create explicit dependency for inventory data
	depends('inventory:counts');

	// Disable caching for this page's data
	setHeaders({
		'cache-control': 'no-store'
	});

	try {
		// Fetch storage locations with card counts. Throws on failure so the
		// catch below can surface an error rather than render an empty list.
		const locations = await fetchStorageLocationsWithCounts(fetch);

		// Fetch unassigned count
		const unassignedResponse = await fetch(`${BACKEND_URL}/api/inventory/unassigned/count`);
		if (!unassignedResponse.ok) {
			throw new Error('Failed to fetch unassigned count');
		}
		const { count: unassignedCount } = await unassignedResponse.json();

		return {
			locations,
			unassignedCount
		};
	} catch (e) {
		return {
			error: e instanceof Error ? e.message : 'Failed to load inventory data',
			locations: [],
			unassignedCount: 0
		};
	}
};
