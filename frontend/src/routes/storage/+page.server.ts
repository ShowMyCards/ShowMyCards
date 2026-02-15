import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail } from '@sveltejs/kit';
import type { StorageLocation } from '$lib';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/storage`);

		if (!response.ok) {
			return {
				locations: [],
				error: 'Failed to load storage locations'
			};
		}

		const result = await response.json();
		const locations: StorageLocation[] = result.data || [];

		return {
			locations
		};
	} catch {
		return {
			locations: [],
			error: 'Failed to load storage locations'
		};
	}
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const data = await request.formData();
		const name = data.get('name') as string;
		const storage_type = data.get('storage_type') as string;

		if (!name) {
			return fail(400, { error: 'Name is required' });
		}

		if (!storage_type) {
			return fail(400, { error: 'Storage type is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/storage`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name,
					storage_type
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				return fail(response.status, {
					error: errorData.error || 'Failed to create storage location'
				});
			}

			const location = await response.json();

			return { success: true, action: 'create', data: location };
		} catch {
			return fail(500, { error: 'Failed to create storage location' });
		}
	},

	update: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;
		const name = data.get('name') as string;
		const storage_type = data.get('storage_type') as string;

		if (!id) {
			return fail(400, { error: 'ID is required' });
		}

		if (!name) {
			return fail(400, { error: 'Name is required' });
		}

		if (!storage_type) {
			return fail(400, { error: 'Storage type is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/storage/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name,
					storage_type
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				return fail(response.status, {
					error: errorData.error || 'Failed to update storage location'
				});
			}

			const location = await response.json();

			return { success: true, action: 'update', data: location };
		} catch {
			return fail(500, { error: 'Failed to update storage location' });
		}
	},

	delete: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;

		if (!id) {
			return fail(400, { error: 'ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/storage/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, {
					error: errorData.error || 'Failed to delete storage location'
				});
			}

			return { success: true, action: 'delete' };
		} catch {
			return fail(500, { error: 'Failed to delete storage location' });
		}
	}
};
