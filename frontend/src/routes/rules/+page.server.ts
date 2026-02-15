import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const page = url.searchParams.get('page') || '1';
	const enabled = url.searchParams.get('enabled');

	try {
		// Build query params
		let queryParams = `?page=${page}&page_size=20`;
		if (enabled !== null) {
			queryParams += `&enabled=${enabled}`;
		}

		// Fetch sorting rules
		const rulesResponse = await fetch(`${BACKEND_URL}/sorting-rules${queryParams}`);
		if (!rulesResponse.ok) {
			throw new Error('Failed to load sorting rules');
		}
		const rulesData = await rulesResponse.json();

		// Fetch storage locations for dropdown
		const locationsResponse = await fetch(`${BACKEND_URL}/storage?page_size=100`);
		const locationsData = locationsResponse.ok ? await locationsResponse.json() : { data: [] };

		return {
			rules: rulesData.data || [],
			pagination: {
				page: rulesData.page || 1,
				page_size: rulesData.page_size || 20,
				total: rulesData.total || 0,
				total_pages: rulesData.total_pages || 1
			},
			storageLocations: locationsData.data || []
		};
	} catch (error) {
		return {
			rules: [],
			pagination: { page: 1, page_size: 20, total: 0, total_pages: 1 },
			storageLocations: [],
			error: 'Failed to load sorting rules'
		};
	}
};

export const actions: Actions = {
	// Create a new sorting rule
	create: async ({ request, fetch }) => {
		const data = await request.formData();
		const ruleJson = data.get('rule') as string;

		if (!ruleJson) {
			return fail(400, { error: 'Rule data is required' });
		}

		try {
			const ruleData = JSON.parse(ruleJson);

			const response = await fetch(`${BACKEND_URL}/sorting-rules`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(ruleData)
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to create rule' });
			}

			return { success: true, action: 'create' };
		} catch (error) {
			return fail(500, { error: 'Failed to create rule' });
		}
	},

	// Update a sorting rule
	update: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;
		const ruleJson = data.get('rule') as string;

		if (!id || !ruleJson) {
			return fail(400, { error: 'Rule ID and data are required' });
		}

		try {
			const ruleData = JSON.parse(ruleJson);

			const response = await fetch(`${BACKEND_URL}/sorting-rules/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(ruleData)
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to update rule' });
			}

			return { success: true, action: 'update' };
		} catch (error) {
			return fail(500, { error: 'Failed to update rule' });
		}
	},

	// Delete a sorting rule
	delete: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;

		if (!id) {
			return fail(400, { error: 'Rule ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/sorting-rules/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to delete rule' });
			}

			return { success: true, action: 'delete' };
		} catch (error) {
			return fail(500, { error: 'Failed to delete rule' });
		}
	},

	// Toggle rule enabled state
	toggleEnabled: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;
		const enabled = data.get('enabled') === 'true';

		if (!id) {
			return fail(400, { error: 'Rule ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/sorting-rules/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ enabled })
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to toggle rule' });
			}

			return {
				success: true,
				action: 'toggle',
				message: `Rule ${enabled ? 'enabled' : 'disabled'}`
			};
		} catch (error) {
			return fail(500, { error: 'Failed to toggle rule' });
		}
	},

	// Batch update priorities (for drag-and-drop)
	batchUpdatePriorities: async ({ request, fetch }) => {
		const data = await request.formData();
		const updatesJson = data.get('updates') as string;

		if (!updatesJson) {
			return fail(400, { error: 'Updates data is required' });
		}

		try {
			const updates = JSON.parse(updatesJson);

			const response = await fetch(`${BACKEND_URL}/sorting-rules/batch/priorities`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ updates })
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, {
					error: errorData.error || 'Failed to update priorities'
				});
			}

			const result = await response.json();
			return {
				success: true,
				action: 'batchUpdate',
				message: `${result.updated_count} rule priorities updated`
			};
		} catch (error) {
			return fail(500, { error: 'Failed to update priorities' });
		}
	}
};
