import { BACKEND_URL, type DashboardStats } from '$lib';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/api/dashboard/stats`);
		if (!response.ok) {
			throw new Error('Failed to fetch dashboard statistics');
		}
		const stats: DashboardStats = await response.json();

		return {
			stats
		};
	} catch (e) {
		return {
			error: e instanceof Error ? e.message : 'Failed to load dashboard data',
			stats: {
				total_inventory_cards: 0,
				total_wishlist_cards: 0,
				total_collection_value: 0,
				total_collected_from_lists: 0,
				total_remaining_lists_value: 0,
				total_storage_locations: 0,
				total_lists: 0,
				unassigned_cards: 0
			}
		};
	}
};
