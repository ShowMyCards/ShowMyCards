import { BACKEND_URL } from '$lib';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const body = await request.text();

		const response = await fetch(`${BACKEND_URL}/api/data/import`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			throw error(response.status, errorData.error || 'Failed to import data');
		}

		const data = await response.json();
		return json(data);
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to import data');
	}
};
