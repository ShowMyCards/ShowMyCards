import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { BACKEND_URL } from '$lib';

/**
 * API route that proxies expression validation requests to the backend
 * This allows client-side code to validate expressions without directly
 * accessing the backend URL (which may be localhost)
 */
export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const { expression } = await request.json();

		if (!expression || typeof expression !== 'string') {
			return json({ valid: false, error: 'Expression is required' }, { status: 400 });
		}

		const response = await fetch(`${BACKEND_URL}/sorting-rules/validate`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ expression })
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			return json(
				{ valid: false, error: errorData.error || 'Validation failed' },
				{ status: response.status }
			);
		}

		const data = await response.json();
		return json(data);
	} catch {
		return json({ valid: false, error: 'Validation request failed' }, { status: 500 });
	}
};
