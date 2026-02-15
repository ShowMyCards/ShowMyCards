import { BACKEND_URL } from '$lib';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, fetch }) => {
	const { id } = params;

	try {
		const response = await fetch(`${BACKEND_URL}/cards/${id}`);

		if (!response.ok) {
			throw error(response.status, 'Card not found');
		}

		const card = await response.json();
		return new Response(JSON.stringify(card), {
			headers: {
				'Content-Type': 'application/json'
			}
		});
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to fetch card');
	}
};
