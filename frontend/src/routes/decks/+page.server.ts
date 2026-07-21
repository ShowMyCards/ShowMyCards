import type { PageServerLoad, Actions } from './$types';
import { BACKEND_URL } from '$lib';
import { fail } from '@sveltejs/kit';
import type { DeckSummary } from '$lib';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/api/decks`);

		if (!response.ok) {
			return {
				decks: [] as DeckSummary[],
				error: 'Failed to load decks'
			};
		}

		const decks: DeckSummary[] = await response.json();

		return {
			decks
		};
	} catch {
		return {
			decks: [] as DeckSummary[],
			error: 'Failed to load decks'
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
			const response = await fetch(`${BACKEND_URL}/api/decks`, {
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
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to create deck' });
			}

			const deck = await response.json();

			return { success: true, action: 'create', data: deck };
		} catch {
			return fail(500, { error: 'Failed to create deck' });
		}
	},

	update: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;
		const name = data.get('name') as string;
		const description = data.get('description') as string;

		if (!id) {
			return fail(400, { error: 'ID is required' });
		}

		if (!name) {
			return fail(400, { error: 'Name is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name,
					description
				})
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, { error: errorData.error || 'Failed to update deck' });
			}

			return { success: true, action: 'update' };
		} catch {
			return fail(500, { error: 'Failed to update deck' });
		}
	},

	delete: async ({ request, fetch }) => {
		const data = await request.formData();
		const id = data.get('id') as string;

		if (!id) {
			return fail(400, { error: 'ID is required' });
		}

		try {
			const response = await fetch(`${BACKEND_URL}/api/decks/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				return fail(response.status, { error: 'Failed to delete deck' });
			}

			return { success: true, action: 'delete' };
		} catch {
			return fail(500, { error: 'Failed to delete deck' });
		}
	}
};
