import { BACKEND_URL } from '$lib';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/api/data/export`);

		if (!response.ok) {
			throw error(response.status, 'Failed to export data');
		}

		// Stream the response through, preserving headers for download
		return new Response(response.body, {
			status: response.status,
			headers: {
				'Content-Type': response.headers.get('Content-Type') || 'application/json',
				'Content-Disposition':
					response.headers.get('Content-Disposition') || 'attachment; filename="export.json"'
			}
		});
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to export data');
	}
};
