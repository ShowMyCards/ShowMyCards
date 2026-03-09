import { BACKEND_URL } from '$lib';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/health`);

		if (!response.ok) {
			throw error(response.status, 'Health check failed');
		}

		const data = await response.json();
		return json(data);
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Health check failed');
	}
};
