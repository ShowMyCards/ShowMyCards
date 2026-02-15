import { BACKEND_URL } from '$lib';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url, fetch }) => {
	const query = url.searchParams.get('q');
	const page = url.searchParams.get('page') || '1';

	if (!query) {
		throw error(400, 'Query parameter is required');
	}

	try {
		const backendUrl = new URL(`${BACKEND_URL}/search`);
		backendUrl.searchParams.set('q', query);
		backendUrl.searchParams.set('page', page);

		const response = await fetch(backendUrl.toString());

		if (!response.ok) {
			throw error(response.status, 'Search failed');
		}

		const data = await response.json();
		return new Response(JSON.stringify(data), {
			headers: {
				'Content-Type': 'application/json'
			}
		});
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to search cards');
	}
};
