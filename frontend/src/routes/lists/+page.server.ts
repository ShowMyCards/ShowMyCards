import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail, redirect } from '@sveltejs/kit';
import type { ListSummary } from '$lib';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/lists`);

		if (!response.ok) {
			return {
				lists: [],
				error: 'Failed to load lists'
			};
		}

		const lists: ListSummary[] = await response.json();

		return {
			lists
		};
	} catch (error) {
		return {
			lists: [],
			error: 'Failed to load lists'
		};
	}
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const data = await request.formData();
		const name = data.get('name') as string;
		const description = data.get('description') as string;

		if (!name) {
			return fail(400, { error: 'Name is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/lists`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name,
					description
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				return fail(response.status, { error: errorData.error || 'Failed to create list' });
			}

			const list = await response.json();

			// Redirect to the new list page
			throw redirect(303, `/lists/${list.id}`);
		} catch (error) {
			if (error instanceof Response && error.status === 303) {
				throw error;
			}
			return fail(500, { error: 'Failed to create list' });
		}
	},

	delete: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;

		if (!id) {
			return fail(400, { error: 'ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/lists/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to delete list' });
			}

			return { success: true, action: 'delete' };
		} catch (error) {
			return fail(500, { error: 'Failed to delete list' });
		}
	}
};
