import { BACKEND_URL } from '$lib';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, fetch }) => {
	const { code } = params;

	if (!code) {
		throw error(400, 'Set code is required');
	}

	try {
		const response = await fetch(`${BACKEND_URL}/sets/code/${code}/icon`);

		if (!response.ok) {
			throw error(response.status, 'Icon not found');
		}

		const svg = await response.text();
		return new Response(svg, {
			headers: {
				'Content-Type': 'image/svg+xml',
				'Cache-Control': 'public, max-age=86400'
			}
		});
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to fetch set icon');
	}
};
