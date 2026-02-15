import { BACKEND_URL } from '$lib';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, fetch }) => {
	const { oracle_id } = params;

	try {
		const response = await fetch(`${BACKEND_URL}/inventory/by-oracle/${oracle_id}`);

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			throw error(response.status, errorData.error || 'Failed to fetch inventory');
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
		throw error(500, 'Failed to fetch inventory by oracle ID');
	}
};
