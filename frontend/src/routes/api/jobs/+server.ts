import { BACKEND_URL } from '$lib';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url, fetch }) => {
	try {
		// Forward query parameters to backend
		const backendUrl = new URL(`${BACKEND_URL}/api/jobs`);
		url.searchParams.forEach((value, key) => {
			backendUrl.searchParams.set(key, value);
		});

		const response = await fetch(backendUrl.toString());

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			throw error(response.status, errorData.error || 'Failed to fetch jobs');
		}

		const data = await response.json();
		return json(data);
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to fetch jobs');
	}
};
