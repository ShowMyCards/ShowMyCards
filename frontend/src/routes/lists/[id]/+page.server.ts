import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail } from '@sveltejs/kit';
import type { List, EnrichedListItem } from '$lib';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const { id } = params;

	try {
		// Load list metadata
		const listResponse = await fetch(`${BACKEND_URL}/lists/${id}`);
		if (!listResponse.ok) {
			return {
				list: null,
				items: [],
				totalItems: 0,
				totalWanted: 0,
				totalCollected: 0,
				completionPercent: 0,
				totalCollectedValue: 0,
				totalRemainingValue: 0,
				error: 'Failed to load list'
			};
		}
		const list: List = await listResponse.json();

		// Fetch ALL list items for client-side filtering/pagination
		// First, get page 1 to know total pages
		const firstPageResponse = await fetch(
			`${BACKEND_URL}/lists/${id}/items?page=1&page_size=100`
		);
		if (!firstPageResponse.ok) {
			return {
				list,
				items: [],
				totalItems: 0,
				totalWanted: 0,
				totalCollected: 0,
				completionPercent: 0,
				totalCollectedValue: 0,
				totalRemainingValue: 0,
				error: 'Failed to load list items'
			};
		}

		const firstPageData = await firstPageResponse.json();
		const allItems: EnrichedListItem[] = firstPageData.data || [];
		const totalPages = firstPageData.total_pages || 1;

		// Fetch remaining pages in parallel
		if (totalPages > 1) {
			const pagePromises = [];
			for (let page = 2; page <= totalPages; page++) {
				pagePromises.push(
					fetch(`${BACKEND_URL}/lists/${id}/items?page=${page}&page_size=100`).then((res) =>
						res.ok ? res.json() : { data: [] }
					)
				);
			}

			const remainingPages = await Promise.all(pagePromises);
			for (const pageData of remainingPages) {
				allItems.push(...(pageData.data || []));
			}
		}

		return {
			list,
			items: allItems,
			totalItems: firstPageData.total_items || allItems.length,
			totalWanted: firstPageData.total_wanted || 0,
			totalCollected: firstPageData.total_collected || 0,
			completionPercent: firstPageData.completion_percent || 0,
			totalCollectedValue: firstPageData.total_collected_value || 0,
			totalRemainingValue: firstPageData.total_remaining_value || 0
		};
	} catch {
		return {
			list: null,
			items: [],
			totalItems: 0,
			totalWanted: 0,
			totalCollected: 0,
			completionPercent: 0,
			totalCollectedValue: 0,
			totalRemainingValue: 0,
			error: 'Failed to load list'
		};
	}
};

export const actions: Actions = {
	// Search for cards
	search: async ({ request, fetch }) => {
		const data = await request.formData();
		const query = data.get('q') as string;

		if (!query) {
			return fail(400, { error: 'Search query is required', searchResults: [] });
		}

		try {
			let page = 1;
			let hasMore = true;
			const allResults = [];

			// Fetch all pages of search results
			while (hasMore) {
				const searchResponse = await fetch(
					`${BACKEND_URL}/search?q=${encodeURIComponent(query)}&page=${page}`
				);
				if (searchResponse.ok) {
					const searchData = await searchResponse.json();
					allResults.push(...(searchData.data || []));
					hasMore = searchData.has_more || false;
					page++;

					// Safety limit: don't fetch more than 10 pages (1750 cards)
					if (page > 10) {
						break;
					}
				} else {
					hasMore = false;
				}
			}

			return {
				success: true,
				action: 'search',
				searchResults: allResults,
				query
			};
		} catch {
			return fail(500, { error: 'Search failed', searchResults: [] });
		}
	},

	// Add cards to list (batch)
	addItems: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemsJson = data.get('items') as string;

		if (!itemsJson) {
			return fail(400, { error: 'No items provided' });
		}

		try {
			const items = JSON.parse(itemsJson);

			const response = await fetch(`${BACKEND_URL}/lists/${id}/items/batch`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ items })
			});

			if (!response.ok) {
				const errorData = await response.json();
				return fail(response.status, { error: errorData.error || 'Failed to add items' });
			}

			return { success: true, action: 'add' };
		} catch {
			return fail(500, { error: 'Failed to add items' });
		}
	},

	// Update collected quantity
	updateItem: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemId = data.get('item_id') as string;
		const collectedQuantity = data.get('collected_quantity') as string;

		if (!itemId) {
			return fail(400, { error: 'Item ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/lists/${id}/items/${itemId}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					collected_quantity: parseInt(collectedQuantity, 10)
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				return fail(response.status, { error: errorData.error || 'Failed to update item' });
			}

			return { success: true, action: 'update' };
		} catch {
			return fail(500, { error: 'Failed to update item' });
		}
	},

	// Delete item from list
	deleteItem: async ({ params, request, fetch }) => {
		const { id } = params;
		const data = await request.formData();
		const itemId = data.get('item_id') as string;

		if (!itemId) {
			return fail(400, { error: 'Item ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/lists/${id}/items/${itemId}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to delete item' });
			}

			return { success: true, action: 'delete' };
		} catch {
			return fail(500, { error: 'Failed to delete item' });
		}
	}
};
