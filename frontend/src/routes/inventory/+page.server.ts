import { BACKEND_URL, type StorageLocationWithCount } from '$lib';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, depends, setHeaders }) => {
	// Create explicit dependency for inventory data
	depends('inventory:counts');

	// Disable caching for this page's data
	setHeaders({
		'cache-control': 'no-store'
	});

	try {
		// Fetch storage locations with card counts
		const locationsResponse = await fetch(`${BACKEND_URL}/storage/with-counts`);
		if (!locationsResponse.ok) {
			throw new Error('Failed to fetch storage locations');
		}
		const locations: StorageLocationWithCount[] = await locationsResponse.json();

		// Fetch unassigned count
		const unassignedResponse = await fetch(`${BACKEND_URL}/inventory/unassigned/count`);
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
